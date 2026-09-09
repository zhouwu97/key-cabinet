package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/zhouwu97/key-cabinet/server/internal/infrastructure/device"
	apperrors "github.com/zhouwu97/key-cabinet/server/internal/platform/errors"
	"github.com/zhouwu97/key-cabinet/server/internal/repository"
)

type OperationService interface {
	StartPickup(ctx context.Context, userID, reservationID, requestID string) (*repository.DeviceOperation, error)
	StartReturn(ctx context.Context, userID, borrowRecordID, deviceID, requestID string) (*repository.DeviceOperation, error)
	GetOperation(ctx context.Context, userID, operationID string) (*repository.DeviceOperation, error)
	GetActiveOperation(ctx context.Context, userID string) (*repository.DeviceOperation, error)
	CancelOperation(ctx context.Context, userID, operationID string) error
	ExpireTimedOutOperations(ctx context.Context, now time.Time, timeout time.Duration) (int, error)
}

type operationService struct {
	operationRepo  repository.OperationRepository
	reservationSvc ReservationService
	borrowSvc      BorrowService
	keyRepo        repository.KeyRepository
	deviceRepo     repository.DeviceRepository
	slotRepo       repository.SlotRepository
	deviceGateway  device.DeviceGateway
	userRepo       repository.UserRepository
}

func NewOperationService(
	operationRepo repository.OperationRepository,
	reservationSvc ReservationService,
	borrowSvc BorrowService,
	keyRepo repository.KeyRepository,
	deviceRepo repository.DeviceRepository,
	slotRepo repository.SlotRepository,
	deviceGateway device.DeviceGateway,
	userRepo repository.UserRepository,
) OperationService {
	service := &operationService{
		operationRepo:  operationRepo,
		reservationSvc: reservationSvc,
		borrowSvc:      borrowSvc,
		keyRepo:        keyRepo,
		deviceRepo:     deviceRepo,
		slotRepo:       slotRepo,
		deviceGateway:  deviceGateway,
		userRepo:       userRepo,
	}
	deviceGateway.RegisterEventHandler(service)
	return service
}

