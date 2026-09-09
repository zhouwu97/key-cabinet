package device

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var (
	ErrAbortTimeout  = errors.New("device abort acknowledgement timed out")
	ErrAbortRejected = errors.New("device rejected abort request")
)

type MQTTGatewayConfig struct {
	Broker           string
	ClientID         string
	Username         string
	Password         string
	TopicPrefix      string
	QoS              byte
	ConnectTimeout   time.Duration
	CommandTimeout   time.Duration
	HeartbeatTimeout time.Duration
}

type mqttDeviceState struct {
	status     string
	lastSeen   time.Time
	receivedAt time.Time
}

type abortResult struct {
	err error
}

type retryableMessageError struct {
	err error
}

func (e retryableMessageError) Error() string { return e.err.Error() }
func (e retryableMessageError) Unwrap() error { return e.err }

type MQTTDeviceGateway struct {
	client     mqtt.Client
	config     MQTTGatewayConfig
	statusSink DeviceStatusSink
	inventorySink DeviceInventorySink

	handlerMu sync.RWMutex
	handler   DeviceEventHandler
	pending   []mqtt.Message

	mu           sync.Mutex
	deviceStates map[string]mqttDeviceState
	abortWaiters map[string]chan abortResult
	seenMessages map[string]time.Time
	stopCh       chan struct{}
	closeOnce    sync.Once
}

type mqttCommandEnvelope struct {
	MsgID     string          `json:"msgId"`
	Timestamp int64           `json:"timestamp"`
	DeviceID  string          `json:"deviceId"`
	Action    string          `json:"action"`
	Data      mqttCommandData `json:"data"`
}

type mqttCommandData struct {
	OperationID    string `json:"operationId"`
	SlotNo         int    `json:"slotNo,omitempty"`
	TargetSlotNo   int    `json:"targetSlotNo,omitempty"`
	KeyID          string `json:"keyId,omitempty"`
	ExpectedKeyID  string `json:"expectedKeyId,omitempty"`
	ExpectedRFID   string `json:"expectedRfidTag,omitempty"`
	TimeoutSeconds int    `json:"timeoutSeconds,omitempty"`
}

type mqttEventEnvelope struct {
	MsgID        string          `json:"msgId"`
	ReplyMsgID   string          `json:"replyMsgId"`
	Timestamp    int64           `json:"timestamp"`
	DeviceID     string          `json:"deviceId"`
	Status       string          `json:"status"`
	ErrorCode    string          `json:"errorCode"`
	ErrorMessage string          `json:"errorMessage"`
	Data         json.RawMessage `json:"data"`
}

type mqttEventData struct {
	OperationID  string `json:"operationId"`
	Stage        string `json:"stage"`
	Action       string `json:"action"`
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
	IsMatch      *bool  `json:"isMatch"`
}

type mqttStatusEnvelope struct {
	DeviceID  string `json:"deviceId"`
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
}

func NewMQTTDeviceGateway(config MQTTGatewayConfig, statusSink DeviceStatusSink, inventorySink DeviceInventorySink) (*MQTTDeviceGateway, error) {
	if strings.TrimSpace(config.Broker) == "" {
		return nil, errors.New("mqtt broker is required")
	}
	if config.ClientID == "" {
		config.ClientID = "key-cabinet-api"
	}
	config.TopicPrefix = strings.Trim(strings.TrimSpace(config.TopicPrefix), "/")
	if config.TopicPrefix == "" {
		config.TopicPrefix = "kcab"
	}
	if config.ConnectTimeout <= 0 {
		config.ConnectTimeout = 10 * time.Second
	}
	if config.CommandTimeout <= 0 {
		config.CommandTimeout = 5 * time.Second
	}
	if config.HeartbeatTimeout <= 0 {
		config.HeartbeatTimeout = 90 * time.Second
	}
	if config.QoS > 2 {
		return nil, errors.New("mqtt qos must be between 0 and 2")
	}
	if strings.ContainsAny(config.TopicPrefix, "+#") {
		return nil, errors.New("mqtt topic prefix cannot contain wildcard characters")
	}

	gateway := &MQTTDeviceGateway{
		config:        config,
		statusSink:    statusSink,
		inventorySink: inventorySink,
		deviceStates:  make(map[string]mqttDeviceState),
		abortWaiters:  make(map[string]chan abortResult),
		seenMessages:  make(map[string]time.Time),
		stopCh:        make(chan struct{}),
	}
	options := mqtt.NewClientOptions().
		AddBroker(config.Broker).
		SetClientID(config.ClientID).
		SetUsername(config.Username).
		SetPassword(config.Password).
		SetConnectTimeout(config.ConnectTimeout).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetOrderMatters(false).
		SetAutoAckDisabled(true)
	options.SetOnConnectHandler(func(client mqtt.Client) {
		if err := gateway.subscribe(client); err != nil {
			log.Printf("[MQTTDeviceGateway] subscribe failed: %v", err)
		}
	})
	options.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		log.Printf("[MQTTDeviceGateway] connection lost: %v", err)
	})
	gateway.client = mqtt.NewClient(options)
	token := gateway.client.Connect()
	if !token.WaitTimeout(config.ConnectTimeout) {
		return nil, fmt.Errorf("mqtt connection timed out after %s", config.ConnectTimeout)
	}
	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("mqtt connection failed: %w", err)
	}
	go gateway.monitorHeartbeats()
	return gateway, nil
}

