#ifndef MQTT_CLIENT_TASK_H
#define MQTT_CLIENT_TASK_H

#include "app_config.h"
#include "esp_err.h"

#ifdef __cplusplus
extern "C" {
#endif

esp_err_t mqtt_task_init(void);

void mqtt_publish_status(const char *subtopic, const char *payload);

void mqtt_publish_event(const char *subtopic, const char *payload);

#ifdef __cplusplus
}
#endif

#endif // MQTT_CLIENT_TASK_H
