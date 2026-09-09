package device

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mqttTestHandler struct {
	mu     sync.Mutex
	events []DeviceEvent
	err    error
}

func (h *mqttTestHandler) OnDeviceEvent(_ context.Context, event DeviceEvent) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.events = append(h.events, event)
	return h.err
}

func (h *mqttTestHandler) OnPickupSuccess(context.Context, DeviceEvent) error { return nil }
func (h *mqttTestHandler) OnPickupFailed(context.Context, DeviceEvent) error  { return nil }
func (h *mqttTestHandler) OnReturnSuccess(context.Context, DeviceEvent) error { return nil }
func (h *mqttTestHandler) OnReturnFailed(context.Context, DeviceEvent) error  { return nil }

type mqttTestStatusSink struct {
	deviceID string
	status   string
	lastSeen time.Time
}

type mqttTestMessage struct {
	topic    string
	payload  []byte
	ackCount int
}

func (m *mqttTestMessage) Duplicate() bool   { return false }
func (m *mqttTestMessage) Qos() byte         { return 1 }
func (m *mqttTestMessage) Retained() bool    { return false }
func (m *mqttTestMessage) Topic() string     { return m.topic }
func (m *mqttTestMessage) MessageID() uint16 { return 1 }
func (m *mqttTestMessage) Payload() []byte   { return m.payload }
func (m *mqttTestMessage) Ack()              { m.ackCount++ }

func (s *mqttTestStatusSink) UpdateRuntimeStatus(_ context.Context, deviceID, status string, lastSeen time.Time) error {
	s.deviceID = deviceID
	s.status = status
	s.lastSeen = lastSeen
	return nil
}

func newMQTTGatewayForTest(handler DeviceEventHandler, sink DeviceStatusSink) *MQTTDeviceGateway {
	return &MQTTDeviceGateway{
		config: MQTTGatewayConfig{
			TopicPrefix:      "kcab",
			CommandTimeout:   time.Second,
			HeartbeatTimeout: time.Minute,
		},
		statusSink:   sink,
		handler:      handler,
		deviceStates: make(map[string]mqttDeviceState),
		abortWaiters: make(map[string]chan abortResult),
		seenMessages: make(map[string]time.Time),
	}
}

func TestMQTTGatewayHandlesAndDeduplicatesProgress(t *testing.T) {
	handler := &mqttTestHandler{}
	gateway := newMQTTGatewayForTest(handler, nil)
	payload := []byte(`{
		"msgId":"evt-1","timestamp":1788350403000,"deviceId":"CAB001",
		"data":{"operationId":"op-1","stage":"DOOR_OPEN","slotNo":3}
	}`)

	require.NoError(t, gateway.handleIncoming("kcab/cab/CAB001/event/operation_progress", payload))
	require.NoError(t, gateway.handleIncoming("kcab/cab/CAB001/event/operation_progress", payload))

	require.Len(t, handler.events, 1)
	assert.Equal(t, "CAB001_evt-1", handler.events[0].EventID)
	assert.Equal(t, "op-1", handler.events[0].OperationID)
	assert.Equal(t, "DOOR_OPEN", handler.events[0].EventType)
}

func TestMQTTGatewayMapsReplySuccessToCommandAcknowledgement(t *testing.T) {
	handler := &mqttTestHandler{}
	gateway := newMQTTGatewayForTest(handler, nil)
	payload := []byte(`{
		"msgId":"evt-ack","replyMsgId":"cmd_pickup_op-1","status":"SUCCESS","deviceId":"CAB001",
		"data":{"operationId":"op-1"}
	}`)

	require.NoError(t, gateway.handleIncoming("kcab/cab/CAB001/event/operation_progress", payload))
	require.Len(t, handler.events, 1)
	assert.Equal(t, "COMMAND_ACK", handler.events[0].EventType)
}

