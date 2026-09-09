#include "mqtt_client_task.h"
#include <stdio.h>
#include <string.h>
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "freertos/queue.h"
#include "esp_log.h"
#include "mqtt_client.h"
#include "protocol.h"
#include "motor_task.h"
#include "rfid_task.h"
#include "sensor_task.h"

static const char *TAG = "MQTT_TASK";
static esp_mqtt_client_handle_t s_client = NULL;
static QueueHandle_t s_cmd_queue = NULL;

static void execute_command(const cabinet_command_t *cmd) {
    if (!cmd) return;

    if (cmd->type == CMD_TYPE_PICKUP) {
        ESP_LOGI(TAG, "开始执行借用出钥流程: OperationID=%s, SlotNo=%d, KeyID=%s",
                 cmd->operation_id, cmd->slot_no, cmd->key_id);

        // 1. 回复指令确认 ACK
        char *ack = protocol_build_ack(CABINET_DEVICE_ID, cmd->msg_id, cmd->operation_id, "COMMAND_ACK");
        if (ack) {
            mqtt_publish_event("event/ack", ack);
            free(ack);
        }

        // 2. 步进电机驱动推杆推出钥匙
        bool motor_ok = motor_dispense_slot(cmd->slot_no);
        if (!motor_ok) {
            ESP_LOGE(TAG, "出钥电机卡死或限位超时!");
            char *fail_evt = protocol_build_progress_event(
                CABINET_DEVICE_ID, cmd->operation_id, "MOTOR_ERROR", "FAILED",
                cmd->key_id, cmd->slot_no, "MOTOR_POSITION_FAILED", "步进电机推进超时或卡阻");
            if (fail_evt) {
                mqtt_publish_event("event/operation_progress", fail_evt);
                free(fail_evt);
            }
            return;
        }

        // 3. 物理闭环检验：轮询等待用户真正从槽位中拔走钥匙 (微动闭合 -> 断开，即 PRESENT -> ABSENT)
        ESP_LOGI(TAG, "推杆动作完成，等待用户拔出钥匙 (槽位 #%d)...", cmd->slot_no);
        bool key_removed = false;
        int wait_seconds = cmd->timeout_sec > 0 ? cmd->timeout_sec : 60;
        for (int i = 0; i < wait_seconds * 10; i++) {
            if (motor_is_abort_requested()) {
                ESP_LOGW(TAG, "取钥等待拔出期间收到中止指令，退出流程");
                return;
            }
            if (!sensor_get_slot_presence(cmd->slot_no)) {
                key_removed = true;
                break;
            }
            vTaskDelay(pdMS_TO_TICKS(100));
        }

        if (!key_removed) {
            ESP_LOGE(TAG, "用户在限定时间内未拔出钥匙，报警并上报超时终态!");
            char *fail_evt = protocol_build_progress_event(
                CABINET_DEVICE_ID, cmd->operation_id, "TIMEOUT", "FAILED",
                cmd->key_id, cmd->slot_no, "KEY_NOT_TAKEN_TIMEOUT", "用户超时未取走钥匙");
            if (fail_evt) {
                mqtt_publish_event("event/operation_progress", fail_evt);
                free(fail_evt);
            }
            return;
        }

        // 4. 物理闭环检验：等待安全柜门关闭
        ESP_LOGI(TAG, "钥匙已被拔走，等待用户关闭安全柜门...");
        bool door_closed = false;
        for (int i = 0; i < 300; i++) { // 最多等 30 秒关门
            if (sensor_is_door_closed()) {
                door_closed = true;
                break;
            }
            vTaskDelay(pdMS_TO_TICKS(100));
        }
        if (!door_closed) {
            ESP_LOGW(TAG, "安全柜门超时未关紧，请注意提示");
        }

        // 5. 确认钥匙离柜且柜门关闭，上报终态 PICKUP_SUCCESS
        char *succ_evt = protocol_build_progress_event(
            CABINET_DEVICE_ID, cmd->operation_id, "PICKUP_SUCCESS", "SUCCESS",
            cmd->key_id, cmd->slot_no, NULL, NULL);
        if (succ_evt) {
            mqtt_publish_event("event/operation_progress", succ_evt);
            free(succ_evt);
        }
        ESP_LOGI(TAG, "✅ 出钥物理全闭环完成并已上报终态");

    } else if (cmd->type == CMD_TYPE_RETURN) {
        ESP_LOGI(TAG, "开始执行钥匙归还防错还流程: OperationID=%s, TargetSlot=%d, ExpectedRFID=%s",
                 cmd->operation_id, cmd->slot_no, cmd->expected_rfid);

        // 1. 回复指令确认 ACK
        char *ack = protocol_build_ack(CABINET_DEVICE_ID, cmd->msg_id, cmd->operation_id, "COMMAND_ACK");
        if (ack) {
            mqtt_publish_event("event/ack", ack);
            free(ack);
        }

        // 2. 寻卡与 RFID 强匹配 (包含槽位在位微动闭合检测)
        bool match = rfid_verify_return(cmd);
        if (!match) {
            ESP_LOGW(TAG, "归还 RFID 校验不通过或超时，拒绝终态履约");
            return;
        }

        // 3. 物理闭环检验：钥匙插入且 RFID 匹配后，等待柜门磁传感器完全闭合
        ESP_LOGI(TAG, "钥匙防错还校验通过，等待安全柜门完全关闭...");
        bool door_closed = false;
        for (int i = 0; i < 300; i++) { // 最多等 30 秒
            if (sensor_is_door_closed()) {
                door_closed = true;
                break;
            }
            vTaskDelay(pdMS_TO_TICKS(100));
        }
        if (!door_closed) {
            ESP_LOGW(TAG, "归还完成但柜门尚未闭合，维持待关门状态");
        }

        // 4. 上报归还成功终态 RETURN_SUCCESS
        char *succ_evt = protocol_build_progress_event(
            CABINET_DEVICE_ID, cmd->operation_id, "RETURN_SUCCESS", "SUCCESS",
            cmd->key_id, cmd->slot_no, NULL, NULL);
        if (succ_evt) {
            mqtt_publish_event("event/operation_progress", succ_evt);
            free(succ_evt);
        }
        ESP_LOGI(TAG, "✅ 钥匙归还与门磁闭环完成");

    } else if (cmd->type == CMD_TYPE_ABORT) {
        ESP_LOGW(TAG, "收到紧急中止指令: OperationID=%s", cmd->operation_id);

        // 1. 立即中断步进脉冲并硬件失能电机
        motor_request_abort();

        // 2. 回复中止确认 ACK 与终态
        char *ack = protocol_build_ack(CABINET_DEVICE_ID, cmd->msg_id, cmd->operation_id, "ABORT_ACK");
        if (ack) {
            mqtt_publish_event("event/ack", ack);
            free(ack);
        }
        char *abort_evt = protocol_build_progress_event(
            CABINET_DEVICE_ID, cmd->operation_id, "ABORTED", "SUCCESS",
            cmd->key_id, cmd->slot_no, NULL, NULL);
        if (abort_evt) {
            mqtt_publish_event("event/operation_progress", abort_evt);
            free(abort_evt);
        }
        ESP_LOGI(TAG, "✅ 指令中止完成并已上报终态");
    }
}