func (s *operationService) StartPickup(ctx context.Context, userID, reservationID, requestID string) (*repository.DeviceOperation, error) {
	userID = strings.TrimSpace(userID)
	reservationID = strings.TrimSpace(reservationID)
	requestID = strings.TrimSpace(requestID)
	if userID == "" || reservationID == "" || requestID == "" {
		return nil, apperrors.New(apperrors.CodeInvalidInput, "reservation id and client request id are required")
	}
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query operation user")
	}
	if user == nil || user.Status != "ACTIVE" || !user.IdentityVerified {
		return nil, apperrors.New(apperrors.CodeForbidden, "verified active identity is required for pickup")
	}
	if existing, err := s.findIdempotent(ctx, userID, requestID); existing != nil || err != nil {
		return existing, err
	}

	reservation, err := s.reservationSvc.GetUserReservation(ctx, userID, reservationID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if reservation.Status == "APPROVED" && now.Before(reservation.PickupWindowStart) {
		return nil, apperrors.New(apperrors.CodeTooEarly, "pickup window has not started")
	}
	if (reservation.Status == "ACTIVE" || reservation.Status == "APPROVED") && now.After(reservation.PickupWindowEnd) {
		return nil, apperrors.New(apperrors.CodeExpired, "reservation pickup window has expired")
	}
	if reservation.Status != "ACTIVE" && reservation.Status != "APPROVED" {
		return nil, apperrors.New(apperrors.CodeInvalidState, "reservation is not active")
	}

	key, slot, err := s.loadKeyAndSlot(ctx, reservation.KeyID)
	if err != nil {
		return nil, err
	}
	if key.DeviceID == "" || slot.DeviceID != key.DeviceID {
		return nil, apperrors.New(apperrors.CodeInvalidState, "key and slot are not bound to the same device")
	}
	if key.Status != "AVAILABLE" && key.Status != "RESERVED" {
		return nil, apperrors.New(apperrors.CodeInvalidState, "key is not available for pickup")
	}
	if !slot.Enabled || normalizeSlotPresence(slot.Presence) != "PRESENT" {
		return nil, apperrors.New(apperrors.CodeInvalidState, "key is not present in its slot")
	}
	if err := s.ensureDeviceReady(ctx, key.DeviceID); err != nil {
		return nil, err
	}
	if err := s.ensureNoActiveOperation(ctx, key.DeviceID, key.ID, reservationID); err != nil {
		return nil, err
	}

	if activeBorrow, lookupErr := s.borrowSvc.GetActiveBorrowByKey(ctx, key.ID); lookupErr != nil {
		return nil, lookupErr
	} else if activeBorrow != nil {
		return nil, apperrors.New(apperrors.CodeConflict, "key already has an active borrow record")
	}
	borrow := &repository.BorrowRecord{
		ID:               "bor_" + generateUUID()[:12],
		ReservationID:    reservation.ID,
		UserID:           userID,
		KeyID:            key.ID,
		DeviceID:         key.DeviceID,
		SlotID:           slot.ID,
		Status:           "BORROWING",
		ExpectedReturnAt: reservation.ExpectedReturnAt.UTC(),
		Purpose:          strings.TrimSpace(reservation.Purpose),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	operation := &repository.DeviceOperation{
		ID:             "op_" + generateUUID()[:12],
		RequestID:      requestID,
		UserID:         userID,
		ReservationID:  reservation.ID,
		BorrowRecordID: borrow.ID,
		DeviceID:       key.DeviceID,
		SlotID:         slot.ID,
		KeyID:          key.ID,
		Action:         "PICKUP",
		Status:         "AUTHORIZED",
		CreatedAt:      now,
		StartedAt:      &now,
	}
	if err := s.operationRepo.PreparePickup(ctx, borrow, operation); err != nil {
		return s.resolvePrepareError(ctx, operation, err)
	}
	return s.dispatchPrepared(ctx, operation, func() error {
		return s.deviceGateway.StartPickup(ctx, device.DeviceCommand{
			OperationID:    operation.ID,
			DeviceID:       operation.DeviceID,
			SlotID:         operation.SlotID,
			SlotNo:         slot.SlotNo,
			KeyID:          key.ID,
			Type:           operation.Action,
			TimeoutSeconds: 120,
		})
	})
}

func (s *operationService) StartReturn(ctx context.Context, userID, borrowRecordID, deviceID, requestID string) (*repository.DeviceOperation, error) {
	userID = strings.TrimSpace(userID)
	borrowRecordID = strings.TrimSpace(borrowRecordID)
	deviceID = strings.TrimSpace(deviceID)
	requestID = strings.TrimSpace(requestID)
	if userID == "" || borrowRecordID == "" || requestID == "" {
		return nil, apperrors.New(apperrors.CodeInvalidInput, "borrow record id and client request id are required")
	}
	if existing, err := s.findIdempotent(ctx, userID, requestID); existing != nil || err != nil {
		return existing, err
	}

	borrow, err := s.borrowSvc.GetUserBorrowRecord(ctx, userID, borrowRecordID)
	if err != nil {
		return nil, err
	}
	if deviceID == "" {
		deviceID = borrow.DeviceID
	}
	if deviceID != borrow.DeviceID {
		return nil, apperrors.New(apperrors.CodeInvalidState, "return device does not match borrow record")
	}
	key, slot, err := s.loadKeyAndSlot(ctx, borrow.KeyID)
	if err != nil {
		return nil, err
	}
	if slot.ID != borrow.SlotID {
		return nil, apperrors.New(apperrors.CodeInvalidState, "return slot does not match borrow record")
	}
	if err := s.ensureDeviceReady(ctx, deviceID); err != nil {
		return nil, err
	}
	if err := s.ensureNoActiveOperation(ctx, deviceID, key.ID, ""); err != nil {
		return nil, err
	}
	if borrow.Status != "BORROWED" {
		return nil, apperrors.New(apperrors.CodeInvalidState, "borrow record cannot be returned in its current state")
	}

	now := time.Now().UTC()
	operation := &repository.DeviceOperation{
		ID:             "op_" + generateUUID()[:12],
		RequestID:      requestID,
		UserID:         userID,
		BorrowRecordID: borrow.ID,
		DeviceID:       deviceID,
		SlotID:         slot.ID,
		KeyID:          key.ID,
		Action:         "RETURN",
		Status:         "AUTHORIZED",
		CreatedAt:      now,
		StartedAt:      &now,
	}
	if err := s.operationRepo.PrepareReturn(ctx, borrow.ID, userID, operation); err != nil {
		return s.resolvePrepareError(ctx, operation, err)
	}
	result, err := s.dispatchPrepared(ctx, operation, func() error {
		return s.deviceGateway.StartReturn(ctx, device.DeviceCommand{
			OperationID:    operation.ID,
			DeviceID:       operation.DeviceID,
			SlotID:         operation.SlotID,
			SlotNo:         slot.SlotNo,
			KeyID:          key.ID,
			ExpectedRFID:   key.RFIDTag,
			Type:           operation.Action,
			TimeoutSeconds: 120,
		})
	})
	return result, err
}

func (s *operationService) GetOperation(ctx context.Context, userID, operationID string) (*repository.DeviceOperation, error) {
	operation, err := s.operationRepo.FindByID(ctx, strings.TrimSpace(operationID))
	if err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query operation")
	}
	if operation == nil || operation.UserID != strings.TrimSpace(userID) {
		return nil, apperrors.New(apperrors.CodeNotFound, "operation not found")
	}
	return operation, nil
}

