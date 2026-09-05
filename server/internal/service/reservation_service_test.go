package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zhouwu97/key-cabinet/server/internal/repository"
)

type fakeReservationRepository struct {
	reservation *repository.Reservation
}

func (r *fakeReservationRepository) Create(_ context.Context, reservation *repository.Reservation) error {
	r.reservation = reservation
	return nil
}

func (r *fakeReservationRepository) FindByID(_ context.Context, id string) (*repository.Reservation, error) {
	if r.reservation != nil && r.reservation.ID == id {
		return r.reservation, nil
	}
	return nil, nil
}

func (r *fakeReservationRepository) FindByUserID(_ context.Context, _ string) ([]*repository.Reservation, error) {
	if r.reservation == nil {
		return nil, nil
	}
	return []*repository.Reservation{r.reservation}, nil
}

func (r *fakeReservationRepository) List(_ context.Context, _ repository.ReservationListFilter) ([]*repository.Reservation, error) {
	return r.FindByUserID(context.Background(), "")
}

func (r *fakeReservationRepository) FindConflicts(_ context.Context, _ string, _, _ time.Time) ([]*repository.Reservation, error) {
	return nil, nil
}

func (r *fakeReservationRepository) Update(_ context.Context, reservation *repository.Reservation) error {
	r.reservation = reservation
	return nil
}

type fakeReservationKeyRepository struct {
	key *repository.Key
}

func (r *fakeReservationKeyRepository) FindByID(_ context.Context, id string) (*repository.Key, error) {
	if r.key != nil && r.key.ID == id {
		return r.key, nil
	}
	return nil, nil
}

func (r *fakeReservationKeyRepository) FindAll(_ context.Context) ([]*repository.Key, error) {
	return []*repository.Key{r.key}, nil
}

func (r *fakeReservationKeyRepository) FindByStatus(_ context.Context, _ string) ([]*repository.Key, error) {
	return []*repository.Key{r.key}, nil
}

func (r *fakeReservationKeyRepository) List(_ context.Context, _ repository.KeyListFilter) ([]*repository.Key, error) {
	return []*repository.Key{r.key}, nil
}

func (r *fakeReservationKeyRepository) Update(_ context.Context, _ *repository.Key) error {
	return nil
}

type fakeReservationDeviceRepository struct {
	device *repository.Device
}

func (r *fakeReservationDeviceRepository) FindByID(_ context.Context, id string) (*repository.Device, error) {
	if r.device != nil && r.device.ID == id {
		return r.device, nil
	}
	return nil, nil
}

func (r *fakeReservationDeviceRepository) FindAll(_ context.Context) ([]*repository.Device, error) {
	return []*repository.Device{r.device}, nil
}

type fakeReservationBorrowRepository struct{}

func (fakeReservationBorrowRepository) FindActiveByKeyID(_ context.Context, _ string) (*repository.BorrowRecord, error) {
	return nil, nil
}

func (fakeReservationBorrowRepository) Create(_ context.Context, _ *repository.BorrowRecord) error {
	return nil
}

func (fakeReservationBorrowRepository) FindByID(_ context.Context, _ string) (*repository.BorrowRecord, error) {
	return nil, nil
}

func (fakeReservationBorrowRepository) FindByReservationID(_ context.Context, _ string) (*repository.BorrowRecord, error) {
	return nil, nil
}

func (fakeReservationBorrowRepository) FindByUserID(_ context.Context, _ string) ([]*repository.BorrowRecord, error) {
	return nil, nil
}

func (fakeReservationBorrowRepository) List(_ context.Context, _ repository.BorrowListFilter) ([]*repository.BorrowRecord, error) {
	return nil, nil
}

func (fakeReservationBorrowRepository) Update(_ context.Context, _ *repository.BorrowRecord) error {
	return nil
}

func (fakeReservationBorrowRepository) MarkOverdue(_ context.Context, _ time.Time) error {
	return nil
}

func TestReservationServiceCreatesActiveReservation(t *testing.T) {
	keyRepo := &fakeReservationKeyRepository{key: &repository.Key{
		ID: "KEY-1", DeviceID: "CAB-1", Status: "AVAILABLE", Enabled: true,
	}}
	deviceRepo := &fakeReservationDeviceRepository{device: &repository.Device{ID: "CAB-1", Status: "ONLINE"}}
	reservationRepo := &fakeReservationRepository{}
	reservationService := NewReservationService(
		reservationRepo,
		keyRepo,
		deviceRepo,
		fakeReservationBorrowRepository{},
	)

	start := time.Now().UTC().Add(5 * time.Minute)
	reservation, err := reservationService.CreateReservation(context.Background(), "USER-1", CreateReservationRequest{
		KeyID:             "KEY-1",
		PickupWindowStart: start,
		PickupWindowEnd:   start.Add(30 * time.Minute),
		ExpectedReturnAt:  start.Add(2 * time.Hour),
		Purpose:           "实验测试",
	})

	require.NoError(t, err)
	require.Equal(t, "USER-1", reservation.UserID)
	require.Equal(t, "ACTIVE", reservation.Status)
	require.Equal(t, "实验测试", reservation.Purpose)
}

func TestReservationServiceRejectsUnavailableKey(t *testing.T) {
	keyRepo := &fakeReservationKeyRepository{key: &repository.Key{
		ID: "KEY-1", DeviceID: "CAB-1", Status: "MAINTENANCE", Enabled: false,
	}}
	deviceRepo := &fakeReservationDeviceRepository{device: &repository.Device{ID: "CAB-1", Status: "ONLINE"}}
	reservationService := NewReservationService(
		&fakeReservationRepository{},
		keyRepo,
		deviceRepo,
		fakeReservationBorrowRepository{},
	)

	_, err := reservationService.CreateReservation(context.Background(), "USER-1", CreateReservationRequest{KeyID: "KEY-1"})

	require.Error(t, err)
}
