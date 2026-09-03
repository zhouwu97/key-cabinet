package repository

import (
	"context"
	"time"
)

type DeviceOperation struct {
	ID             string
	ReservationID  string
	BorrowRecordID string
	DeviceID       string
	SlotID         string
	KeyID          string
	OperationType  string
	Status         string
	InitiatedAt    time.Time
	CompletedAt    *time.Time
	ErrorMessage   string
}

type OperationRepository interface {
	Create(ctx context.Context, op *DeviceOperation) error
	FindByID(ctx context.Context, id string) (*DeviceOperation, error)
	FindByReservationID(ctx context.Context, reservationID string) ([]*DeviceOperation, error)
	FindActiveByReservation(ctx context.Context, reservationID string) (*DeviceOperation, error)
	Update(ctx context.Context, op *DeviceOperation) error
}
