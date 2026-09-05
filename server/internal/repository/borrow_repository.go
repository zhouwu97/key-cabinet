package repository

import (
	"context"
	"time"
)

type BorrowRecord struct {
	ID               string     `gorm:"primaryKey;column:id" json:"id"`
	ReservationID    string     `gorm:"column:reservation_id" json:"reservationId,omitempty"`
	UserID           string     `gorm:"column:user_id" json:"userId"`
	KeyID            string     `gorm:"column:key_id" json:"keyId"`
	KeyName          string     `gorm:"column:key_name;->" json:"keyName,omitempty"`
	RoomNo           string     `gorm:"column:room_no;->" json:"roomNo,omitempty"`
	DeviceID         string     `gorm:"column:device_id" json:"deviceId"`
	SlotID           string     `gorm:"column:slot_id" json:"slotId"`
	Status           string     `gorm:"column:status" json:"status"`
	BorrowedAt       *time.Time `gorm:"column:borrowed_at" json:"borrowedAt,omitempty"`
	ExpectedReturnAt time.Time  `gorm:"column:expected_return_at" json:"expectedReturnAt"`
	OverdueAt        *time.Time `gorm:"column:overdue_at" json:"overdueAt,omitempty"`
	ReturnedAt       *time.Time `gorm:"column:returned_at" json:"returnedAt,omitempty"`
	RFIDVerified     bool       `gorm:"column:rfid_verified" json:"rfidVerified"`
	Purpose          string     `gorm:"column:purpose" json:"purpose,omitempty"`
	Notes            string     `gorm:"column:notes" json:"notes,omitempty"`
	CreatedAt        time.Time  `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt        time.Time  `gorm:"column:updated_at" json:"updatedAt"`
}

func (BorrowRecord) TableName() string {
	return "borrow_records"
}

type BorrowListFilter struct {
	UserID string
	KeyID  string
	Status string
}

type BorrowRepository interface {
	Create(ctx context.Context, record *BorrowRecord) error
	FindByID(ctx context.Context, id string) (*BorrowRecord, error)
	FindByReservationID(ctx context.Context, reservationID string) (*BorrowRecord, error)
	FindByUserID(ctx context.Context, userID string) ([]*BorrowRecord, error)
	List(ctx context.Context, filter BorrowListFilter) ([]*BorrowRecord, error)
	FindActiveByKeyID(ctx context.Context, keyID string) (*BorrowRecord, error)
	Update(ctx context.Context, record *BorrowRecord) error
	MarkOverdue(ctx context.Context, now time.Time) error
}
