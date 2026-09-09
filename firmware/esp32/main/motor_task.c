#include "motor_task.h"
#include <stdio.h>
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "driver/gpio.h"
#include "esp_log.h"
#include "rom/ets_sys.h"

static const char *TAG = "MOTOR";

esp_err_t motor_init(void) {
    // 1. 配置步进电机控制引脚 (STEP, DIR, EN)
    gpio_config_t out_conf = {
        .pin_bit_mask = (1ULL << PIN_MOTOR_STEP) | (1ULL << PIN_MOTOR_DIR) | (1ULL << PIN_MOTOR_EN),
        .mode = GPIO_MODE_OUTPUT,
        .pull_down_en = GPIO_PULLDOWN_DISABLE,
        .pull_up_en = GPIO_PULLUP_DISABLE,
    };
    gpio_config(&out_conf);

    // 默认失能电机以降低发热 (EN=1 禁用)
    gpio_set_level(PIN_MOTOR_EN, 1);
    gpio_set_level(PIN_MOTOR_DIR, 0);
    gpio_set_level(PIN_MOTOR_STEP, 0);

    // 2. 配置原点限位微动开关 (GPIO 32)
    gpio_config_t in_conf = {
        .pin_bit_mask = (1ULL << PIN_LIMIT_ORIGIN),
        .mode = GPIO_MODE_INPUT,
        .pull_down_en = GPIO_PULLDOWN_ENABLE,
        .pull_up_en = GPIO_PULLUP_DISABLE,
    };
    gpio_config(&in_conf);

    ESP_LOGI(TAG, "步进电机驱动与原点限位 GPIO 初始化完成");
    return ESP_OK;
}

static volatile bool s_abort_requested = false;

void motor_request_abort(void) {
    s_abort_requested = true;
    gpio_set_level(PIN_MOTOR_EN, 1); // 立即硬件失能断电
    ESP_LOGW(TAG, "⚠️ 电机中止请求已置位，脉冲立即停止");
}

bool motor_is_abort_requested(void) {
    return s_abort_requested;
}

void motor_clear_abort(void) {
    s_abort_requested = false;
}

static bool step_motor(int steps, bool dir) {
    if (s_abort_requested) {
        gpio_set_level(PIN_MOTOR_EN, 1);
        ESP_LOGW(TAG, "电机已被中止，拒绝启动步进脉冲");
        return false;
    }
    gpio_set_level(PIN_MOTOR_EN, 0); // 使能电机
    gpio_set_level(PIN_MOTOR_DIR, dir ? 1 : 0);
    ets_delay_us(50);

    for (int i = 0; i < steps; i++) {
        if (s_abort_requested) {
            gpio_set_level(PIN_MOTOR_EN, 1);
            ESP_LOGW(TAG, "步进脉冲在第 %d 步被中止", i);
            return false;
        }
        gpio_set_level(PIN_MOTOR_STEP, 1);
        ets_delay_us(MOTOR_STEP_DELAY_US);
        gpio_set_level(PIN_MOTOR_STEP, 0);
        ets_delay_us(MOTOR_STEP_DELAY_US);
    }
    return true;
}

bool motor_calibrate_home(void) {
    gpio_set_level(PIN_MOTOR_EN, 0);
    gpio_set_level(PIN_MOTOR_DIR, 0); // 反向回退寻原点
    ets_delay_us(50);

    ESP_LOGI(TAG, "开始原点归零校准...");
    int steps = 0;
    while (gpio_get_level(PIN_LIMIT_ORIGIN) == 0) {
        if (s_abort_requested) {
            gpio_set_level(PIN_MOTOR_EN, 1);
            return false;
        }
        gpio_set_level(PIN_MOTOR_STEP, 1);
        ets_delay_us(MOTOR_STEP_DELAY_US);
        gpio_set_level(PIN_MOTOR_STEP, 0);
        ets_delay_us(MOTOR_STEP_DELAY_US);
        steps++;
        if (steps > MOTOR_MAX_TRAVEL_STEPS) {
            ESP_LOGE(TAG, "原点归零失败: 步数超限，机构可能脱轨或限位损坏");
            gpio_set_level(PIN_MOTOR_EN, 1);
            return false;
        }
    }

    // 微量正转脱离接触点
    step_motor(50, true);
    gpio_set_level(PIN_MOTOR_EN, 1); // 关闭使能防发热
    ESP_LOGI(TAG, "原点归零校准成功 (回退步数=%d)", steps);
    return true;
}

bool motor_dispense_slot(int slot_no) {
    if (slot_no < 1 || slot_no > TOTAL_CABINET_SLOTS) {
        ESP_LOGE(TAG, "非法槽位号: %d", slot_no);
        return false;
    }

    int target_steps = 400 + (slot_no * 600);
    ESP_LOGI(TAG, "执行槽位 #%d 推杆出钥动作 (步数=%d)...", slot_no, target_steps);

    // 1. 正向推进推杆推出钥匙
    if (!step_motor(target_steps, true)) {
        ESP_LOGW(TAG, "出钥推进被中止");
        return false;
    }
    vTaskDelay(pdMS_TO_TICKS(200));

    // 2. 反向拉回复位
    if (!step_motor(target_steps, false)) {
        ESP_LOGW(TAG, "推杆回退被中止");
        return false;
    }

    // 3. 原点闭环校准
    bool home_ok = motor_calibrate_home();
    if (!home_ok) {
        ESP_LOGE(TAG, "出钥后机构归位异常!");
        return false;
    }

    ESP_LOGI(TAG, "槽位 #%d 出钥动作完成，机构已复位", slot_no);
    return true;
}
