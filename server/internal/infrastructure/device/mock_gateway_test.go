package device

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type countingEventHandler struct {
	pickupSuccess atomic.Int32
}

func (*countingEventHandler) OnDeviceEvent(context.Context, DeviceEvent) error { return nil }

func (h *countingEventHandler) OnPickupSuccess(context.Context, DeviceEvent) error {
	h.pickupSuccess.Add(1)
	return nil
}

func (*countingEventHandler) OnPickupFailed(context.Context, DeviceEvent) error  { return nil }
func (*countingEventHandler) OnReturnSuccess(context.Context, DeviceEvent) error { return nil }
func (*countingEventHandler) OnReturnFailed(context.Context, DeviceEvent) error  { return nil }

func TestMockGatewayAbortPreventsLateSuccess(t *testing.T) {
	gateway := NewMockDeviceGateway()
	handler := &countingEventHandler{}
	gateway.RegisterEventHandler(handler)
	command := DeviceCommand{OperationID: "op_cancel", DeviceID: "dev_1", SlotID: "slot_1", Type: "PICKUP"}

	if err := gateway.StartPickup(context.Background(), command); err != nil {
		t.Fatalf("StartPickup() error = %v", err)
	}
	if err := gateway.AbortOperation(context.Background(), command); err != nil {
		t.Fatalf("AbortOperation() error = %v", err)
	}
	time.Sleep(150 * time.Millisecond)

	if got := handler.pickupSuccess.Load(); got != 0 {
		t.Fatalf("pickup success callbacks = %d, want 0", got)
	}
}