func (g *MQTTDeviceGateway) RegisterEventHandler(handler DeviceEventHandler) {
	g.handlerMu.Lock()
	g.handler = handler
	pending := g.pending
	g.pending = nil
	g.handlerMu.Unlock()

	for _, message := range pending {
		g.processMessage(message)
	}
}

func (g *MQTTDeviceGateway) SetInventorySink(sink DeviceInventorySink) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.inventorySink = sink
}

func (g *MQTTDeviceGateway) StartPickup(ctx context.Context, command DeviceCommand) error {
	return g.publishCommand(ctx, "pickup", command)
}

func (g *MQTTDeviceGateway) StartReturn(ctx context.Context, command DeviceCommand) error {
	return g.publishCommand(ctx, "return", command)
}

func (g *MQTTDeviceGateway) AbortOperation(ctx context.Context, command DeviceCommand) error {
	waiter := make(chan abortResult, 1)
	g.mu.Lock()
	if _, exists := g.abortWaiters[command.OperationID]; exists {
		g.mu.Unlock()
		return ErrAbortRejected
	}
	g.abortWaiters[command.OperationID] = waiter
	g.mu.Unlock()
	defer func() {
		g.mu.Lock()
		delete(g.abortWaiters, command.OperationID)
		g.mu.Unlock()
	}()

	if err := g.publishCommand(ctx, "abort", command); err != nil {
		return err
	}
	timer := time.NewTimer(g.config.CommandTimeout)
	defer timer.Stop()
	select {
	case result := <-waiter:
		return result.err
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ErrAbortTimeout
	}
}

func (g *MQTTDeviceGateway) GetDeviceStatus(_ context.Context, deviceID string) (*DeviceStatus, error) {
	g.mu.Lock()
	state, ok := g.deviceStates[deviceID]
	g.mu.Unlock()
	if !ok {
		return &DeviceStatus{DeviceID: deviceID, Online: false}, nil
	}
	online := state.status != "OFFLINE" && time.Since(state.receivedAt) <= g.config.HeartbeatTimeout
	return &DeviceStatus{DeviceID: deviceID, Online: online, LastSeen: state.lastSeen}, nil
}

func (g *MQTTDeviceGateway) Close() {
	g.closeOnce.Do(func() {
		if g.stopCh != nil {
			close(g.stopCh)
		}
		if g.client != nil {
			g.client.Disconnect(250)
		}
	})
}

// monitorHeartbeats 将失联设备主动收敛为离线，避免仅依赖可能缺失的 LWT。
func (g *MQTTDeviceGateway) monitorHeartbeats() {
	interval := g.config.HeartbeatTimeout / 2
	if interval > 30*time.Second {
		interval = 30 * time.Second
	}
	if interval < time.Second {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case now := <-ticker.C:
			g.expireStaleDevices(now)
		case <-g.stopCh:
			return
		}
	}
}

func (g *MQTTDeviceGateway) expireStaleDevices(now time.Time) {
	type staleDevice struct {
		id       string
		lastSeen time.Time
	}
	var stale []staleDevice
	g.mu.Lock()
	for id, state := range g.deviceStates {
		if state.status == "OFFLINE" || state.receivedAt.IsZero() || now.Sub(state.receivedAt) <= g.config.HeartbeatTimeout {
			continue
		}
		state.status = "OFFLINE"
		g.deviceStates[id] = state
		stale = append(stale, staleDevice{id: id, lastSeen: state.lastSeen})
	}
	g.mu.Unlock()
	if g.statusSink == nil {
		return
	}
	for _, device := range stale {
		if err := g.statusSink.UpdateRuntimeStatus(context.Background(), device.id, "OFFLINE", device.lastSeen); err != nil {
			log.Printf("[MQTTDeviceGateway] failed to persist heartbeat timeout for %s: %v", device.id, err)
		}
	}
}

