package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type commandEnvelope struct {
	MsgID     string          `json:"msgId"`
	Timestamp int64           `json:"timestamp"`
	DeviceID  string          `json:"deviceId"`
	Action    string          `json:"action"`
	Data      json.RawMessage `json:"data"`
}

type pickupCommandData struct {
	OperationID    string `json:"operationId"`
	SlotNo         int    `json:"slotNo"`
	KeyID          string `json:"keyId"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
}

type returnCommandData struct {
	OperationID    string `json:"operationId"`
	TargetSlotNo   int    `json:"targetSlotNo"`
	ExpectedKeyID  string `json:"expectedKeyId"`
	ExpectedRFID   string `json:"expectedRfid"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
}

type eventEnvelope struct {
	MsgID       string      `json:"msgId"`
	Timestamp   int64       `json:"timestamp"`
	DeviceID    string      `json:"deviceId"`
	ReplyMsgID  string      `json:"replyMsgId,omitempty"`
	Status      string      `json:"status,omitempty"`
	ErrorCode   string      `json:"errorCode,omitempty"`
	ErrorMessage string     `json:"errorMessage,omitempty"`
	Data        interface{} `json:"data,omitempty"`
}

func main() {
	broker := flag.String("broker", "tcp://localhost:1883", "MQTT broker URL")
	deviceID := flag.String("device", "CAB001", "Simulated device ID")
	prefix := flag.String("prefix", "kcab", "MQTT topic prefix")
	failRFID := flag.Bool("fail-rfid", false, "Simulate RFID mismatch failure on return")
	failMotor := flag.Bool("fail-motor", false, "Simulate stepper motor failure on pickup")
	flag.Parse()

	log.Printf("=== 智能钥匙柜 ESP32 硬件行为模拟器 ===")
	log.Printf("机柜 ID: %s", *deviceID)
	log.Printf("Broker: %s", *broker)
	log.Printf("Topic Prefix: %s", *prefix)
	log.Printf("故障注入: fail-rfid=%v, fail-motor=%v", *failRFID, *failMotor)

	opts := mqtt.NewClientOptions().
		AddBroker(*broker).
		SetClientID(fmt.Sprintf("sim_%s_%d", *deviceID, time.Now().Unix())).
		SetAutoReconnect(true).
		SetConnectRetry(true)

	client := mqtt.NewClient(opts)
	token := client.Connect()
	if !token.WaitTimeout(5 * time.Second) {
		log.Printf("[警告] MQTT 连接超时 (可能是本地未启动 Broker，模拟器将继续以离线逻辑保持就绪)")
	} else if err := token.Error(); err != nil {
		log.Printf("[警告] MQTT 连接失败: %v (可在启动 Mosquitto 后自动重连)", err)
	} else {
		log.Printf("[成功] 已连接至 MQTT Broker: %s", *broker)
	}

	subTopic := fmt.Sprintf("%s/cab/%s/cmd/+", *prefix, *deviceID)
	client.Subscribe(subTopic, 1, func(c mqtt.Client, m mqtt.Message) {
		topic := m.Topic()
		log.Printf("\n[📥 收到控制指令] Topic: %s\nPayload: %s", topic, string(m.Payload()))

		var cmd commandEnvelope
		if err := json.Unmarshal(m.Payload(), &cmd); err != nil {
			log.Printf("[错误] 解析指令失败: %v", err)
			return
		}

		parts := strings.Split(topic, "/")
		action := parts[len(parts)-1]

		switch action {
		case "pickup":
			handlePickup(client, *prefix, *deviceID, cmd, *failMotor)
		case "return":
			handleReturn(client, *prefix, *deviceID, cmd, *failRFID)
		case "abort":
			handleAbort(client, *prefix, *deviceID, cmd)
		default:
			log.Printf("[提示] 未知指令动作: %s", action)
		}
	})

	// 周期发送心跳 (每20秒)
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if client.IsConnected() {
				heartbeatTopic := fmt.Sprintf("%s/cab/%s/status/heartbeat", *prefix, *deviceID)
				payload, _ := json.Marshal(map[string]interface{}{
					"deviceId":  *deviceID,
					"status":    "ONLINE",
					"timestamp": time.Now().UnixMilli(),
				})
				client.Publish(heartbeatTopic, 0, false, payload)
				log.Printf("[💓 心跳广播] %s: ONLINE", *deviceID)
			}
		}
	}()

	log.Printf("[就绪] 模拟器正在监听指令 topic: %s", subTopic)
	log.Printf("按 Ctrl+C 退出模拟器...")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("\n正在关闭模拟器...")
	client.Disconnect(250)
	log.Println("模拟器已退出。")
}

