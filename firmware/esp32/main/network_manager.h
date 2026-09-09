#ifndef NETWORK_MANAGER_H
#define NETWORK_MANAGER_H

#include <stdbool.h>
#include <stdint.h>
#include "esp_err.h"

#ifdef __cplusplus
extern "C" {
#endif

/**
 * 初始化网络管理器：
 * 1. 建立 Wi-Fi STA 网络接口
 * 2. 注册网络事件与 IP 获得回调
 * 3. 连接接入点并在连接成功后自动启动 SNTP 网络校时
 */
esp_err_t network_manager_init(void);

/**
 * 阻塞等待网络建立并获取有效 IP 地址
 * @param timeout_ms 超时毫秒数 (如 30000)
 * @return true 表示已联网就绪，false 表示超时未就绪
 */
bool network_wait_connected(uint32_t timeout_ms);

/**
 * 查询当前网络是否已连接
 */
bool network_is_connected(void);

/**
 * 查询系统时间是否已成功通过 SNTP 网络校时 (RTC 经过 2024 年)
 */
bool network_is_time_synced(void);

#ifdef __cplusplus
}
#endif

#endif // NETWORK_MANAGER_H