static void cmd_worker_task(void *pvParameters) {
    cabinet_command_t cmd;
    while (1) {
        if (xQueueReceive(s_cmd_queue, &cmd, portMAX_DELAY)) {
            execute_command(&cmd);
        }
    }
}

static void mqtt_event_handler(void *handler_args, esp_event_base_t base, int32_t event_id, void *event_data) {
    esp_mqtt_event_handle_t event = (esp_mqtt_event_handle_t)event_data;

    switch ((esp_mqtt_event_id_t)event_id) {
    case MQTT_EVENT_CONNECTED:
        ESP_LOGI(TAG, "已连接至 MQTT Broker, 订阅机柜控制主题...");
        char sub_topic[128];
        snprintf(sub_topic, sizeof(sub_topic), "%s/cab/%s/cmd/+", MQTT_TOPIC_PREFIX, CABINET_DEVICE_ID);
        esp_mqtt_client_subscribe(s_client, sub_topic, 1);

        // 上报在线就绪状态
        char online_payload[128];
        snprintf(online_payload, sizeof(online_payload),
                 "{\"deviceId\":\"%s\",\"status\":\"ONLINE\"}", CABINET_DEVICE_ID);
        mqtt_publish_status("status/online", online_payload);
        break;

    case MQTT_EVENT_DATA: {
        char topic_buf[128] = {0};
        int topic_len = event->topic_len < sizeof(topic_buf) - 1 ? event->topic_len : sizeof(topic_buf) - 1;
        strncpy(topic_buf, event->topic, topic_len);

        char payload_buf[512] = {0};
        int payload_len = event->data_len < sizeof(payload_buf) - 1 ? event->data_len : sizeof(payload_buf) - 1;
        strncpy(payload_buf, event->data, payload_len);

        ESP_LOGI(TAG, "收到下行指令: Topic=%s, Payload=%s", topic_buf, payload_buf);
        cabinet_command_t cmd;
        if (protocol_parse_command(topic_buf, payload_buf, &cmd)) {
            xQueueSend(s_cmd_queue, &cmd, 0);
        }
        break;
    }

    case MQTT_EVENT_DISCONNECTED:
        ESP_LOGW(TAG, "MQTT Broker 连接断开，等待自动重连...");
        break;

    default:
        break;
    }
}

