package repository

import (
	"context"
	"time"
)

type Reservation struct {
	ID                string     `gorm:"primaryKey;column:id" json:"id"`
	UserID            string     `gorm:"column:user_id" json:"userId"`
	KeyID             string     `gorm:"column:key_id" json:"keyId"`
	KeyName           string     `gorm:"column:key_name;->" json:"keyName,omitempty"`
	RoomNo            string     `gorm:"column:room_no;->" json:"roomNo,omitempty"`
	DeviceID          string     `gorm:"column:device_id;->" json:"deviceId,omitempty"`
	DeviceName        string     `gorm:"column:device_name;->" json:"deviceName,omitempty"`
	Status            string     `gorm:"column:status" json:"status"`
	PickupWindowStart time.Time  `gorm:"column:pickup_window_start" json:"pickupWindowStart"`
	PickupWindowEnd   time.Time  `gorm:"column:pickup_window_end" json:"pickupWindowEnd"`
	ExpectedReturnAt  time.Time  `gorm:"column:expected_return_at" json:"expectedReturnAt"`
	Purpose           string     `gorm:"column:purpose" json:"purpose,omitempty"`
	ApprovedAt        *time.Time `gorm:"column:approved_at" json:"approvedAt,omitempty"`
	UsedAt            *time.Time `gorm:"column:used_at" json:"usedAt,omitempty"`
	CancelledAt       *time.Time `gorm:"column:cancelled_at" json:"cancelledAt,omitempty"`
	CreatedAt         time.Time  `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt         time.Time  `gorm:"column:updated_at" json:"updatedAt"`
}

func (Reservation) TableName() string {
	return "reservations"
}

type ReservationListFilter struct {
	UserID string
	Status string
}

type ReservationRepository interface {
	Create(ctx context.Context, r *Reservation) error
	FindByID(ctx context.Context, id string) (*Reservation, error)
	FindByUserID(ctx context.Context, userID string) ([]*Reservation, error)
	List(ctx context.Context, filter ReservationListFilter) ([]*Reservation, error)
	FindConflicts(ctx context.Context, keyID string, startTime, endTime time.Time) ([]*Reservation, error)
	Update(ctx context.Context, r *Reservation) error
}
