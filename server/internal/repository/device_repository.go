package repository

import (
	"context"
	"time"
)

type Device struct {
	ID              string     `gorm:"primaryKey;column:id" json:"id"`
	Name            string     `gorm:"column:name" json:"name"`
	Location        string     `gorm:"column:location" json:"location,omitempty"`
	Capacity        int        `gorm:"column:capacity" json:"capacity"`
	Status          string     `gorm:"column:status" json:"status"`
	IPAddress       string     `gorm:"column:ip_address" json:"ipAddress,omitempty"`
	LastHeartbeatAt *time.Time `gorm:"column:last_heartbeat_at" json:"lastHeartbeatAt"`
	CreatedAt       time.Time  `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt       time.Time  `gorm:"column:updated_at" json:"updatedAt"`
}

func (Device) TableName() string {
	return "devices"
}

type DeviceRepository interface {
	FindByID(ctx context.Context, id string) (*Device, error)
	FindAll(ctx context.Context) ([]*Device, error)
}
