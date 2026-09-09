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

        // 3. 上报出钥成功 PICKUP_SUCCESS
        char *succ_evt = protocol_build_progress_event(
            CABINET_DEVICE_ID, cmd->operation_id, "PICKUP_SUCCESS", "SUCCESS",
            cmd->key_id, cmd->slot_no, NULL, NULL);
        if (succ_evt) {
            mqtt_publish_event("event/operation_progress", succ_evt);
            free(succ_evt);
        }
        ESP_LOGI(TAG, "✅ 出钥完成并已上报终态");

    } else if (cmd->type == CMD_TYPE_RETURN) {
        ESP_LOGI(TAG, "开始执行钥匙归还防错还流程: OperationID=%s, TargetSlot=%d, ExpectedRFID=%s",
                 cmd->operation_id, cmd->slot_no, cmd->expected_rfid);

        // 1. 回复指令确认 ACK
        char *ack = protocol_build_ack(CABINET_DEVICE_ID, cmd->msg_id, cmd->operation_id, "COMMAND_ACK");
        if (ack) {
            mqtt_publish_event("event/ack", ack);
            free(ack);
        }

        // 2. 寻卡与 RFID 强匹配
        bool match = rfid_verify_return(cmd);
        if (!match) {
            ESP_LOGW(TAG, "归还 RFID 校验不通过，拒绝终态履约");
            return;
        }

        // 3. 上报归还成功终态
        char *succ_evt = protocol_build_progress_event(
            CABINET_DEVICE_ID, cmd->operation_id, "RETURN_SUCCESS", "SUCCESS",
            cmd->key_id, cmd->slot_no, NULL, NULL);
        if (succ_evt) {
            mqtt_publish_event("event/operation_progress", succ_evt);
            free(succ_evt);
        }
        ESP_LOGI(TAG, "✅ 钥匙归还闭环完成");
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
        .broker.address.uri = MQTT_DEFAULT_BROKER_URI,
        .credentials.client_id = "esp32_" CABINET_DEVICE_ID,
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
