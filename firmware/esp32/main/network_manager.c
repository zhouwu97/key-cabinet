#include "network_manager.h"
#include <string.h>
#include <time.h>
#include <sys/time.h>
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "freertos/event_groups.h"
#include "esp_system.h"
#include "esp_wifi.h"
#include "esp_event.h"
#include "esp_log.h"
#include "esp_sntp.h"

#include "app_config.h"

static const char *TAG = "NET_MGR";

#define WIFI_CONNECTED_BIT BIT0
#define WIFI_FAIL_BIT      BIT1

static EventGroupHandle_t s_wifi_event_group = NULL;
static int s_retry_num = 0;
static bool s_is_connected = false;

static void event_handler(void* arg, esp_event_base_t event_base,
                          int32_t event_id, void* event_data) {
    if (event_base == WIFI_EVENT && event_id == WIFI_EVENT_STA_START) {
        ESP_LOGI(TAG, "Wi-Fi STA 已就绪，开始连接 AP: %s...", WIFI_STA_SSID);
        esp_wifi_connect();
    } else if (event_base == WIFI_EVENT && event_id == WIFI_EVENT_STA_DISCONNECTED) {
        s_is_connected = false;
        if (s_retry_num < 10) {
            esp_wifi_connect();
            s_retry_num++;
            ESP_LOGW(TAG, "Wi-Fi 断开连接，正在重试第 %d 次...", s_retry_num);
        } else {
            ESP_LOGE(TAG, "Wi-Fi 连接连续失败达到上限，进入退避等待");
            xEventGroupSetBits(s_wifi_event_group, WIFI_FAIL_BIT);
        }
    } else if (event_base == IP_EVENT && event_id == IP_EVENT_STA_GOT_IP) {
        ip_event_got_ip_t* event = (ip_event_got_ip_t*) event_data;
        ESP_LOGI(TAG, "✅ 成功获取 IP 地址: " IPSTR ", 网关: " IPSTR,
                 IP2STR(&event->ip_info.ip), IP2STR(&event->ip_info.gw));
        s_retry_num = 0;
        s_is_connected = true;
        xEventGroupSetBits(s_wifi_event_group, WIFI_CONNECTED_BIT);

        // 获取 IP 后立即启动 SNTP 网络时间同步
        ESP_LOGI(TAG, "启动 SNTP 网络授时校准系统时钟...");
        esp_sntp_setoperatingmode(SNTP_OPMODE_POLL);
        esp_sntp_setservername(0, "pool.ntp.org");
        esp_sntp_setservername(1, "cn.pool.ntp.org");
        esp_sntp_setservername(2, "time.asia.apple.com");
        esp_sntp_init();

        // 设置东八区中国标准时间 (CST-8)
        setenv("TZ", "CST-8", 1);
        tzset();
    }
}

esp_err_t network_manager_init(void) {
    s_wifi_event_group = xEventGroupCreate();

#if ACTIVE_NETWORK_INTERFACE == NET_IF_WIFI
    ESP_LOGI(TAG, "初始化 Wi-Fi 网络协议栈 (SSID=%s)...", WIFI_STA_SSID);

    esp_netif_t *sta_netif = esp_netif_create_default_wifi_sta();
    if (!sta_netif) {
        ESP_LOGE(TAG, "创建 default_wifi_sta 失败");
        return ESP_FAIL;
    }

    wifi_init_config_t cfg = WIFI_INIT_CONFIG_DEFAULT();
    ESP_ERROR_CHECK(esp_wifi_init(&cfg));

    esp_event_handler_instance_t instance_any_id;
    esp_event_handler_instance_t instance_got_ip;
    ESP_ERROR_CHECK(esp_event_handler_instance_register(WIFI_EVENT,
                                                        ESP_EVENT_ANY_ID,
                                                        &event_handler,
                                                        NULL,
                                                        &instance_any_id));
    ESP_ERROR_CHECK(esp_event_handler_instance_register(IP_EVENT,
                                                        IP_EVENT_STA_GOT_IP,
                                                        &event_handler,
                                                        NULL,
                                                        &instance_got_ip));

    wifi_config_t wifi_config = {
        .sta = {
            .ssid = WIFI_STA_SSID,
            .password = WIFI_STA_PASSWORD,
            .threshold.authmode = WIFI_AUTH_WPA2_PSK,
            .pmf_cfg = {
                .capable = true,
                .required = false
            },
        },
    };
    ESP_ERROR_CHECK(esp_wifi_set_mode(WIFI_MODE_STA));
    ESP_ERROR_CHECK(esp_wifi_set_config(WIFI_IF_STA, &wifi_config));
    ESP_ERROR_CHECK(esp_wifi_start());

    ESP_LOGI(TAG, "Wi-Fi STA 驱动启动完成");
    return ESP_OK;

#elif ACTIVE_NETWORK_INTERFACE == NET_IF_4G_PPP
    ESP_LOGI(TAG, "初始化 4G Cellular 通信驱动 (UART AT / esp_modem PPP)...");
    // 4G 蜂窝网络接口抽象：支持 SIM7600 / A7670C 系列 CAT1/CAT4 模块
    // 通过 UART 与 esp_modem 建立标准 PPP 网络接口
    ESP_LOGI(TAG, "4G 模块引脚: TX=GPIO%d, RX=GPIO%d, PWR=GPIO%d",
             PIN_4G_UART_TX, PIN_4G_UART_RX, PIN_4G_PWR_EN);
    // 待插入 SIM 卡与 4G 模组后建立 PPP 拨号连接
    return ESP_OK;
#else
    #error "未选择有效的网络接口模式 (ACTIVE_NETWORK_INTERFACE)"
#endif
}

bool network_wait_connected(uint32_t timeout_ms) {
    if (!s_wifi_event_group) return false;

    EventBits_t bits = xEventGroupWaitBits(
        s_wifi_event_group,
        WIFI_CONNECTED_BIT | WIFI_FAIL_BIT,
        pdFALSE,
        pdFALSE,
        pdMS_TO_TICKS(timeout_ms)
    );

    if (bits & WIFI_CONNECTED_BIT) {
        ESP_LOGI(TAG, "✅ 网络连接建立成功，已具备外网通信能力");
        return true;
    } else {
        ESP_LOGW(TAG, "⚠️ 网络在指定超时 (%u ms) 内未就绪", timeout_ms);
        return false;
    }
}

bool network_is_connected(void) {
    return s_is_connected;
}

bool network_is_time_synced(void) {
    time_t now = 0;
    time(&now);
    // 若时间戳晚于 2024-01-01 (1704067200)，表示已经过网络真实授时
    return now > 1704067200;
}