func (s *operationService) GetActiveOperation(ctx context.Context, userID string) (*repository.DeviceOperation, error) {
	operation, err := s.operationRepo.FindActiveByUserID(ctx, strings.TrimSpace(userID))
	if err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query active operation")
	}
	if operation == nil {
		return nil, nil
	}
	return operation, nil
}

func (s *operationService) CancelOperation(ctx context.Context, userID, operationID string) error {
	operation, err := s.GetOperation(ctx, userID, operationID)
	if err != nil {
		return err
	}
	if isTerminalOperation(operation.Status) {
		return apperrors.New(apperrors.CodeInvalidState, "operation is already finished")
	}
	if operation.Status == "SENT" || operation.Status == "EXECUTING" {
		err := s.deviceGateway.AbortOperation(ctx, device.DeviceCommand{
			OperationID: operation.ID,
			DeviceID:    operation.DeviceID,
			SlotID:      operation.SlotID,
			Type:        operation.Action,
		})
		if err != nil && !errors.Is(err, device.ErrOperationNotRunning) {
			return apperrors.WrapWithCode(err, apperrors.CodeConflict, "device is executing and cannot be safely cancelled")
		}
		if errors.Is(err, device.ErrOperationNotRunning) {
			latest, lookupErr := s.operationRepo.FindByID(ctx, operation.ID)
			if lookupErr != nil {
				return apperrors.WrapWithCode(lookupErr, apperrors.CodeInternalError, "failed to confirm operation status")
			}
			if latest == nil || isTerminalOperation(latest.Status) {
				return apperrors.New(apperrors.CodeInvalidState, "operation has already finished")
			}
			return apperrors.New(apperrors.CodeConflict, "device operation can no longer be aborted safely")
		}
	}
	if err := s.operationRepo.Cancel(ctx, operation.ID, time.Now().UTC()); err != nil {
		if errors.Is(err, repository.ErrOperationTerminal) {
			return apperrors.New(apperrors.CodeInvalidState, "operation has already finished")
		}
		return apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to cancel operation")
	}
	return nil
}