func TestMQTTGatewayBuildsActionSpecificCommandPayloads(t *testing.T) {
	gateway := newMQTTGatewayForTest(&mqttTestHandler{}, nil)
	gateway.config.CommandTimeout = 5 * time.Second
	command := DeviceCommand{
		OperationID: "op-1", DeviceID: "CAB001", SlotID: "slot-3", SlotNo: 3,
		KeyID: "KEY103", ExpectedRFID: "E200001A9903", TimeoutSeconds: 60,
	}
	now := time.UnixMilli(1788350400000)

	pickup := gateway.commandEnvelope("pickup", command, now)
	assert.Equal(t, "PICKUP", pickup.Action)
	assert.Equal(t, 3, pickup.Data.SlotNo)
	assert.Zero(t, pickup.Data.TargetSlotNo)
	assert.Empty(t, pickup.Data.ExpectedRFID)

	returned := gateway.commandEnvelope("return", command, now)
	assert.Equal(t, "RETURN", returned.Action)
	assert.Zero(t, returned.Data.SlotNo)
	assert.Equal(t, 3, returned.Data.TargetSlotNo)
	assert.Equal(t, "E200001A9903", returned.Data.ExpectedRFID)

	abort := gateway.commandEnvelope("abort", command, now)
	assert.Equal(t, "ABORT", abort.Action)
	assert.Equal(t, 5, abort.Data.TimeoutSeconds)
	assert.Zero(t, abort.Data.SlotNo)
	assert.Empty(t, abort.Data.ExpectedRFID)
}

func TestMQTTGatewayRetriesMessageAfterHandlerFailure(t *testing.T) {
	handler := &mqttTestHandler{err: errors.New("database unavailable")}
	gateway := newMQTTGatewayForTest(handler, nil)
	payload := []byte(`{
		"msgId":"evt-retry","deviceId":"CAB001",
		"data":{"operationId":"op-1","stage":"COMMAND_ACK"}
	}`)

	err := gateway.handleIncoming("kcab/cab/CAB001/event/operation_progress", payload)
	require.ErrorContains(t, err, "database unavailable")
	handler.err = nil
	require.NoError(t, gateway.handleIncoming("kcab/cab/CAB001/event/operation_progress", payload))

	require.Len(t, handler.events, 2)
}

func TestMQTTGatewayAcknowledgesOnlyAfterSuccessfulBusinessHandling(t *testing.T) {
	handler := &mqttTestHandler{err: errors.New("database unavailable")}
	gateway := newMQTTGatewayForTest(handler, nil)
	message := &mqttTestMessage{
		topic:   "kcab/cab/CAB001/event/operation_progress",
		payload: []byte(`{"msgId":"evt-retry","deviceId":"CAB001","data":{"operationId":"op-1","stage":"COMMAND_ACK"}}`),
	}

	gateway.processMessage(message)
	assert.Zero(t, message.ackCount)
	handler.err = nil
	gateway.processMessage(message)
	assert.Equal(t, 1, message.ackCount)
}

func TestMQTTGatewayQueuesEventsUntilHandlerRegistration(t *testing.T) {
	gateway := newMQTTGatewayForTest(nil, nil)
	message := &mqttTestMessage{
		topic:   "kcab/cab/CAB001/event/operation_progress",
		payload: []byte(`{"msgId":"evt-startup","deviceId":"CAB001","data":{"operationId":"op-1","stage":"COMMAND_ACK"}}`),
	}

	gateway.onMessage(nil, message)
	assert.Zero(t, message.ackCount)
	handler := &mqttTestHandler{}
	gateway.RegisterEventHandler(handler)

	assert.Equal(t, 1, message.ackCount)
	require.Len(t, handler.events, 1)
}

func TestMQTTGatewayAcknowledgesPoisonMessage(t *testing.T) {
	gateway := newMQTTGatewayForTest(&mqttTestHandler{}, nil)
	message := &mqttTestMessage{
		topic:   "kcab/cab/CAB001/event/operation_progress",
		payload: []byte(`{"msgId":"evt-invalid","deviceId":"CAB001","data":{"stage":"COMMAND_ACK"}}`),
	}

	gateway.processMessage(message)
	assert.Equal(t, 1, message.ackCount)
}

func TestMQTTGatewayRejectsMalformedMessageWithoutConsumingMessageID(t *testing.T) {
	handler := &mqttTestHandler{}
	gateway := newMQTTGatewayForTest(handler, nil)
	invalid := []byte(`{"msgId":"evt-fixed","deviceId":"CAB001","data":{"stage":"DOOR_OPEN"}}`)
	valid := []byte(`{"msgId":"evt-fixed","deviceId":"CAB001","data":{"operationId":"op-1","stage":"DOOR_OPEN"}}`)

	require.ErrorContains(t, gateway.handleIncoming("kcab/cab/CAB001/event/operation_progress", invalid), "operationId")
	require.NoError(t, gateway.handleIncoming("kcab/cab/CAB001/event/operation_progress", valid))
	require.Len(t, handler.events, 1)
}

