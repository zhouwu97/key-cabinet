package repository

import (
	"context"
	"time"
)

type DeviceAlert struct {
	ID           string     `json:"id"`
	DeviceID     string     `json:"deviceId"`
	SlotID       *string    `json:"slotId,omitempty"`
	SlotNo       *int       `json:"slotNo,omitempty"`
	Type         string     `json:"type"`
	ExpectedRFID *string    `json:"expectedRfid,omitempty"`
	ActualRFID   *string    `json:"actualRfid,omitempty"`
	Message      string     `json:"message"`
	Status       string     `json:"status"` // OPEN, RESOLVED
	DetectedAt   time.Time  `json:"detectedAt"`
	ResolvedAt   *time.Time `json:"resolvedAt,omitempty"`
}

type AlertRepository interface {
	Create(ctx context.Context, alert *DeviceAlert) error
	ResolveBySlotAndType(ctx context.Context, deviceID string, slotNo int, alertType string, resolvedAt time.Time) error
	ResolveAllBySlot(ctx context.Context, deviceID string, slotNo int, resolvedAt time.Time) error
	FindOpenBySlotAndType(ctx context.Context, deviceID string, slotNo int, alertType string) (*DeviceAlert, error)
	ListOpen(ctx context.Context, deviceID string) ([]*DeviceAlert, error)
}
