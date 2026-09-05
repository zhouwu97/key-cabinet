package repository

import (
	"context"
	"time"
)

type DeviceOperation struct {
	ID             string            `gorm:"primaryKey;column:id" json:"id"`
	RequestID      string            `gorm:"column:request_id" json:"requestId"`
	UserID         string            `gorm:"column:user_id" json:"userId"`
	ReservationID  string            `gorm:"column:reservation_id" json:"reservationId,omitempty"`
	BorrowRecordID string            `gorm:"column:borrow_record_id" json:"borrowRecordId,omitempty"`
	DeviceID       string            `gorm:"column:device_id" json:"deviceId"`
	SlotID         string            `gorm:"column:slot_id" json:"slotId,omitempty"`
	KeyID          string            `gorm:"column:key_id" json:"keyId,omitempty"`
	Action         string            `gorm:"column:operation_type" json:"action"`
	Status         string            `gorm:"column:status" json:"status"`
	CreatedAt      time.Time         `gorm:"column:created_at" json:"createdAt"`
	StartedAt      *time.Time        `gorm:"column:initiated_at" json:"startedAt,omitempty"`
	SentAt         *time.Time        `gorm:"column:sent_at" json:"sentAt,omitempty"`
	AckAt          *time.Time        `gorm:"column:ack_at" json:"ackAt,omitempty"`
	FinishedAt     *time.Time        `gorm:"column:completed_at" json:"finishedAt,omitempty"`
	ErrorCode      string            `gorm:"column:error_code" json:"errorCode,omitempty"`
	ErrorMessage   string            `gorm:"column:error_message" json:"errorMessage,omitempty"`
	Events         []*OperationEvent `gorm:"-" json:"events,omitempty"`
}

func (DeviceOperation) TableName() string {
	return "device_operations"
}

type OperationEvent struct {
	ID          string                 `gorm:"primaryKey;column:id" json:"eventId"`
	OperationID string                 `gorm:"column:operation_id" json:"operationId"`
	Seq         int                    `gorm:"column:seq" json:"seq"`
	Type        string                 `gorm:"column:event_type" json:"type"`
	Data        map[string]interface{} `gorm:"column:event_data;serializer:json" json:"data,omitempty"`
	OccurredAt  time.Time              `gorm:"column:occurred_at" json:"timestamp"`
}

func (OperationEvent) TableName() string {
	return "operation_events"
}

type OperationRepository interface {
	Create(ctx context.Context, op *DeviceOperation) error
	FindByID(ctx context.Context, id string) (*DeviceOperation, error)
	FindByRequestID(ctx context.Context, requestID string) (*DeviceOperation, error)
	FindByReservationID(ctx context.Context, reservationID string) ([]*DeviceOperation, error)
	FindActiveByReservation(ctx context.Context, reservationID string) (*DeviceOperation, error)
	FindActiveByDeviceID(ctx context.Context, deviceID string) (*DeviceOperation, error)
	FindActiveByKeyID(ctx context.Context, keyID string) (*DeviceOperation, error)
	FindActiveByUserID(ctx context.Context, userID string) (*DeviceOperation, error)
	Update(ctx context.Context, op *DeviceOperation) error
	CreateEvent(ctx context.Context, event *OperationEvent) error
	CompletePickup(ctx context.Context, operationID string, now time.Time) error
	CompleteReturn(ctx context.Context, operationID string, now time.Time) error
	Fail(ctx context.Context, operationID, errorCode, errorMessage string, now time.Time) error
	Cancel(ctx context.Context, operationID string, now time.Time) error
}
