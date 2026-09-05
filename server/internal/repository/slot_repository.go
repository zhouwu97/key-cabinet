package repository

import (
	"context"
	"time"
)

type Slot struct {
	ID            string    `gorm:"primaryKey;column:id" json:"id"`
	DeviceID      string    `gorm:"column:device_id" json:"deviceId"`
	SlotNo        int       `gorm:"column:slot_number" json:"slotNo"`
	KeyID         *string   `gorm:"column:key_id;->" json:"keyId,omitempty"`
	Presence      string    `gorm:"column:presence" json:"presence"`
	Enabled       bool      `gorm:"column:enabled" json:"enabled"`
	LastUpdatedAt time.Time `gorm:"column:updated_at" json:"lastUpdatedAt"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"createdAt,omitempty"`
}

func (Slot) TableName() string {
	return "slots"
}

type SlotRepository interface {
	FindByID(ctx context.Context, id string) (*Slot, error)
	FindByKeyID(ctx context.Context, keyID string) (*Slot, error)
	FindByDeviceID(ctx context.Context, deviceID string) ([]*Slot, error)
}