func TestMQTTGatewayMapsRFIDMismatchToFailure(t *testing.T) {
	handler := &mqttTestHandler{}
	gateway := newMQTTGatewayForTest(handler, nil)
	payload := []byte(`{
		"msgId":"evt-rfid","deviceId":"CAB001",
		"data":{"operationId":"op-return","isMatch":false,"errorCode":"E208"}
	}`)

	require.NoError(t, gateway.handleIncoming("kcab/cab/CAB001/event/rfid_scanned", payload))
	require.Len(t, handler.events, 1)
	assert.Equal(t, "FAILED", handler.events[0].EventType)
	assert.Equal(t, "E208", handler.events[0].ErrorCode)
}

func TestMQTTGatewayHeartbeatUpdatesRuntimeStatus(t *testing.T) {
	sink := &mqttTestStatusSink{}
	gateway := newMQTTGatewayForTest(&mqttTestHandler{}, sink)
	payload := []byte(`{"deviceId":"CAB001","timestamp":1788350430000,"isBusy":false}`)

	require.NoError(t, gateway.handleIncoming("kcab/cab/CAB001/status/heartbeat", payload))
	status, err := gateway.GetDeviceStatus(context.Background(), "CAB001")
	require.NoError(t, err)
	assert.True(t, status.Online)
	assert.Equal(t, "CAB001", sink.deviceID)
	assert.Equal(t, "ONLINE", sink.status)
	assert.Equal(t, time.UnixMilli(1788350430000).UTC(), sink.lastSeen)
}

func TestMQTTGatewayExpiresStaleHeartbeat(t *testing.T) {
	sink := &mqttTestStatusSink{}
	gateway := newMQTTGatewayForTest(&mqttTestHandler{}, sink)
	lastSeen := time.Now().Add(-2 * time.Minute).UTC()
	gateway.deviceStates["CAB001"] = mqttDeviceState{
		status: "ONLINE", lastSeen: lastSeen, receivedAt: time.Now().Add(-2 * time.Minute),
	}

	gateway.expireStaleDevices(time.Now())
	status, err := gateway.GetDeviceStatus(context.Background(), "CAB001")
	require.NoError(t, err)
	assert.False(t, status.Online)
	assert.Equal(t, "OFFLINE", sink.status)
	assert.Equal(t, lastSeen, sink.lastSeen)
}

func TestMQTTGatewaySignalsAbortAcknowledgement(t *testing.T) {
	handler := &mqttTestHandler{}
	gateway := newMQTTGatewayForTest(handler, nil)
	waiter := make(chan abortResult, 1)
	gateway.abortWaiters["op-1"] = waiter
	payload := []byte(`{
		"msgId":"evt-abort","deviceId":"CAB001",
		"data":{"operationId":"op-1","stage":"ABORTED"}
	}`)

	require.NoError(t, gateway.handleIncoming("kcab/cab/CAB001/event/operation_progress", payload))
	select {
	case result := <-waiter:
		require.NoError(t, result.err)
	case <-time.After(time.Second):
		t.Fatal("未收到中止确认")
	}
	require.Len(t, handler.events, 1)
	assert.Equal(t, "ABORT_ACK", handler.events[0].EventType)
}

func TestMQTTGatewayValidatesTopicIdentityAndExternalIDLength(t *testing.T) {
	gateway := newMQTTGatewayForTest(&mqttTestHandler{}, nil)
	payload := []byte(`{"msgId":"evt-1","deviceId":"CAB002","data":{"operationId":"op-1","stage":"SUCCESS"}}`)

	require.ErrorContains(t, gateway.handleIncoming("kcab/cab/CAB001/event/operation_progress", payload), "does not match")
	assert.LessOrEqual(t, len(normalizeExternalEventID(strings.Repeat("x", 100))), 64)
	assert.NotEqual(t, mqttEventID("CAB001", "evt-1"), mqttEventID("CAB002", "evt-1"))
	assert.Equal(t, "cmd_pickup_op-1", commandMessageID("pickup", "op-1"))
}
