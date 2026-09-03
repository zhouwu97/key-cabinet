package device

import (
	"context"
	"log"
	"time"
)

type MockDeviceGateway struct {
	handler DeviceEventHandler
}

func NewMockDeviceGateway() *MockDeviceGateway {
	return &MockDeviceGateway{}
}

func (g *MockDeviceGateway) RegisterEventHandler(handler DeviceEventHandler) {
	g.handler = handler
}

func (g *MockDeviceGateway) StartPickup(ctx context.Context, cmd DeviceCommand) error {
	log.Printf("[MockDeviceGateway] StartPickup: OperationID=%s, DeviceID=%s, SlotID=%s",
		cmd.OperationID, cmd.DeviceID, cmd.SlotID)

	// 模拟设备处理延迟
	go func() {
		time.Sleep(100 * time.Millisecond)
		event := DeviceEvent{
			OperationID: cmd.OperationID,
			DeviceID:    cmd.DeviceID,
			EventType:   "PICKUP_SUCCESS",
			Timestamp:   time.Now(),
		}
		if g.handler != nil {
			if err := g.handler.OnPickupSuccess(context.Background(), event); err != nil {
				log.Printf("[MockDeviceGateway] OnPickupSuccess failed: %v", err)
			}
		}
	}()

	return nil
}

func (g *MockDeviceGateway) StartReturn(ctx context.Context, cmd DeviceCommand) error {
	log.Printf("[MockDeviceGateway] StartReturn: OperationID=%s, DeviceID=%s, SlotID=%s",
		cmd.OperationID, cmd.DeviceID, cmd.SlotID)

	// 模拟设备处理延迟
	go func() {
		time.Sleep(100 * time.Millisecond)
		event := DeviceEvent{
			OperationID: cmd.OperationID,
			DeviceID:    cmd.DeviceID,
			EventType:   "RETURN_SUCCESS",
			Timestamp:   time.Now(),
		}
		if g.handler != nil {
			if err := g.handler.OnReturnSuccess(context.Background(), event); err != nil {
				log.Printf("[MockDeviceGateway] OnReturnSuccess failed: %v", err)
			}
		}
	}()

	return nil
}

func (g *MockDeviceGateway) GetDeviceStatus(ctx context.Context, deviceID string) (*DeviceStatus, error) {
	return &DeviceStatus{
		DeviceID: deviceID,
		Online:   true,
		LastSeen: time.Now(),
	}, nil
}
