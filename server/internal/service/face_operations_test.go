package service

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zhouwu97/key-cabinet/server/internal/infrastructure/device"
	"github.com/zhouwu97/key-cabinet/server/internal/platform/jwt"
	"github.com/zhouwu97/key-cabinet/server/internal/repository"
)

func TestFaceAuthScopesTasksAndRejectsInvalidProof(t *testing.T) {
	now := time.Now().UTC()
	user := &repository.User{ID: "u1", StudentNo: "20230001", Status: "ACTIVE", IdentityVerified: true}
	keys := &fakeKeyRepository{keys: []*repository.Key{
		{ID: "k1", DeviceID: "CAB001", RoomNo: "101", Name: "实验室"},
		{ID: "k2", DeviceID: "CAB002"},
	}}
	reservations := &fakeCabinetReservationRepository{}
	for _, entry := range []struct {
		id, key, status string
		start, end      time.Time
	}{
		{"ready", "k1", "APPROVED", now.Add(-time.Minute), now.Add(time.Minute)},
		{"other", "k2", "ACTIVE", now.Add(-time.Minute), now.Add(time.Minute)},
		{"future", "k1", "APPROVED", now.Add(time.Minute), now.Add(time.Hour)},
		{"expired", "k1", "ACTIVE", now.Add(-time.Hour), now.Add(-time.Minute)},
		{"pending", "k1", "PENDING", now.Add(-time.Minute), now.Add(time.Minute)},
	} {
		reservations.reservations = append(reservations.reservations, &repository.Reservation{ID: entry.id, UserID: user.ID,
			KeyID: entry.key, Status: entry.status, PickupWindowStart: entry.start, PickupWindowEnd: entry.end})
	}
	borrows := &fakeCabinetBorrowRepository{borrows: []*repository.BorrowRecord{
		{ID: "returnable", UserID: user.ID, DeviceID: "CAB001", Status: "BORROWED"},
		{ID: "returning", UserID: user.ID, DeviceID: "CAB001", Status: "RETURNING"},
		{ID: "other", UserID: user.ID, DeviceID: "CAB002", Status: "BORROWED"},
	}}
	svc := NewCabinetService(&fakeCabinetUserRepository{user: user}, keys, &fakeSlotRepository{},
		&fakeCabinetDeviceRepository{}, reservations, borrows, nil, nil, jwt.NewTokenService("local-test-secret", 300))
	params := FaceAuthParams{DeviceID: "CAB001", StudentNo: user.StudentNo, Confidence: 0.95, LivenessPassed: true}
	result, err := svc.FaceAuth(context.Background(), params)
	require.NoError(t, err)
	require.Len(t, result.ActiveReservations, 1)
	require.Equal(t, "ready", result.ActiveReservations[0].ID)
	require.Equal(t, "101", result.ActiveReservations[0].RoomNo)
	require.Len(t, result.ActiveBorrows, 1)
	require.Equal(t, "returnable", result.ActiveBorrows[0].ID)
	require.Equal(t, 300, result.ExpiresIn)
	for _, confidence := range []float64{math.NaN(), math.Inf(1), -1, 0.79, 1.01} {
		params.Confidence = confidence
		_, err = svc.FaceAuth(context.Background(), params)
		require.Error(t, err)
	}
	params.Confidence = 0.95
	reservations.err = errors.New("database unavailable")
	_, err = svc.FaceAuth(context.Background(), params)
	require.Error(t, err)
	reservations.err = nil
	user.IdentityVerified = false
	_, err = svc.FaceAuth(context.Background(), params)
	require.Error(t, err)
}

type faceReservationService struct {
	ReservationService
	reservation *repository.Reservation
}

func (s faceReservationService) GetUserReservation(_ context.Context, user, id string) (*repository.Reservation, error) {
	if s.reservation.ID != id || s.reservation.UserID != user {
		return nil, errors.New("reservation not owned")
	}
	return s.reservation, nil
}

type faceBorrowService struct {
	BorrowService
	borrow *repository.BorrowRecord
}

func (s faceBorrowService) GetActiveBorrowByKey(context.Context, string) (*repository.BorrowRecord, error) {
	return nil, nil
}
func (s faceBorrowService) GetUserBorrowRecord(_ context.Context, user, id string) (*repository.BorrowRecord, error) {
	if s.borrow.ID != id || s.borrow.UserID != user {
		return nil, errors.New("borrow not owned")
	}
	return s.borrow, nil
}

type faceGateway struct {
	device.DeviceGateway
	commands []device.DeviceCommand
}

