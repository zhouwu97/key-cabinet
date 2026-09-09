#ifndef MOTOR_TASK_H
#define MOTOR_TASK_H

#include "app_config.h"
#include "esp_err.h"

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @brief 初始化步进电机驱动与限位开关 GPIO
 */
esp_err_t motor_init(void);

/**
 * @brief 电机机构原点归零校准
 */
bool motor_calibrate_home(void);

/**
 * @brief 执行槽位出钥推杆动作
 * @param slot_no 槽位编号 (1 ~ TOTAL_CABINET_SLOTS)
 * @return true 成功推出并复位，false 机构卡死或限位超时
 */
bool motor_dispense_slot(int slot_no);

/**
 * @brief 启动电机控制 FreeRTOS 任务
 */
void motor_task_start(void);

#ifdef __cplusplus
}
#endif

#endif // MOTOR_TASK_H
