package repository

import (
	"context"
	"time"
)

type Reservation struct {
	ID        string
	UserID    string
	KeyID     string
	Status    string
	StartTime time.Time
	EndTime   time.Time
	Purpose   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ReservationRepository interface {
	Create(ctx context.Context, r *Reservation) error
	FindByID(ctx context.Context, id string) (*Reservation, error)
	FindByUserID(ctx context.Context, userID string) ([]*Reservation, error)
	FindConflicts(ctx context.Context, keyID string, startTime, endTime time.Time) ([]*Reservation, error)
	Update(ctx context.Context, r *Reservation) error
}