func handlePickup(client mqtt.Client, prefix, deviceID string, cmd commandEnvelope, failMotor bool) {
	var data pickupCommandData
	_ = json.Unmarshal(cmd.Data, &data)

	eventTopic := fmt.Sprintf("%s/cab/%s/event/operation_progress", prefix, deviceID)

	// 1. 立即回复指令确认 ACK
	ackTopic := fmt.Sprintf("%s/cab/%s/event/ack", prefix, deviceID)
	ack := eventEnvelope{
		MsgID:      fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Timestamp:  time.Now().UnixMilli(),
		DeviceID:   deviceID,
		ReplyMsgID: cmd.MsgID,
		Status:     "SUCCESS",
		Data: map[string]interface{}{
			"operationId": data.OperationID,
			"stage":       "COMMAND_ACK",
		},
	}
	publishJSON(client, ackTopic, ack)
	log.Printf("  └─ 📤 回复指令确认 ACK: %s", cmd.MsgID)

	time.Sleep(300 * time.Millisecond)

	// 2. 模拟步进电机运转推进机构
	if failMotor {
		log.Printf("  └─ ⚠️ [故障注入] 步进电机失步/卡死！")
		failEvent := eventEnvelope{
			MsgID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()),
			Timestamp:    time.Now().UnixMilli(),
			DeviceID:     deviceID,
			Status:       "FAILED",
			ErrorCode:    "MOTOR_POSITION_FAILED",
			ErrorMessage: "步进电机运行超时或机械推杆卡阻",
			Data: map[string]interface{}{
				"operationId": data.OperationID,
				"stage":       "MOTOR_ERROR",
			},
		}
		publishJSON(client, eventTopic, failEvent)
		return
	}

	log.Printf("  └─ ⚙️ 步进电机运转推杆至 槽位 #%d...", data.SlotNo)
	progress1 := eventEnvelope{
		MsgID:     fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Timestamp: time.Now().UnixMilli(),
		DeviceID:  deviceID,
		Status:    "EXECUTING",
		Data: map[string]interface{}{
			"operationId": data.OperationID,
			"stage":       "MOTOR_ARRIVED",
			"slotNo":      data.SlotNo,
		},
	}
	publishJSON(client, eventTopic, progress1)

	time.Sleep(500 * time.Millisecond)

	// 3. 微动开关检测钥匙脱出
	log.Printf("  └─ 🔓 槽位微动检测钥匙脱出，推出完成！")
	successEvent := eventEnvelope{
		MsgID:     fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Timestamp: time.Now().UnixMilli(),
		DeviceID:  deviceID,
		Status:    "SUCCESS",
		Data: map[string]interface{}{
			"operationId": data.OperationID,
			"stage":       "PICKUP_SUCCESS",
			"keyId":       data.KeyID,
			"slotNo":      data.SlotNo,
		},
	}
	publishJSON(client, eventTopic, successEvent)
	log.Printf("  └─ ✅ 出钥操作完成: OperationID=%s", data.OperationID)
}

