#ifndef RFID_TASK_H
#define RFID_TASK_H

#include "app_config.h"
#include <stdbool.h>

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @brief 执行归还操作中的射频防错还校验
 * @param cmd 归还指令上下文
 * @return true 匹配成功且允许完成归还，false 错卡或未检测到标签
 */
bool rfid_verify_return(const cabinet_command_t *cmd);

#ifdef __cplusplus
}
#endif

#endif // RFID_TASK_H
