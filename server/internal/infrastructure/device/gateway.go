package device

import (
	"context"
	"errors"
	"time"
)

var ErrOperationNotRunning = errors.New("device operation is not running")

type DeviceCommand struct {
	OperationID    string
	DeviceID       string
	SlotID         string
	SlotNo         int
	KeyID          string
	ExpectedRFID   string
	Type           string // PICKUP / RETURN
	TimeoutSeconds int
}

type DeviceEvent struct {
	EventID      string
	OperationID  string
	DeviceID     string
	EventType    string
	Timestamp    time.Time
	ErrorCode    string
	ErrorMessage string
	Data         map[string]interface{}
}

type DeviceStatus struct {
	DeviceID string
	Online   bool
	LastSeen time.Time
}

type DeviceGateway interface {
	StartPickup(ctx context.Context, cmd DeviceCommand) error
	StartReturn(ctx context.Context, cmd DeviceCommand) error
	AbortOperation(ctx context.Context, cmd DeviceCommand) error
	GetDeviceStatus(ctx context.Context, deviceID string) (*DeviceStatus, error)
	RegisterEventHandler(handler DeviceEventHandler)
}

type DeviceEventHandler interface {
	OnDeviceEvent(ctx context.Context, event DeviceEvent) error
	OnPickupSuccess(ctx context.Context, event DeviceEvent) error
	OnPickupFailed(ctx context.Context, event DeviceEvent) error
	OnReturnSuccess(ctx context.Context, event DeviceEvent) error
	OnReturnFailed(ctx context.Context, event DeviceEvent) error
}

type DeviceStatusSink interface {
	UpdateRuntimeStatus(ctx context.Context, deviceID, status string, lastSeen time.Time) error
}
