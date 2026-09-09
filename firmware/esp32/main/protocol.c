#include "protocol.h"
#include <string.h>
#include <stdio.h>
#include <time.h>
#include "cJSON.h"
#include "esp_log.h"
#include "esp_timer.h"

#include <sys/time.h>

static const char *TAG = "PROTOCOL";

static int64_t get_timestamp_ms(void) {
    struct timeval tv;
    gettimeofday(&tv, NULL);
    // 若尚未通过 SNTP 网络校时 (系统时钟仍处于 1970 年)，返回 0
    // 由服务端收到消息时以可靠接收时间替代，杜绝时钟混乱
    if (tv.tv_sec < 1704067200) {
        return 0;
    }
    return ((int64_t)tv.tv_sec * 1000) + (tv.tv_usec / 1000);
}

bool protocol_parse_command(const char *topic, const char *payload, cabinet_command_t *out_cmd) {
    if (!topic || !payload || !out_cmd) {
        return false;
    }
    memset(out_cmd, 0, sizeof(cabinet_command_t));

    if (strstr(topic, "/cmd/pickup")) {
        out_cmd->type = CMD_TYPE_PICKUP;
    } else if (strstr(topic, "/cmd/return")) {
        out_cmd->type = CMD_TYPE_RETURN;
    } else if (strstr(topic, "/cmd/abort")) {
        out_cmd->type = CMD_TYPE_ABORT;
    } else {
        return false;
    }

    cJSON *root = cJSON_Parse(payload);
    if (!root) {
        ESP_LOGE(TAG, "Failed to parse JSON payload");
        return false;
    }

    cJSON *msg_id = cJSON_GetObjectItem(root, "msgId");
    if (cJSON_IsString(msg_id) && msg_id->valuestring) {
        strncpy(out_cmd->msg_id, msg_id->valuestring, sizeof(out_cmd->msg_id) - 1);
    }

    cJSON *data = cJSON_GetObjectItem(root, "data");
    if (cJSON_IsObject(data)) {
        cJSON *op_id = cJSON_GetObjectItem(data, "operationId");
        if (cJSON_IsString(op_id) && op_id->valuestring) {
            strncpy(out_cmd->operation_id, op_id->valuestring, sizeof(out_cmd->operation_id) - 1);
        }

        cJSON *slot = cJSON_GetObjectItem(data, "slotNo");
        if (!slot) {
            slot = cJSON_GetObjectItem(data, "targetSlotNo");
        }
        if (cJSON_IsNumber(slot)) {
            out_cmd->slot_no = slot->valueint;
        }

        cJSON *key = cJSON_GetObjectItem(data, "keyId");
        if (!key) {
            key = cJSON_GetObjectItem(data, "expectedKeyId");
        }
        if (cJSON_IsString(key) && key->valuestring) {
            strncpy(out_cmd->key_id, key->valuestring, sizeof(out_cmd->key_id) - 1);
        }

        cJSON *rfid = cJSON_GetObjectItem(data, "expectedRfidTag");
        if (!rfid) {
            rfid = cJSON_GetObjectItem(data, "expectedRfid");
        }
        if (cJSON_IsString(rfid) && rfid->valuestring) {
            strncpy(out_cmd->expected_rfid, rfid->valuestring, sizeof(out_cmd->expected_rfid) - 1);
        }

        cJSON *timeout = cJSON_GetObjectItem(data, "timeoutSeconds");
        if (cJSON_IsNumber(timeout)) {
            out_cmd->timeout_sec = timeout->valueint;
        } else {
            out_cmd->timeout_sec = 60;
        }
    }

    cJSON_Delete(root);
    return true;
}

char *protocol_build_ack(const char *device_id, const char *reply_msg_id, const char *operation_id, const char *stage) {
    cJSON *root = cJSON_CreateObject();
    char msg_id_buf[64];
    snprintf(msg_id_buf, sizeof(msg_id_buf), "ack_%lld", get_timestamp_ms());

    cJSON_AddStringToObject(root, "msgId", msg_id_buf);
    cJSON_AddNumberToObject(root, "timestamp", (double)get_timestamp_ms());
    cJSON_AddStringToObject(root, "deviceId", device_id);
    if (reply_msg_id && strlen(reply_msg_id) > 0) {
        cJSON_AddStringToObject(root, "replyMsgId", reply_msg_id);
    }
    cJSON_AddStringToObject(root, "status", "SUCCESS");

    cJSON *data = cJSON_CreateObject();
    if (operation_id) cJSON_AddStringToObject(data, "operationId", operation_id);
    if (stage) cJSON_AddStringToObject(data, "stage", stage);
    cJSON_AddItemToObject(root, "data", data);

    char *out = cJSON_PrintUnformatted(root);
    cJSON_Delete(root);
    return out;
}

