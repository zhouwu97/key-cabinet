#ifndef PROTOCOL_H
#define PROTOCOL_H

#include "app_config.h"
#include <stdbool.h>

#ifdef __cplusplus
extern "C" {
#endif

// 解析控制指令 JSON
bool protocol_parse_command(const char *topic, const char *payload, cabinet_command_t *out_cmd);

// 构造 ACK 回复报文
char *protocol_build_ack(const char *device_id, const char *reply_msg_id, const char *operation_id, const char *stage);

// 构造 RFID 扫描事件报文
char *protocol_build_rfid_event(const char *device_id, const char *operation_id, int slot_no,
                                const char *expected_rfid, const char *scanned_rfid, bool is_match);

// 构造操作进度与终态报文 (PICKUP_SUCCESS, MOTOR_ARRIVED, RETURN_SUCCESS 等)
char *protocol_build_progress_event(const char *device_id, const char *operation_id,
                                    const char *stage, const char *status,
                                    const char *key_id, int slot_no,
                                    const char *err_code, const char *err_msg);

// 构造心跳报文
char *protocol_build_heartbeat(const char *device_id, const char *status);

// 构造实物在位主动快照上报 (kcab/cab/{deviceId}/status/inventory)
char *protocol_build_inventory_snapshot(const char *device_id, const slot_status_t *slots, int count, bool door_closed);

#ifdef __cplusplus
}
#endif

#endif // PROTOCOL_H