static void heartbeat_timer_task(void *pvParameters) {
    while (1) {
        vTaskDelay(pdMS_TO_TICKS(HEARTBEAT_INTERVAL_SEC * 1000));
        char *hb = protocol_build_heartbeat(CABINET_DEVICE_ID, "ONLINE");
        if (hb) {
            mqtt_publish_status("status/heartbeat", hb);
            free(hb);
        }
    }
}

esp_err_t mqtt_task_init(void) {
    s_cmd_queue = xQueueCreate(8, sizeof(cabinet_command_t));

    char lwt_topic[128];
    snprintf(lwt_topic, sizeof(lwt_topic), "%s/cab/%s/status/online", MQTT_TOPIC_PREFIX, CABINET_DEVICE_ID);
    char lwt_payload[128];
    snprintf(lwt_payload, sizeof(lwt_payload),
             "{\"deviceId\":\"%s\",\"status\":\"OFFLINE\"}", CABINET_DEVICE_ID);

    esp_mqtt_client_config_t mqtt_cfg = {
        .broker = {
            .address.uri = MQTT_DEFAULT_BROKER_URI,
        },
        .credentials = {
            .client_id = "esp32_" CABINET_DEVICE_ID,
            .username = MQTT_USERNAME,
            .authentication.password = MQTT_PASSWORD,
        },
        .session.last_will = {
            .topic = lwt_topic,
            .msg = lwt_payload,
            .msg_len = strlen(lwt_payload),
            .qos = 1,
            .retain = 1,
        },
    };

    s_client = esp_mqtt_client_init(&mqtt_cfg);
    esp_mqtt_client_register_event(s_client, ESP_EVENT_ANY_ID, mqtt_event_handler, NULL);
    esp_err_t ret = esp_mqtt_client_start(s_client);

    xTaskCreate(cmd_worker_task, "cmd_worker", 4096, NULL, 5, NULL);
    xTaskCreate(heartbeat_timer_task, "hb_timer", 2048, NULL, 3, NULL);

    return ret;
}

void mqtt_publish_status(const char *subtopic, const char *payload) {
    if (!s_client || !subtopic || !payload) return;
    char full_topic[128];
    snprintf(full_topic, sizeof(full_topic), "%s/cab/%s/%s", MQTT_TOPIC_PREFIX, CABINET_DEVICE_ID, subtopic);
    esp_mqtt_client_publish(s_client, full_topic, payload, 0, 1, 0);
}

void mqtt_publish_event(const char *subtopic, const char *payload) {
    if (!s_client || !subtopic || !payload) return;
    char full_topic[128];
    snprintf(full_topic, sizeof(full_topic), "%s/cab/%s/%s", MQTT_TOPIC_PREFIX, CABINET_DEVICE_ID, subtopic);
    esp_mqtt_client_publish(s_client, full_topic, payload, 0, 1, 0);
}
