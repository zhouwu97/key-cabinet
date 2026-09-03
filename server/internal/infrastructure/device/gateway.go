package device

import (
	"context"
	"time"
)

type DeviceCommand struct {
	OperationID string
	DeviceID    string
	SlotID      string
	Type        string // PICKUP / RETURN
}

type DeviceEvent struct {
	OperationID  string
	DeviceID     string
	EventType    string // PICKUP_SUCCESS / PICKUP_FAILED / RETURN_SUCCESS / RETURN_FAILED
	Timestamp    time.Time
	ErrorCode    string
	ErrorMessage string
}

type DeviceStatus struct {
	DeviceID string
	Online   bool
	LastSeen time.Time
}

type DeviceGateway interface {
	StartPickup(ctx context.Context, cmd DeviceCommand) error
	StartReturn(ctx context.Context, cmd DeviceCommand) error
	GetDeviceStatus(ctx context.Context, deviceID string) (*DeviceStatus, error)
	RegisterEventHandler(handler DeviceEventHandler)
}

type DeviceEventHandler interface {
	OnPickupSuccess(ctx context.Context, event DeviceEvent) error
	OnPickupFailed(ctx context.Context, event DeviceEvent) error
	OnReturnSuccess(ctx context.Context, event DeviceEvent) error
	OnReturnFailed(ctx context.Context, event DeviceEvent) error
}
