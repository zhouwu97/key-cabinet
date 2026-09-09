#include "rfid_task.h"
#include <string.h>
#include <strings.h>
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "driver/gpio.h"
#include "esp_log.h"
#include "rc522.h"
#include "sensor_task.h"
#include "protocol.h"

static const char *TAG = "RFID_TASK";

extern void mqtt_publish_status(const char *subtopic, const char *payload);

static void beep_feedback(int count, int duration_ms) {
    for (int i = 0; i < count; i++) {
        gpio_set_level(PIN_BUZZER, 1);
        vTaskDelay(pdMS_TO_TICKS(duration_ms));
        gpio_set_level(PIN_BUZZER, 0);
        vTaskDelay(pdMS_TO_TICKS(duration_ms));
    }
}

bool rfid_verify_return(const cabinet_command_t *cmd) {
    if (!cmd) return false;

    ESP_LOGI(TAG, "等待用户将钥匙插入槽位 #%d (超时=%d 秒)...", cmd->slot_no, cmd->timeout_sec);
    int wait_limit = cmd->timeout_sec > 0 ? cmd->timeout_sec : 60;
    bool inserted = false;

    // 1. 轮询等待钥匙插入微动闭合
    for (int i = 0; i < wait_limit * 10; i++) {
        if (sensor_get_slot_presence(cmd->slot_no)) {
            inserted = true;
            break;
        }
        vTaskDelay(pdMS_TO_TICKS(100));
    }

    if (!inserted) {
        ESP_LOGW(TAG, "归还超时: 用户未在限定时间内插入钥匙");
        beep_feedback(3, 100);
        char *evt = protocol_build_rfid_event(CABINET_DEVICE_ID, cmd->operation_id, cmd->slot_no,
                                              cmd->expected_rfid, "", false);
        if (evt) {
            mqtt_publish_status("event/rfid_scanned", evt);
            free(evt);
        }
        return false;
    }

    // 插入后微延时稳定触点
    vTaskDelay(pdMS_TO_TICKS(200));

    // 2. RC522 寻卡读取 UID (最多重试 5 次)
    char scanned_uid[32] = {0};
    bool read_ok = false;
    for (int retry = 0; retry < 5; retry++) {
        if (rc522_read_uid(scanned_uid, sizeof(scanned_uid))) {
            read_ok = true;
            break;
        }
        vTaskDelay(pdMS_TO_TICKS(50));
    }

    if (!read_ok) {
        ESP_LOGE(TAG, "防错还拦截: 未能识别到钥匙 RFID 标签!");
        beep_feedback(3, 150);
        char *evt = protocol_build_rfid_event(CABINET_DEVICE_ID, cmd->operation_id, cmd->slot_no,
                                              cmd->expected_rfid, "", false);
        if (evt) {
            mqtt_publish_status("event/rfid_scanned", evt);
            free(evt);
        }
        return false;
    }

    // 3. 强校验实扫 UID 与预期 expectedRfidTag
    bool is_match = (strcasecmp(scanned_uid, cmd->expected_rfid) == 0);
    if (is_match) {
        ESP_LOGI(TAG, "✅ 标签比对一致 (UID=%s, 预期=%s)", scanned_uid, cmd->expected_rfid);
        beep_feedback(1, 80); // 短鸣一声确认
    } else {
        ESP_LOGE(TAG, "❌ 防错还拦截: 钥匙不匹配! 实读=%s, 预期=%s", scanned_uid, cmd->expected_rfid);
        beep_feedback(3, 200); // 连续三声警报
    }

    char *evt = protocol_build_rfid_event(CABINET_DEVICE_ID, cmd->operation_id, cmd->slot_no,
                                          cmd->expected_rfid, scanned_uid, is_match);
    if (evt) {
        mqtt_publish_status("event/rfid_scanned", evt);
        free(evt);
    }

    return is_match;
}