char *protocol_build_rfid_event(const char *device_id, const char *operation_id, int slot_no,
                                const char *expected_rfid, const char *scanned_rfid, bool is_match) {
    cJSON *root = cJSON_CreateObject();
    char msg_id_buf[64];
    snprintf(msg_id_buf, sizeof(msg_id_buf), "rfid_%lld", get_timestamp_ms());

    cJSON_AddStringToObject(root, "msgId", msg_id_buf);
    cJSON_AddNumberToObject(root, "timestamp", (double)get_timestamp_ms());
    cJSON_AddStringToObject(root, "deviceId", device_id);
    cJSON_AddStringToObject(root, "status", is_match ? "SUCCESS" : "FAILED");

    cJSON *data = cJSON_CreateObject();
    cJSON_AddStringToObject(data, "operationId", operation_id ? operation_id : "");
    cJSON_AddNumberToObject(data, "targetSlotNo", slot_no);
    cJSON_AddStringToObject(data, "stage", "rfid_scanned");
    cJSON_AddBoolToObject(data, "isMatch", is_match);
    cJSON_AddStringToObject(data, "expectedRfidTag", expected_rfid ? expected_rfid : "");
    cJSON_AddStringToObject(data, "scannedRfid", scanned_rfid ? scanned_rfid : "");
    cJSON_AddStringToObject(data, "uid", scanned_rfid ? scanned_rfid : "");
    cJSON_AddItemToObject(root, "data", data);

    char *out = cJSON_PrintUnformatted(root);
    cJSON_Delete(root);
    return out;
}

char *protocol_build_progress_event(const char *device_id, const char *operation_id,
                                    const char *stage, const char *status,
                                    const char *key_id, int slot_no,
                                    const char *err_code, const char *err_msg) {
    cJSON *root = cJSON_CreateObject();
    char msg_id_buf[64];
    snprintf(msg_id_buf, sizeof(msg_id_buf), "evt_%lld", get_timestamp_ms());

    cJSON_AddStringToObject(root, "msgId", msg_id_buf);
    cJSON_AddNumberToObject(root, "timestamp", (double)get_timestamp_ms());
    cJSON_AddStringToObject(root, "deviceId", device_id);
    cJSON_AddStringToObject(root, "status", status ? status : "SUCCESS");
    if (err_code) cJSON_AddStringToObject(root, "errorCode", err_code);
    if (err_msg) cJSON_AddStringToObject(root, "errorMessage", err_msg);

    cJSON *data = cJSON_CreateObject();
    if (operation_id) cJSON_AddStringToObject(data, "operationId", operation_id);
    if (stage) cJSON_AddStringToObject(data, "stage", stage);
    if (key_id && strlen(key_id) > 0) cJSON_AddStringToObject(data, "keyId", key_id);
    if (slot_no > 0) cJSON_AddNumberToObject(data, "slotNo", slot_no);
    cJSON_AddItemToObject(root, "data", data);

    char *out = cJSON_PrintUnformatted(root);
    cJSON_Delete(root);
    return out;
}

char *protocol_build_heartbeat(const char *device_id, const char *status) {
    cJSON *root = cJSON_CreateObject();
    cJSON_AddStringToObject(root, "deviceId", device_id);
    cJSON_AddStringToObject(root, "status", status ? status : "ONLINE");
    cJSON_AddNumberToObject(root, "timestamp", (double)get_timestamp_ms());

    char *out = cJSON_PrintUnformatted(root);
    cJSON_Delete(root);
    return out;
}

char *protocol_build_inventory_snapshot(const char *device_id, const slot_status_t *slots, int count, bool door_closed) {
    cJSON *root = cJSON_CreateObject();
    cJSON_AddStringToObject(root, "deviceId", device_id);
    cJSON_AddNumberToObject(root, "timestamp", (double)get_timestamp_ms());
    cJSON_AddBoolToObject(root, "doorClosed", door_closed);

    cJSON *arr = cJSON_CreateArray();
    for (int i = 0; i < count; i++) {
        cJSON *s = cJSON_CreateObject();
        cJSON_AddNumberToObject(s, "slotNo", slots[i].slot_no);
        cJSON_AddBoolToObject(s, "presence", slots[i].presence);
        if (slots[i].presence && strlen(slots[i].rfid_tag) > 0) {
            cJSON_AddStringToObject(s, "rfid", slots[i].rfid_tag);
        }
        cJSON_AddItemToArray(arr, s);
    }
    cJSON_AddItemToObject(root, "slots", arr);

    char *out = cJSON_PrintUnformatted(root);
    cJSON_Delete(root);
    return out;
}
