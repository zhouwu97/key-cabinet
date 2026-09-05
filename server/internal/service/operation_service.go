package service

import (
	"context"
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
}

type operationService struct {
	operationRepo  repository.OperationRepository
	reservationSvc ReservationService
	borrowSvc      BorrowService
	keyRepo        repository.KeyRepository
	deviceRepo     repository.DeviceRepository
	slotRepo       repository.SlotRepository
	deviceGateway  device.DeviceGateway
}

func NewOperationService(
	operationRepo repository.OperationRepository,
	reservationSvc ReservationService,
	borrowSvc BorrowService,
	keyRepo repository.KeyRepository,
	deviceRepo repository.DeviceRepository,
	slotRepo repository.SlotRepository,
	deviceGateway device.DeviceGateway,
) OperationService {
	service := &operationService{
		operationRepo:  operationRepo,
		reservationSvc: reservationSvc,
		borrowSvc:      borrowSvc,
		keyRepo:        keyRepo,
		deviceRepo:     deviceRepo,
		slotRepo:       slotRepo,
		deviceGateway:  deviceGateway,
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
	if reservation.Status == "ACTIVE" && now.After(reservation.PickupWindowEnd) {
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

	borrow, err := s.borrowSvc.CreateBorrowing(ctx, userID, key.ID, key.DeviceID, slot.ID, reservation.ID, reservation.Purpose, reservation.ExpectedReturnAt)
	if err != nil {
		return nil, err
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
		Status:         "CREATED",
		CreatedAt:      now,
		StartedAt:      &now,
	}
	return s.createAndDispatch(ctx, operation, func() error {
		return s.deviceGateway.StartPickup(ctx, device.DeviceCommand{
			OperationID: operation.ID,
			DeviceID:    operation.DeviceID,
			SlotID:      operation.SlotID,
			Type:        operation.Action,
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
	if _, err := s.borrowSvc.BeginReturn(ctx, userID, borrow.ID); err != nil {
		return nil, err
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
		Status:         "CREATED",
		CreatedAt:      now,
		StartedAt:      &now,
	}
	result, err := s.createAndDispatch(ctx, operation, func() error {
		return s.deviceGateway.StartReturn(ctx, device.DeviceCommand{
			OperationID: operation.ID,
			DeviceID:    operation.DeviceID,
			SlotID:      operation.SlotID,
			Type:        operation.Action,
		})
	})
	if err != nil {
		_ = s.borrowSvc.MarkOperationFailed(ctx, borrow.ID, err.Error())
	}
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
	if err := s.operationRepo.Cancel(ctx, operation.ID, time.Now().UTC()); err != nil {
		return apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to cancel operation")
	}
	return nil
}

func (s *operationService) OnPickupSuccess(ctx context.Context, event device.DeviceEvent) error {
	return s.operationRepo.CompletePickup(ctx, event.OperationID, event.Timestamp)
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

func (s *operationService) createAndDispatch(ctx context.Context, operation *repository.DeviceOperation, dispatch func() error) (*repository.DeviceOperation, error) {
	if err := s.operationRepo.Create(ctx, operation); err != nil {
		if existing, lookupErr := s.operationRepo.FindByRequestID(ctx, operation.RequestID); lookupErr == nil && existing != nil {
			if existing.UserID != operation.UserID {
				return nil, apperrors.New(apperrors.CodeForbidden, "operation request belongs to another user")
			}
			return existing, nil
		}
		if strings.Contains(strings.ToLower(err.Error()), "active_device") || strings.Contains(strings.ToLower(err.Error()), "active_key") {
			return nil, apperrors.New(apperrors.CodeConflict, "device or key already has an active operation")
		}
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to create device operation")
	}
	if err := s.operationRepo.CreateEvent(ctx, &repository.OperationEvent{OperationID: operation.ID, Type: "RECEIVED", OccurredAt: *operation.StartedAt}); err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to record operation event")
	}
	operation.Status = "AUTHORIZED"
	if err := s.operationRepo.Update(ctx, operation); err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to authorize operation")
	}
	if err := s.operationRepo.CreateEvent(ctx, &repository.OperationEvent{OperationID: operation.ID, Type: "AUTH_CONFIRMED", OccurredAt: time.Now().UTC()}); err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to record authorization event")
	}
	if err := dispatch(); err != nil {
		_ = s.operationRepo.Fail(ctx, operation.ID, "DEVICE_COMMAND_FAILED", err.Error(), time.Now().UTC())
		return nil, apperrors.WrapWithCode(err, apperrors.CodeServiceUnavailable, "failed to send device command")
	}
	now := time.Now().UTC()
	operation.Status = "EXECUTING"
	operation.SentAt = &now
	operation.AckAt = nil
	if err := s.operationRepo.Update(ctx, operation); err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to update operation status")
	}
	if err := s.operationRepo.CreateEvent(ctx, &repository.OperationEvent{OperationID: operation.ID, Type: "POSITIONING", OccurredAt: now}); err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to record device event")
	}
	return operation, nil
}

func isTerminalOperation(status string) bool {
	return status == "SUCCESS" || status == "FAILED" || status == "TIMEOUT" || status == "CANCELLED"
}
