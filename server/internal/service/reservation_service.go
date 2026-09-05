package service

import (
	"context"
	"strings"
	"time"

	apperrors "github.com/zhouwu97/key-cabinet/server/internal/platform/errors"
	"github.com/zhouwu97/key-cabinet/server/internal/repository"
)

type CreateReservationRequest struct {
	KeyID             string
	PickupWindowStart time.Time
	PickupWindowEnd   time.Time
	ExpectedReturnAt  time.Time
	ExpectedDuration  time.Duration
	Purpose           string
}

type ReservationService interface {
	CreateReservation(ctx context.Context, userID string, req CreateReservationRequest) (*repository.Reservation, error)
	ListUserReservations(ctx context.Context, userID, status string) ([]*repository.Reservation, error)
	GetReservation(ctx context.Context, id string) (*repository.Reservation, error)
	GetUserReservation(ctx context.Context, userID, id string) (*repository.Reservation, error)
	CancelReservation(ctx context.Context, userID, id string) (*repository.Reservation, error)
	CanReserveKey(ctx context.Context, keyID string, start, end time.Time) (bool, error)
	MarkReservationUsed(ctx context.Context, id string) error
}

type reservationService struct {
	reservationRepo repository.ReservationRepository
	keyRepo         repository.KeyRepository
	deviceRepo      repository.DeviceRepository
	borrowRepo      repository.BorrowRepository
}

func NewReservationService(
	reservationRepo repository.ReservationRepository,
	keyRepo repository.KeyRepository,
	deviceRepo repository.DeviceRepository,
	borrowRepo repository.BorrowRepository,
) ReservationService {
	return &reservationService{
		reservationRepo: reservationRepo,
		keyRepo:         keyRepo,
		deviceRepo:      deviceRepo,
		borrowRepo:      borrowRepo,
	}
}

func (s *reservationService) CreateReservation(ctx context.Context, userID string, req CreateReservationRequest) (*repository.Reservation, error) {
	userID = strings.TrimSpace(userID)
	req.KeyID = strings.TrimSpace(req.KeyID)
	if userID == "" || req.KeyID == "" {
		return nil, apperrors.New(apperrors.CodeInvalidInput, "user id and key id are required")
	}

	key, err := s.keyRepo.FindByID(ctx, req.KeyID)
	if err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query key")
	}
	if key == nil {
		return nil, apperrors.New(apperrors.CodeNotFound, "key not found")
	}
	if !key.Enabled || key.Status == "DISABLED" || key.Status == "MAINTENANCE" || key.Status == "BORROWED" || key.Status == "OVERDUE" {
		return nil, apperrors.New(apperrors.CodeInvalidState, "key is not available for reservation")
	}

	device, err := s.deviceRepo.FindByID(ctx, key.DeviceID)
	if err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query device")
	}
	if device == nil {
		return nil, apperrors.New(apperrors.CodeNotFound, "device not found")
	}
	if device.Status != "ONLINE" {
		return nil, apperrors.New(apperrors.CodeServiceUnavailable, "device is offline or unavailable")
	}

	activeBorrow, err := s.borrowRepo.FindActiveByKeyID(ctx, req.KeyID)
	if err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query active borrow")
	}
	if activeBorrow != nil {
		return nil, apperrors.New(apperrors.CodeConflict, "key is currently borrowed")
	}

	now := time.Now().UTC()
	start := req.PickupWindowStart
	if start.IsZero() {
		start = now
	}
	start = start.UTC()
	end := req.PickupWindowEnd
	if end.IsZero() {
		end = start.Add(30 * time.Minute)
	}
	end = end.UTC()
	returnAt := req.ExpectedReturnAt
	if returnAt.IsZero() {
		duration := req.ExpectedDuration
		if duration <= 0 {
			duration = 2 * time.Hour
		}
		returnAt = start.Add(duration)
	}
	returnAt = returnAt.UTC()
	if !start.Before(end) || !end.Before(returnAt) {
		return nil, apperrors.New(apperrors.CodeInvalidInput, "pickup window and expected return time are invalid")
	}

	userReservations, err := s.reservationRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query user reservations")
	}
	for _, existing := range userReservations {
		if existing.KeyID == req.KeyID && isReservationActive(existing.Status) && timeRangesOverlap(existing.PickupWindowStart, existing.ExpectedReturnAt, start, returnAt) {
			return nil, apperrors.New(apperrors.CodeConflict, "user already has a conflicting reservation")
		}
	}

	conflicts, err := s.reservationRepo.FindConflicts(ctx, req.KeyID, start, returnAt)
	if err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query reservation conflicts")
	}
	if len(conflicts) > 0 {
		return nil, apperrors.New(apperrors.CodeConflict, "reservation time conflicts with an existing reservation")
	}

	status := "ACTIVE"
	var approvedAt *time.Time
	if key.RequiresApproval {
		status = "PENDING"
	} else {
		approvedAt = &now
	}
	reservation := &repository.Reservation{
		ID:                "res_" + generateUUID()[:12],
		UserID:            userID,
		KeyID:             req.KeyID,
		Status:            status,
		PickupWindowStart: start,
		PickupWindowEnd:   end,
		ExpectedReturnAt:  returnAt,
		Purpose:           strings.TrimSpace(req.Purpose),
		ApprovedAt:        approvedAt,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.reservationRepo.Create(ctx, reservation); err != nil {
		if isReservationConstraintConflict(err) {
			return nil, apperrors.New(apperrors.CodeConflict, "reservation time conflicts with an existing reservation")
		}
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to create reservation")
	}
	return reservation, nil
}

