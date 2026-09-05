package repository

import (
	"context"
	"time"
)

type Key struct {
	ID               string    `gorm:"primaryKey;column:id" json:"id"`
	Name             string    `gorm:"column:name" json:"name"`
	KeyNumber        string    `gorm:"column:key_number" json:"keyNumber,omitempty"`
	RFIDTag          string    `gorm:"column:rfid_tag" json:"rfidTag,omitempty"`
	DeviceID         string    `gorm:"column:device_id" json:"deviceId"`
	DeviceName       string    `gorm:"column:device_name;->" json:"deviceName,omitempty"`
	SlotID           string    `gorm:"column:slot_id" json:"slotId,omitempty"`
	Category         string    `gorm:"column:category" json:"category,omitempty"`
	RoomNo           string    `gorm:"column:room_no" json:"roomNo"`
	Building         string    `gorm:"column:building" json:"building,omitempty"`
	Description      string    `gorm:"column:description" json:"description,omitempty"`
	Status           string    `gorm:"column:status" json:"status"`
	Enabled          bool      `gorm:"column:enabled" json:"enabled"`
	RequiresApproval bool      `gorm:"column:requires_approval" json:"requiresApproval"`
	CreatedAt        time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt        time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

func (Key) TableName() string {
	return "keys"
}

type KeyListFilter struct {
	Keyword  string
	DeviceID string
	Status   string
	Enabled  *bool
}

type KeyRepository interface {
	FindByID(ctx context.Context, id string) (*Key, error)
	FindAll(ctx context.Context) ([]*Key, error)
	FindByStatus(ctx context.Context, status string) ([]*Key, error)
	List(ctx context.Context, filter KeyListFilter) ([]*Key, error)
	Update(ctx context.Context, key *Key) error
}