func (g *MQTTDeviceGateway) subscribe(client mqtt.Client) error {
	base := g.config.TopicPrefix + "/cab/+/"
	topics := map[string]byte{
		base + "event/operation_progress": g.config.QoS,
		base + "event/rfid_scanned":       g.config.QoS,
		base + "event/ack":                g.config.QoS,
		base + "status/heartbeat":         g.config.QoS,
		base + "status/online":            g.config.QoS,
		base + "status/inventory":         g.config.QoS,
	}
	token := client.SubscribeMultiple(topics, g.onMessage)
	if !token.WaitTimeout(g.config.ConnectTimeout) {
		return errors.New("mqtt subscription timed out")
	}
	return token.Error()
}

func (g *MQTTDeviceGateway) publishCommand(ctx context.Context, topicAction string, command DeviceCommand) error {
	if g.client == nil || !g.client.IsConnectionOpen() {
		return errors.New("mqtt broker is not connected")
	}
	if command.OperationID == "" || command.DeviceID == "" {
		return errors.New("operation id and device id are required")
	}
	if strings.ContainsAny(command.DeviceID, "/+#") {
		return errors.New("device id contains invalid mqtt topic characters")
	}
	payload, err := json.Marshal(g.commandEnvelope(topicAction, command, time.Now()))
	if err != nil {
		return fmt.Errorf("failed to encode mqtt command: %w", err)
	}
	topic := fmt.Sprintf("%s/cab/%s/cmd/%s", g.config.TopicPrefix, command.DeviceID, topicAction)
	token := g.client.Publish(topic, g.config.QoS, false, payload)
	if err := ctx.Err(); err != nil {
		return err
	}
	if !token.WaitTimeout(g.config.CommandTimeout) {
		if err := ctx.Err(); err != nil {
			return err
		}
		return errors.New("mqtt command publish timed out")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := token.Error(); err != nil {
		return fmt.Errorf("mqtt command publish failed: %w", err)
	}
	return nil
}

func (g *MQTTDeviceGateway) commandEnvelope(topicAction string, command DeviceCommand, now time.Time) mqttCommandEnvelope {
	timeoutSeconds := command.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 120
	}
	data := mqttCommandData{OperationID: command.OperationID, TimeoutSeconds: timeoutSeconds}
	switch topicAction {
	case "pickup":
		data.SlotNo = command.SlotNo
		data.KeyID = command.KeyID
	case "return":
		data.TargetSlotNo = command.SlotNo
		data.ExpectedKeyID = command.KeyID
		data.ExpectedRFID = command.ExpectedRFID
	case "abort":
		data.TimeoutSeconds = max(1, int((g.config.CommandTimeout+time.Second-1)/time.Second))
	}
	return mqttCommandEnvelope{
		MsgID:     commandMessageID(topicAction, command.OperationID),
		Timestamp: now.UnixMilli(),
		DeviceID:  command.DeviceID,
		Action:    strings.ToUpper(topicAction),
		Data:      data,
	}
}

func (g *MQTTDeviceGateway) onMessage(_ mqtt.Client, message mqtt.Message) {
	if strings.Contains(message.Topic(), "/event/") && g.queueUntilHandlerRegistered(message) {
		return
	}
	g.processMessage(message)
}

func (g *MQTTDeviceGateway) queueUntilHandlerRegistered(message mqtt.Message) bool {
	g.handlerMu.Lock()
	defer g.handlerMu.Unlock()
	if g.handler != nil {
		return false
	}
	g.pending = append(g.pending, message)
	return true
}

func (g *MQTTDeviceGateway) processMessage(message mqtt.Message) {
	if err := g.handleIncoming(message.Topic(), message.Payload()); err != nil {
		var retryable retryableMessageError
		if errors.As(err, &retryable) {
			log.Printf("[MQTTDeviceGateway] message processing failed on %s, waiting for redelivery: %v", message.Topic(), err)
			return
		}
		log.Printf("[MQTTDeviceGateway] discarded invalid message on %s: %v", message.Topic(), err)
	}
	message.Ack()
}