func (s *reservationService) ListUserReservations(ctx context.Context, userID, status string) ([]*repository.Reservation, error) {
	userID = strings.TrimSpace(userID)
	status = strings.ToUpper(strings.TrimSpace(status))
	if userID == "" {
		return nil, apperrors.New(apperrors.CodeUnauthorized, "user identity is required")
	}
	if status != "" && !isReservationStatus(status) {
		return nil, apperrors.Newf(apperrors.CodeInvalidInput, "unsupported reservation status: %s", status)
	}
	reservations, err := s.reservationRepo.List(ctx, repository.ReservationListFilter{UserID: userID, Status: status})
	if err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query reservations")
	}
	return reservations, nil
}

func (s *reservationService) GetReservation(ctx context.Context, id string) (*repository.Reservation, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, apperrors.New(apperrors.CodeInvalidInput, "reservation id is required")
	}
	reservation, err := s.reservationRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query reservation")
	}
	if reservation == nil {
		return nil, apperrors.New(apperrors.CodeNotFound, "reservation not found")
	}
	return reservation, nil
}

func (s *reservationService) GetUserReservation(ctx context.Context, userID, id string) (*repository.Reservation, error) {
	reservation, err := s.GetReservation(ctx, id)
	if err != nil {
		return nil, err
	}
	if reservation.UserID != strings.TrimSpace(userID) {
		return nil, apperrors.New(apperrors.CodeNotFound, "reservation not found")
	}
	return reservation, nil
}

func (s *reservationService) CancelReservation(ctx context.Context, userID, id string) (*repository.Reservation, error) {
	reservation, err := s.GetUserReservation(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if !isReservationActive(reservation.Status) {
		return nil, apperrors.New(apperrors.CodeInvalidState, "reservation cannot be cancelled in its current state")
	}
	now := time.Now().UTC()
	reservation.Status = "CANCELLED"
	reservation.CancelledAt = &now
	reservation.UpdatedAt = now
	if err := s.reservationRepo.Update(ctx, reservation); err != nil {
		return nil, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to cancel reservation")
	}
	return reservation, nil
}

func (s *reservationService) CanReserveKey(ctx context.Context, keyID string, start, end time.Time) (bool, error) {
	key, err := s.keyRepo.FindByID(ctx, strings.TrimSpace(keyID))
	if err != nil {
		return false, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query key")
	}
	if key == nil {
		return false, apperrors.New(apperrors.CodeNotFound, "key not found")
	}
	if !key.Enabled || key.Status == "DISABLED" || key.Status == "MAINTENANCE" || key.Status == "BORROWED" || key.Status == "OVERDUE" {
		return false, nil
	}
	device, err := s.deviceRepo.FindByID(ctx, key.DeviceID)
	if err != nil {
		return false, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query device")
	}
	if device == nil || device.Status != "ONLINE" {
		return false, nil
	}
	activeBorrow, err := s.borrowRepo.FindActiveByKeyID(ctx, key.ID)
	if err != nil {
		return false, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query active borrow")
	}
	if activeBorrow != nil {
		return false, nil
	}
	now := time.Now().UTC()
	if start.IsZero() {
		start = now
	}
	if end.IsZero() {
		end = start.Add(2 * time.Hour)
	}
	if !start.Before(end) {
		return false, apperrors.New(apperrors.CodeInvalidInput, "reservation time range is invalid")
	}
	conflicts, err := s.reservationRepo.FindConflicts(ctx, key.ID, start.UTC(), end.UTC())
	if err != nil {
		return false, apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to query reservation conflicts")
	}
	return len(conflicts) == 0, nil
}

func (s *reservationService) MarkReservationUsed(ctx context.Context, id string) error {
	reservation, err := s.GetReservation(ctx, id)
	if err != nil {
		return err
	}
	if !isReservationActive(reservation.Status) {
		return apperrors.New(apperrors.CodeInvalidState, "reservation is not active")
	}
	now := time.Now().UTC()
	reservation.Status = "USED"
	reservation.UsedAt = &now
	reservation.UpdatedAt = now
	if err := s.reservationRepo.Update(ctx, reservation); err != nil {
		return apperrors.WrapWithCode(err, apperrors.CodeInternalError, "failed to mark reservation used")
	}
	return nil
}

func isReservationActive(status string) bool {
	switch status {
	case "PENDING", "APPROVED", "ACTIVE":
		return true
	default:
		return false
	}
}

func isReservationStatus(status string) bool {
	switch status {
	case "PENDING", "APPROVED", "ACTIVE", "USED", "REJECTED", "CANCELLED", "EXPIRED":
		return true
	default:
		return false
	}
}

func timeRangesOverlap(startA, endA, startB, endB time.Time) bool {
	return startA.Before(endB) && endA.After(startB)
}

func isReservationConstraintConflict(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "reservations_no_overlap") || strings.Contains(message, "exclusion constraint")
}