func (s *operationService) ExpireTimedOutOperations(ctx context.Context, now time.Time, timeout time.Duration) (int, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if timeout <= 0 {
		return 0, apperrors.New(apperrors.CodeInvalidInput, "operation timeout must be positive")
	}
	operations, err := s.operationRepo.FindExpired(ctx, now.Add(-timeout))
	if err != nil {
		return 0, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query timed out operations")
	}
	expired := 0
	for _, operation := range operations {
		if operation == nil {
			continue
		}
		_ = s.deviceGateway.AbortOperation(ctx, device.DeviceCommand{
			OperationID: operation.ID,
			DeviceID:    operation.DeviceID,
			SlotID:      operation.SlotID,
			Type:        operation.Action,
		})
		if err := s.operationRepo.Timeout(ctx, operation.ID, now); err != nil {
			return expired, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to time out operation")
		}
		expired++
	}
	return expired, nil
}

func (s *operationService) OnPickupSuccess(ctx context.Context, event device.DeviceEvent) error {
	return s.operationRepo.CompletePickup(ctx, event.OperationID, event.Timestamp)
}

func (s *operationService) OnDeviceEvent(ctx context.Context, event device.DeviceEvent) error {
	operation, err := s.operationRepo.FindByID(ctx, strings.TrimSpace(event.OperationID))
	if err != nil {
		return apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query device event operation")
	}
	if operation == nil {
		return apperrors.New(apperrors.CodeNotFound, "device event operation not found")
	}
	if operation.DeviceID != strings.TrimSpace(event.DeviceID) {
		return apperrors.New(apperrors.CodeForbidden, "device event does not belong to operation device")
	}
	eventType := strings.ToUpper(strings.TrimSpace(event.EventType))
	switch eventType {
	case "SUCCESS", "PICKUP_SUCCESS", "RETURN_SUCCESS":
		if operation.Action == "PICKUP" {
			return s.operationRepo.CompletePickup(ctx, operation.ID, event.Timestamp)
		}
		return s.operationRepo.CompleteReturn(ctx, operation.ID, event.Timestamp)
	case "FAILED", "ERROR", "PICKUP_FAILED", "RETURN_FAILED":
		return s.operationRepo.Fail(ctx, operation.ID, event.ErrorCode, event.ErrorMessage, event.Timestamp)
	default:
		return s.operationRepo.AppendEventIfActive(ctx, &repository.OperationEvent{
			ID:          event.EventID,
			OperationID: operation.ID,
			Type:        eventType,
			Data:        event.Data,
			OccurredAt:  event.Timestamp,
		})
	}
}

func (s *operationService) OnPickupFailed(ctx context.Context, event device.DeviceEvent) error {
	return s.operationRepo.Fail(ctx, event.OperationID, event.ErrorCode, event.ErrorMessage, event.Timestamp)
}

func (s *operationService) OnReturnSuccess(ctx context.Context, event device.DeviceEvent) error {
	return s.operationRepo.CompleteReturn(ctx, event.OperationID, event.Timestamp)
}

func (s *operationService) OnReturnFailed(ctx context.Context, event device.DeviceEvent) error {
	return s.operationRepo.Fail(ctx, event.OperationID, event.ErrorCode, event.ErrorMessage, event.Timestamp)
}

func (s *operationService) findIdempotent(ctx context.Context, userID, requestID string) (*repository.DeviceOperation, error) {
	existing, err := s.operationRepo.FindByRequestID(ctx, requestID)
	if err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query operation request")
	}
	if existing == nil {
		return nil, nil
	}
	if existing.UserID != userID {
		return nil, apperrors.New(apperrors.CodeForbidden, "operation request belongs to another user")
	}
	return existing, nil
}

func (s *operationService) loadKeyAndSlot(ctx context.Context, keyID string) (*repository.Key, *repository.Slot, error) {
	key, err := s.keyRepo.FindByID(ctx, keyID)
	if err != nil {
		return nil, nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query key")
	}
	if key == nil {
		return nil, nil, apperrors.New(apperrors.CodeNotFound, "key not found")
	}
	var slot *repository.Slot
	if key.SlotID != "" {
		slot, err = s.slotRepo.FindByID(ctx, key.SlotID)
	}
	if err != nil {
		return nil, nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query slot")
	}
	if slot == nil {
		slot, err = s.slotRepo.FindByKeyID(ctx, key.ID)
		if err != nil {
			return nil, nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query slot by key")
		}
	}
	if slot == nil {
		return nil, nil, apperrors.New(apperrors.CodeNotFound, "slot not found")
	}
	return key, slot, nil
}

