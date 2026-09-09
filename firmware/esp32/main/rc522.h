#ifndef RC522_H
#define RC522_H

#include <stdint.h>
#include <stdbool.h>
#include <stddef.h>
#include "esp_err.h"

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @brief 初始化 RC522 射频天线与 SPI 外设
 */
esp_err_t rc522_init(void);

/**
 * @brief 寻卡并读取 13.56MHz RFID 卡片 UID
 * @param out_uid_hex 输出十六进制格式的卡片 UID 字符串 (至少需要 16 字节缓冲区)
 * @param max_len 缓冲区最大长度
 * @return true 成功读取到卡片 UID，false 未检测到卡片或读取失败
 */
bool rc522_read_uid(char *out_uid_hex, size_t max_len);

/**
 * @brief 关闭 RC522 射频天线以降低功耗
 */
void rc522_antenna_off(void);

#ifdef __cplusplus
}
#endif

#endif // RC522_H
