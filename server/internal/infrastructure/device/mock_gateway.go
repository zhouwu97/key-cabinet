package device

import (
	"context"
	"log"
	"sync"
	"time"
)

type MockDeviceGateway struct {
	handler    DeviceEventHandler
	mu         sync.Mutex
	operations map[string]context.CancelFunc
}

func NewMockDeviceGateway() *MockDeviceGateway {
	return &MockDeviceGateway{operations: make(map[string]context.CancelFunc)}
}

func (g *MockDeviceGateway) RegisterEventHandler(handler DeviceEventHandler) {
	g.handler = handler
}

func (g *MockDeviceGateway) StartPickup(ctx context.Context, cmd DeviceCommand) error {
	log.Printf("[MockDeviceGateway] StartPickup: OperationID=%s, DeviceID=%s, SlotID=%s",
		cmd.OperationID, cmd.DeviceID, cmd.SlotID)

	operationCtx, cancel := context.WithCancel(context.Background())
	g.storeOperation(cmd.OperationID, cancel)
	go func() {
		defer g.deleteOperation(cmd.OperationID)
		select {
		case <-time.After(100 * time.Millisecond):
		case <-operationCtx.Done():
			return
		}
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

	operationCtx, cancel := context.WithCancel(context.Background())
	g.storeOperation(cmd.OperationID, cancel)
	go func() {
		defer g.deleteOperation(cmd.OperationID)
		select {
		case <-time.After(100 * time.Millisecond):
		case <-operationCtx.Done():
			return
		}
		expectedTag := cmd.ExpectedRFID
		if expectedTag == "" {
			expectedTag = "RFID-MOCK-" + cmd.SlotID
		}
		rfidEvent := DeviceEvent{
			EventID:     "mock_evt_rfid_" + cmd.OperationID,
			OperationID: cmd.OperationID,
			DeviceID:    cmd.DeviceID,
			EventType:   "RFID_CONFIRMED",
			Timestamp:   time.Now(),
			Data: map[string]interface{}{
				"isMatch":     true,
				"scannedRfid": expectedTag,
			},
		}
		if g.handler != nil {
			if err := g.handler.OnDeviceEvent(context.Background(), rfidEvent); err != nil {
				log.Printf("[MockDeviceGateway] OnDeviceEvent RFID_CONFIRMED failed: %v", err)
			}
		}

		select {
		case <-time.After(50 * time.Millisecond):
		case <-operationCtx.Done():
			return
		}

		event := DeviceEvent{
			EventID:     "mock_evt_ret_" + cmd.OperationID,
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

func (g *MockDeviceGateway) AbortOperation(_ context.Context, cmd DeviceCommand) error {
	g.mu.Lock()
	cancel, ok := g.operations[cmd.OperationID]
	if ok {
		delete(g.operations, cmd.OperationID)
	}
	g.mu.Unlock()
	if !ok {
		return ErrOperationNotRunning
	}
	cancel()
	return nil
}

func (g *MockDeviceGateway) storeOperation(operationID string, cancel context.CancelFunc) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.operations[operationID] = cancel
}

func (g *MockDeviceGateway) deleteOperation(operationID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.operations, operationID)
}

func (g *MockDeviceGateway) GetDeviceStatus(ctx context.Context, deviceID string) (*DeviceStatus, error) {
	return &DeviceStatus{
		DeviceID: deviceID,
		Online:   true,
		LastSeen: time.Now(),
	}, nil
}
