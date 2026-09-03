package repository

import (
	"context"
	"time"
)

type BorrowRecord struct {
	ID               string
	ReservationID    string
	UserID           string
	KeyID            string
	Status           string
	BorrowedAt       time.Time
	ExpectedReturnAt time.Time
	ReturnedAt       *time.Time
	RFIDVerified     bool
	Notes            string
}

type BorrowRepository interface {
	Create(ctx context.Context, record *BorrowRecord) error
	FindByID(ctx context.Context, id string) (*BorrowRecord, error)
	FindByReservationID(ctx context.Context, reservationID string) (*BorrowRecord, error)
	FindByUserID(ctx context.Context, userID string) ([]*BorrowRecord, error)
	Update(ctx context.Context, record *BorrowRecord) error
}