func handleReturn(client mqtt.Client, prefix, deviceID string, cmd commandEnvelope, failRFID bool) {
	var data returnCommandData
	_ = json.Unmarshal(cmd.Data, &data)

	eventTopic := fmt.Sprintf("%s/cab/%s/event/operation_progress", prefix, deviceID)
	rfidTopic := fmt.Sprintf("%s/cab/%s/event/rfid_scanned", prefix, deviceID)

	// 1. 回复指令确认 ACK
	ackTopic := fmt.Sprintf("%s/cab/%s/event/ack", prefix, deviceID)
	ack := eventEnvelope{
		MsgID:      fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Timestamp:  time.Now().UnixMilli(),
		DeviceID:   deviceID,
		ReplyMsgID: cmd.MsgID,
		Status:     "SUCCESS",
		Data: map[string]interface{}{
			"operationId": data.OperationID,
			"stage":       "COMMAND_ACK",
		},
	}
	publishJSON(client, ackTopic, ack)
	log.Printf("  └─ 📤 回复归还指令确认 ACK")

	time.Sleep(400 * time.Millisecond)

	// 2. 模拟微动在位检测与 RC522 RFID 读卡
	log.Printf("  └─ 📡 RC522 射频天线感应读卡中 (槽位 #%d)...", data.TargetSlotNo)
	time.Sleep(300 * time.Millisecond)

	scannedUID := data.ExpectedRFID
	isMatch := true
	if failRFID || scannedUID == "" {
		scannedUID = "RFID_WRONG_9999"
		isMatch = false
		log.Printf("  └─ ❌ [故障注入/错卡] 读卡 UID 不符! 预期=%s, 实读=%s", data.ExpectedRFID, scannedUID)
	} else {
		log.Printf("  └─ 🏷️ 读取到合法标签 UID: %s (匹配钥匙: %s)", scannedUID, data.ExpectedKeyID)
	}

	rfidEvent := eventEnvelope{
		MsgID:     fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Timestamp: time.Now().UnixMilli(),
		DeviceID:  deviceID,
		Status:    "SUCCESS",
		Data: map[string]interface{}{
			"operationId": data.OperationID,
			"stage":       "rfid_scanned",
			"isMatch":     isMatch,
			"scannedRfid": scannedUID,
			"uid":         scannedUID,
		},
	}
	publishJSON(client, rfidTopic, rfidEvent)

	if !isMatch {
		log.Printf("  └─ ⛔ 错卡拦截，拒绝完成归还流程！")
		return
	}

	time.Sleep(400 * time.Millisecond)

	// 3. 门磁关闭与在位确认，上报归还成功
	log.Printf("  └─ 🔒 柜门闭锁，钥匙完全在位，归还履约成功！")
	successEvent := eventEnvelope{
		MsgID:     fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Timestamp: time.Now().UnixMilli(),
		DeviceID:  deviceID,
		Status:    "SUCCESS",
		Data: map[string]interface{}{
			"operationId": data.OperationID,
			"stage":       "RETURN_SUCCESS",
			"keyId":       data.ExpectedKeyID,
			"slotNo":      data.TargetSlotNo,
		},
	}
	publishJSON(client, eventTopic, successEvent)
	log.Printf("  └─ ✅ 归还闭环完成: OperationID=%s", data.OperationID)
}

func handleAbort(client mqtt.Client, prefix, deviceID string, cmd commandEnvelope) {
	var data struct {
		OperationID string `json:"operationId"`
	}
	_ = json.Unmarshal(cmd.Data, &data)

	log.Printf("  └─ ⏹️ 执行操作紧急中止: %s", data.OperationID)
	eventTopic := fmt.Sprintf("%s/cab/%s/event/operation_progress", prefix, deviceID)
	abortEvent := eventEnvelope{
		MsgID:     fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Timestamp: time.Now().UnixMilli(),
		DeviceID:  deviceID,
		Status:    "SUCCESS",
		Data: map[string]interface{}{
			"operationId": data.OperationID,
			"stage":       "ABORTED",
		},
	}
	publishJSON(client, eventTopic, abortEvent)
}

func publishJSON(client mqtt.Client, topic string, payload interface{}) {
	if client == nil || !client.IsConnected() {
		return
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return
	}
	client.Publish(topic, 1, false, bytes)
}
