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
#include "network_manager.h"
#include "mqtt_client_task.h"

static const char *TAG = "CABINET_MAIN";

void app_main(void) {
    ESP_LOGI(TAG, "==================================================");
    ESP_LOGI(TAG, " 智能钥匙自助存取柜 ESP32 控制器固件启动中... ");
    ESP_LOGI(TAG, " 设备 ID: %s, 固件版本: v1.1.0", CABINET_DEVICE_ID);
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

    // 3. 初始化硬件机电外设驱动
    ESP_LOGI(TAG, "[1/5] 初始化传感器引脚与指示灯...");
    sensor_init();

    ESP_LOGI(TAG, "[2/5] 初始化 RC522 13.56MHz 射频天线...");
    if (rc522_init() != ESP_OK) {
        ESP_LOGE(TAG, "RC522 射频芯片通信失败，请检查 SPI 引脚接线!");
    }

    ESP_LOGI(TAG, "[3/5] 初始化步进电机驱动与原点限位...");
    motor_init();
    motor_calibrate_home();

    // 4. 启动后台传感器周期在位检测与快照任务
    sensor_task_start();

    // 5. 启动无线网络驱动 (Wi-Fi STA / 4G Modem) 并等待网络就绪与 SNTP 对时
    ESP_LOGI(TAG, "[4/5] 启动无线网络协议栈并连接 AP...");
    ESP_ERROR_CHECK(network_manager_init());
    if (network_wait_connected(15000)) {
        ESP_LOGI(TAG, "无线网络就绪，开始连接 MQTT 物联网网关...");
    } else {
        ESP_LOGW(TAG, "网络尚未获得 IP，MQTT 客户端将启动并在后台自动等待重连...");
    }

    // 6. 启动 MQTT 客户端长连接与指令调度
    ESP_LOGI(TAG, "[5/5] 启动 MQTT 通信任务...");
    mqtt_task_init();

    ESP_LOGI(TAG, "✅ ESP32 生产级机电与网络固件已就绪，进入主事件调度状态机。");
}
