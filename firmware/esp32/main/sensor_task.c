#include "sensor_task.h"
#include <string.h>
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "driver/gpio.h"
#include "esp_log.h"
#include "rc522.h"
#include "protocol.h"

static const char *TAG = "SENSOR";
static slot_status_t s_slots[TOTAL_CABINET_SLOTS];

void sensor_init(void) {
    // 1. 槽位在位微动引脚 (内部上拉，钥匙插入闭合接地)
    gpio_config_t in_conf = {
        .pin_bit_mask = (1ULL << PIN_SLOT1_PRESENCE) | (1ULL << PIN_SLOT2_PRESENCE),
        .mode = GPIO_MODE_INPUT,
        .pull_up_en = GPIO_PULLUP_ENABLE,
        .pull_down_en = GPIO_PULLDOWN_DISABLE,
    };
    gpio_config(&in_conf);

    // 2. 柜门磁开关 (GPIO 35 纯输入，依赖板载 10k 外部上拉，门关接地)
    gpio_config_t door_conf = {
        .pin_bit_mask = (1ULL << PIN_DOOR_MAGNET),
        .mode = GPIO_MODE_INPUT,
        .pull_up_en = GPIO_PULLUP_DISABLE,
        .pull_down_en = GPIO_PULLDOWN_DISABLE,
    };
    gpio_config(&door_conf);

    // 3. 状态指示 LED 与蜂鸣器
    gpio_config_t out_conf = {
        .pin_bit_mask = (1ULL << PIN_STATUS_LED) | (1ULL << PIN_BUZZER),
        .mode = GPIO_MODE_OUTPUT,
        .pull_up_en = GPIO_PULLUP_DISABLE,
        .pull_down_en = GPIO_PULLDOWN_DISABLE,
    };
    gpio_config(&out_conf);
    gpio_set_level(PIN_STATUS_LED, 0);
    gpio_set_level(PIN_BUZZER, 0);

    for (int i = 0; i < TOTAL_CABINET_SLOTS; i++) {
        s_slots[i].slot_no = i + 1;
        s_slots[i].presence = false;
        s_slots[i].rfid_tag[0] = '\0';
    }

    ESP_LOGI(TAG, "槽位微动、柜门磁与声光指示 GPIO 初始化完成");
}

bool sensor_get_slot_presence(int slot_no) {
    if (slot_no == 1) {
        // 低电平表示插入微动闭合
        return gpio_get_level(PIN_SLOT1_PRESENCE) == 0;
    } else if (slot_no == 2) {
        return gpio_get_level(PIN_SLOT2_PRESENCE) == 0;
    }
    return false;
}

bool sensor_is_door_closed(void) {
    // 门磁闭合拉低表示关门，悬空上拉表示开门
    return gpio_get_level(PIN_DOOR_MAGNET) == 0;
}

void sensor_get_all_slots(slot_status_t *out_slots, int *out_count) {
    if (!out_slots || !out_count) return;
    *out_count = TOTAL_CABINET_SLOTS;
    for (int i = 0; i < TOTAL_CABINET_SLOTS; i++) {
        out_slots[i] = s_slots[i];
    }
}

// 外部声明 MQTT 发布接口
extern void mqtt_publish_status(const char *subtopic, const char *payload);

static void sensor_periodic_task(void *pvParameters) {
    while (1) {
        bool changed = false;
        for (int i = 0; i < TOTAL_CABINET_SLOTS; i++) {
            bool current_presence = sensor_get_slot_presence(i + 1);
            if (current_presence != s_slots[i].presence) {
                s_slots[i].presence = current_presence;
                changed = true;
                if (current_presence) {
                    // 若检测到钥匙新插入，尝试读取射频标签
                    char uid[32] = {0};
                    if (rc522_read_uid(uid, sizeof(uid))) {
                        strncpy(s_slots[i].rfid_tag, uid, sizeof(s_slots[i].rfid_tag) - 1);
                    }
                } else {
                    s_slots[i].rfid_tag[0] = '\0';
                }
            }
        }

        // 定期或状态发生变化时主动上报整柜在位快照 (status/inventory)
        static int timer_count = 0;
        timer_count++;
        if (changed || timer_count >= INVENTORY_INTERVAL_SEC) {
            timer_count = 0;
            bool door_closed = sensor_is_door_closed();
            char *json = protocol_build_inventory_snapshot(CABINET_DEVICE_ID, s_slots, TOTAL_CABINET_SLOTS, door_closed);
            if (json) {
                mqtt_publish_status("status/inventory", json);
                free(json);
            }
        }

        vTaskDelay(pdMS_TO_TICKS(1000));
    }
}

void sensor_task_start(void) {
    xTaskCreate(sensor_periodic_task, "sensor_task", 4096, NULL, 4, NULL);
}