func (g *MQTTDeviceGateway) handleIncoming(topic string, payload []byte) error {
	deviceID, suffix, err := g.parseTopic(topic)
	if err != nil {
		return err
	}
	if strings.HasPrefix(suffix, "status/") {
		if suffix == "status/inventory" {
			return g.handleInventory(deviceID, payload)
		}
		return g.handleStatus(deviceID, suffix, payload)
	}
	var envelope mqttEventEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return fmt.Errorf("invalid event json: %w", err)
	}
	if envelope.DeviceID == "" || envelope.DeviceID != deviceID {
		return errors.New("event device id does not match topic")
	}
	if envelope.MsgID == "" {
		return errors.New("event msgId is required")
	}
	var data mqttEventData
	if len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, &data); err != nil {
			return fmt.Errorf("invalid event data: %w", err)
		}
	}
	if data.OperationID == "" {
		return errors.New("event operationId is required")
	}
	eventType := strings.ToUpper(strings.TrimSpace(data.Stage))
	if strings.HasSuffix(suffix, "rfid_scanned") && data.IsMatch != nil {
		if *data.IsMatch {
			eventType = "RFID_CONFIRMED"
		} else {
			eventType = "FAILED"
		}
	}
	if eventType == "" && envelope.ReplyMsgID != "" && strings.EqualFold(envelope.Status, "SUCCESS") {
		eventType = "COMMAND_ACK"
	}
	if eventType == "" {
		eventType = strings.ToUpper(strings.TrimSpace(envelope.Status))
	}
	if eventType == "" {
		return errors.New("event stage or status is required")
	}
	dedupKey := deviceID + "\x00" + envelope.MsgID
	if g.hasSeenMessage(dedupKey) {
		return nil
	}
	errorCode := firstNonEmpty(data.ErrorCode, envelope.ErrorCode)
	errorMessage := firstNonEmpty(data.ErrorMessage, envelope.ErrorMessage)
	shouldSignalAbort := false
	var abortErr error
	if eventType == "ABORTED" || eventType == "CANCELLED" || eventType == "ABORT_ACK" {
		shouldSignalAbort = true
		eventType = "ABORT_ACK"
	} else if eventType == "ABORT_FAILED" {
		shouldSignalAbort = true
		abortErr = ErrAbortRejected
	}
	var eventData map[string]interface{}
	if len(envelope.Data) > 0 {
		_ = json.Unmarshal(envelope.Data, &eventData)
	}
	g.handlerMu.RLock()
	handler := g.handler
	g.handlerMu.RUnlock()
	if handler == nil {
		return errors.New("device event handler is not registered")
	}
	err = handler.OnDeviceEvent(context.Background(), DeviceEvent{
		EventID: mqttEventID(deviceID, envelope.MsgID), OperationID: data.OperationID,
		DeviceID: deviceID, EventType: eventType, Timestamp: messageTime(envelope.Timestamp),
		ErrorCode: errorCode, ErrorMessage: errorMessage, Data: eventData,
	})
	if err != nil {
		return retryableMessageError{err: err}
	}
	// 业务处理成功后才确认去重，数据库短暂失败时允许 QoS 1 重投恢复。
	g.markMessageSeen(dedupKey)
	if shouldSignalAbort {
		g.signalAbort(data.OperationID, abortErr)
	}
	return nil
}

func (g *MQTTDeviceGateway) handleStatus(deviceID, suffix string, payload []byte) error {
	var status mqttStatusEnvelope
	if err := json.Unmarshal(payload, &status); err != nil {
		return fmt.Errorf("invalid status json: %w", err)
	}
	if status.DeviceID == "" || status.DeviceID != deviceID {
		return errors.New("status device id does not match topic")
	}
	deviceStatus := strings.ToUpper(strings.TrimSpace(status.Status))
	if strings.HasSuffix(suffix, "heartbeat") {
		deviceStatus = "ONLINE"
	}
	if deviceStatus != "ONLINE" && deviceStatus != "OFFLINE" {
		return errors.New("unsupported device status")
	}
	lastSeen := messageTime(status.Timestamp)
	g.mu.Lock()
	g.deviceStates[deviceID] = mqttDeviceState{status: deviceStatus, lastSeen: lastSeen, receivedAt: time.Now()}
	g.mu.Unlock()
	if g.statusSink != nil {
		if err := g.statusSink.UpdateRuntimeStatus(context.Background(), deviceID, deviceStatus, lastSeen); err != nil {
			return retryableMessageError{err: fmt.Errorf("failed to persist device status: %w", err)}
		}
	}
	return nil
}

