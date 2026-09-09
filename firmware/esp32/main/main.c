#include <stdio.h>
#include <string.h>
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "esp_system.h"
#include "esp_log.h"
#include "nvs_flash.h"
#include "esp_netif.h"
#include "esp_event.h"

#include "app_config.h"
#include "rc522.h"
#include "motor_task.h"
#include "sensor_task.h"
#include "mqtt_client_task.h"

static const char *TAG = "CABINET_MAIN";

void app_main(void) {
    ESP_LOGI(TAG, "==================================================");
    ESP_LOGI(TAG, " 智能钥匙自助存取柜 ESP32 控制器固件启动中... ");
    ESP_LOGI(TAG, " 设备 ID: %s, 固件版本: v1.0.0", CABINET_DEVICE_ID);
    ESP_LOGI(TAG, "==================================================");

    // 1. 初始化 NVS 闪存
    esp_err_t ret = nvs_flash_init();
    if (ret == ESP_ERR_NVS_NO_FREE_PAGES || ret == ESP_ERR_NVS_NEW_VERSION_FOUND) {
        ESP_ERROR_CHECK(nvs_flash_erase());
        ret = nvs_flash_init();
    }
    ESP_ERROR_CHECK(ret);

    // 2. 初始化网络接口层与系统事件循环
    ESP_ERROR_CHECK(esp_netif_init());
    ESP_ERROR_CHECK(esp_event_loop_create_default());

    // 3. 初始化硬件外设驱动
    ESP_LOGI(TAG, "[1/4] 初始化传感器引脚与指示灯...");
    sensor_init();

    ESP_LOGI(TAG, "[2/4] 初始化 RC522 13.56MHz 射频天线...");
    if (rc522_init() != ESP_OK) {
        ESP_LOGE(TAG, "RC522 射频芯片通信失败，请检查 SPI 引脚接线!");
    }

    ESP_LOGI(TAG, "[3/4] 初始化步进电机驱动与原点限位...");
    motor_init();
    motor_calibrate_home();

    // 4. 启动后台传感器周期在位检测与快照任务
    sensor_task_start();

    // 5. 启动 MQTT 客户端任务与下行指令调度
    ESP_LOGI(TAG, "[4/4] 启动 MQTT 通信任务并连接网关...");
    mqtt_task_init();

    ESP_LOGI(TAG, "✅ ESP32 机电一体化固件初始化完成，进入主调度状态机。");
}
