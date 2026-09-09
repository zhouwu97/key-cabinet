#ifndef SENSOR_TASK_H
#define SENSOR_TASK_H

#include "app_config.h"
#include <stdbool.h>

#ifdef __cplusplus
extern "C" {
#endif

void sensor_init(void);

// 获取槽位在位实时状态
bool sensor_get_slot_presence(int slot_no);

// 获取柜门开闭状态 (true=已闭合, false=开门)
bool sensor_is_door_closed(void);

// 获取整柜最新在位与卡片状态快照
void sensor_get_all_slots(slot_status_t *out_slots, int *out_count);

// 启动传感器周期监控与主动盘点任务
void sensor_task_start(void);

#ifdef __cplusplus
}
#endif

#endif // SENSOR_TASK_H