type mqttInventorySlot struct {
	SlotNo   int    `json:"slotNo"`
	Presence bool   `json:"presence"`
	RFID     string `json:"rfid,omitempty"`
}

type mqttInventoryPayload struct {
	DeviceID   string              `json:"deviceId"`
	Timestamp  int64               `json:"timestamp"`
	DoorClosed bool                `json:"doorClosed"`
	Slots      []mqttInventorySlot `json:"slots"`
}

func (g *MQTTDeviceGateway) handleInventory(deviceID string, payload []byte) error {
	var inv mqttInventoryPayload
	if err := json.Unmarshal(payload, &inv); err != nil {
		return fmt.Errorf("invalid inventory json: %w", err)
	}
	if inv.DeviceID == "" || inv.DeviceID != deviceID {
		return errors.New("inventory device id does not match topic")
	}
	lastSeen := messageTime(inv.Timestamp)
	g.mu.Lock()
	g.deviceStates[deviceID] = mqttDeviceState{status: "ONLINE", lastSeen: lastSeen, receivedAt: time.Now()}
	sink := g.inventorySink
	statusSink := g.statusSink
	g.mu.Unlock()

	if statusSink != nil {
		_ = statusSink.UpdateRuntimeStatus(context.Background(), deviceID, "ONLINE", lastSeen)
	}

	snapshot := DeviceInventorySnapshot{
		DeviceID:   deviceID,
		Timestamp:  lastSeen,
		DoorClosed: inv.DoorClosed,
		Slots:      make([]DeviceInventorySlot, len(inv.Slots)),
	}
	for i, s := range inv.Slots {
		snapshot.Slots[i] = DeviceInventorySlot{
			SlotNo:   s.SlotNo,
			Presence: s.Presence,
			RFID:     s.RFID,
		}
	}

	if sink != nil {
		if err := sink.OnInventorySnapshot(context.Background(), snapshot); err != nil {
			log.Printf("[MQTTDeviceGateway] inventory sink reconciliation error for %s: %v", deviceID, err)
		}
	}
	return nil
}

func (g *MQTTDeviceGateway) parseTopic(topic string) (string, string, error) {
	prefix := g.config.TopicPrefix + "/cab/"
	if !strings.HasPrefix(topic, prefix) {
		return "", "", errors.New("topic is outside configured prefix")
	}
	remainder := strings.TrimPrefix(topic, prefix)
	parts := strings.SplitN(remainder, "/", 2)
	if len(parts) != 2 || parts[0] == "" {
		return "", "", errors.New("invalid device topic")
	}
	return parts[0], parts[1], nil
}

func (g *MQTTDeviceGateway) hasSeenMessage(messageID string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	_, exists := g.seenMessages[messageID]
	return exists
}

func (g *MQTTDeviceGateway) markMessageSeen(messageID string) {
	now := time.Now()
	g.mu.Lock()
	defer g.mu.Unlock()
	g.seenMessages[messageID] = now
	if len(g.seenMessages) > 4096 {
		cutoff := now.Add(-time.Hour)
		for id, seenAt := range g.seenMessages {
			if seenAt.Before(cutoff) {
				delete(g.seenMessages, id)
			}
		}
	}
}

func (g *MQTTDeviceGateway) signalAbort(operationID string, err error) {
	g.mu.Lock()
	waiter := g.abortWaiters[operationID]
	g.mu.Unlock()
	if waiter == nil {
		return
	}
	select {
	case waiter <- abortResult{err: err}:
	default:
	}
}

func commandMessageID(action, operationID string) string {
	return fmt.Sprintf("cmd_%s_%s", strings.ToLower(action), operationID)
}

func normalizeExternalEventID(value string) string {
	if len(value) <= 64 {
		return value
	}
	digest := sha256.Sum256([]byte(value))
	return "mqtt_" + hex.EncodeToString(digest[:])[:48]
}

func mqttEventID(deviceID, messageID string) string {
	return normalizeExternalEventID(deviceID + "_" + messageID)
}

func messageTime(timestamp int64) time.Time {
	if timestamp <= 0 {
		return time.Now().UTC()
	}
	parsed := time.UnixMilli(timestamp).UTC()
	// 若设备开机时间戳未通过 SNTP 网络校时 (如 1970 年)，或超过未来 5 分钟，收敛为服务器当前时间
	if parsed.Year() < 2024 || parsed.After(time.Now().Add(5*time.Minute)) {
		return time.Now().UTC()
	}
	return parsed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