func (s *operationService) ensureDeviceReady(ctx context.Context, deviceID string) error {
	stored, err := s.deviceRepo.FindByID(ctx, deviceID)
	if err != nil {
		return apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query device")
	}
	if stored == nil {
		return apperrors.New(apperrors.CodeNotFound, "device not found")
	}
	if stored.Status == "BUSY" {
		return apperrors.New(apperrors.CodeConflict, "device is busy")
	}
	if stored.Status != "ONLINE" {
		return apperrors.New(apperrors.CodeServiceUnavailable, "device is offline or unavailable")
	}
	status, err := s.deviceGateway.GetDeviceStatus(ctx, deviceID)
	if err != nil {
		return apperrors.WrapWithCode(err, apperrors.CodeServiceUnavailable, "device gateway is unavailable")
	}
	if status == nil || !status.Online {
		return apperrors.New(apperrors.CodeServiceUnavailable, "device is offline")
	}
	return nil
}

func (s *operationService) ensureNoActiveOperation(ctx context.Context, deviceID, keyID, reservationID string) error {
	if reservationID != "" {
		active, err := s.operationRepo.FindActiveByReservation(ctx, reservationID)
		if err != nil {
			return apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query active reservation operation")
		}
		if active != nil {
			return apperrors.New(apperrors.CodeConflict, "reservation already has an active operation")
		}
	}
	active, err := s.operationRepo.FindActiveByDeviceID(ctx, deviceID)
	if err != nil {
		return apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query active device operation")
	}
	if active != nil {
		return apperrors.New(apperrors.CodeConflict, "device is busy")
	}
	active, err = s.operationRepo.FindActiveByKeyID(ctx, keyID)
	if err != nil {
		return apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query active key operation")
	}
	if active != nil {
		return apperrors.New(apperrors.CodeConflict, "key already has an active operation")
	}
	return nil
}

func (s *operationService) resolvePrepareError(ctx context.Context, operation *repository.DeviceOperation, err error) (*repository.DeviceOperation, error) {
	if err != nil {
		if existing, lookupErr := s.operationRepo.FindByRequestID(ctx, operation.RequestID); lookupErr == nil && existing != nil {
			if existing.UserID != operation.UserID {
				return nil, apperrors.New(apperrors.CodeForbidden, "operation request belongs to another user")
			}
			return existing, nil
		}
		if strings.Contains(strings.ToLower(err.Error()), "active_device") || strings.Contains(strings.ToLower(err.Error()), "active_key") {
			return nil, apperrors.New(apperrors.CodeConflict, "device or key already has an active operation")
		}
		if errors.Is(err, repository.ErrOperationInvalidState) {
			return nil, apperrors.New(apperrors.CodeInvalidState, "borrow record state changed before operation could start")
		}
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to prepare device operation")
	}
	return nil, nil
}

func (s *operationService) dispatchPrepared(ctx context.Context, operation *repository.DeviceOperation, dispatch func() error) (*repository.DeviceOperation, error) {
	now := time.Now().UTC()
	if err := s.operationRepo.BeginExecution(ctx, operation.ID, now); err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to begin device operation")
	}
	operation.Status = "EXECUTING"
	operation.SentAt = &now
	if err := dispatch(); err != nil {
		_ = s.operationRepo.Fail(ctx, operation.ID, "DEVICE_COMMAND_FAILED", err.Error(), time.Now().UTC())
		return nil, apperrors.WrapWithCode(err, apperrors.CodeServiceUnavailable, "failed to send device command")
	}
	return operation, nil
}

func isTerminalOperation(status string) bool {
	return status == "SUCCESS" || status == "FAILED" || status == "TIMEOUT" || status == "CANCELLED"
}