func (g *faceGateway) GetDeviceStatus(_ context.Context, id string) (*device.DeviceStatus, error) {
	return &device.DeviceStatus{DeviceID: id, Online: true}, nil
}
func (g *faceGateway) StartPickup(_ context.Context, command device.DeviceCommand) error {
	g.commands = append(g.commands, command)
	return nil
}
func (g *faceGateway) StartReturn(_ context.Context, command device.DeviceCommand) error {
	g.commands = append(g.commands, command)
	return nil
}

type faceOperationRepo struct {
	fakeCabinetOperationRepository
	borrow *repository.BorrowRecord
}

func (r *faceOperationRepo) PreparePickup(ctx context.Context, borrow *repository.BorrowRecord, op *repository.DeviceOperation) error {
	r.borrow = borrow
	return r.fakeCabinetOperationRepository.PreparePickup(ctx, borrow, op)
}

func TestCabinetPickupAndReturnUseBoundTransactions(t *testing.T) {
	now := time.Now().UTC()
	reservation := &repository.Reservation{ID: "r1", UserID: "u1", KeyID: "k1", Status: "APPROVED",
		PickupWindowStart: now.Add(-time.Minute), PickupWindowEnd: now.Add(time.Minute), ExpectedReturnAt: now.Add(7 * time.Hour), Purpose: "实验调试"}
	keyID := "k1"
	key := &repository.Key{ID: keyID, DeviceID: "CAB001", SlotID: "s1", Enabled: true, Status: "RESERVED"}
	slot := &repository.Slot{ID: "s1", DeviceID: "CAB001", KeyID: &keyID, Presence: "PRESENT", Enabled: true}
	record := &repository.BorrowRecord{ID: "b1", UserID: "u1", KeyID: keyID, DeviceID: "CAB001", SlotID: "s1", Status: "BORROWED"}
	ops := &faceOperationRepo{fakeCabinetOperationRepository: fakeCabinetOperationRepository{operations: make(map[string]*repository.DeviceOperation)}}
	gateway := &faceGateway{}
	svc := &operationService{operationRepo: ops, reservationSvc: faceReservationService{reservation: reservation},
		borrowSvc: faceBorrowService{borrow: record}, keyRepo: &fakeKeyRepository{keys: []*repository.Key{key}},
		slotRepo: &fakeSlotRepository{slot: slot}, deviceRepo: &fakeCabinetDeviceRepository{}, deviceGateway: gateway,
		userRepo: &fakeCabinetUserRepository{user: &repository.User{ID: "u1", Status: "ACTIVE", IdentityVerified: true}}}
	ctx := context.Background()
	_, err := svc.StartCabinetPickup(ctx, "u1", "r1", "CAB002", "request1")
	require.Error(t, err)
	require.Empty(t, gateway.commands)
	require.Empty(t, ops.operations)
	reservation.PickupWindowStart = now.Add(time.Minute)
	_, err = svc.StartCabinetPickup(ctx, "u1", "r1", "CAB001", "request1")
	require.Error(t, err)
	reservation.PickupWindowStart = now.Add(-time.Minute)
	op, err := svc.StartCabinetPickup(ctx, "u1", "r1", "CAB001", "request1")
	require.NoError(t, err)
	require.Equal(t, "EXECUTING", op.Status)
	require.Equal(t, reservation.ExpectedReturnAt, ops.borrow.ExpectedReturnAt)
	require.Equal(t, reservation.ID, ops.borrow.ReservationID)
	require.Equal(t, reservation.Purpose, ops.borrow.Purpose)
	_, err = svc.StartCabinetPickup(ctx, "u1", "r1", "CAB001", "request1")
	require.NoError(t, err)
	require.Len(t, gateway.commands, 1)
	_, err = svc.StartCabinetPickup(ctx, "u1", "r1", "CAB002", "request1")
	require.Error(t, err)
	_, err = svc.StartReturn(ctx, "u1", "b1", "CAB001", "request1")
	require.Error(t, err)
	_, err = svc.StartReturn(ctx, "u1", "b1", "CAB002", "request2")
	require.Error(t, err)
	_, err = svc.StartReturn(ctx, "u1", "b1", "CAB001", "request2")
	require.NoError(t, err)
	require.Len(t, gateway.commands, 2)
	require.Equal(t, "RETURN", gateway.commands[1].Type)
	_, err = svc.StartReturn(ctx, "u1", "b1", "CAB002", "request2")
	require.Error(t, err)
	record.Status = "RETURNING"
	_, err = svc.StartReturn(ctx, "u1", "b1", "CAB001", "request3")
	require.Error(t, err)
}
