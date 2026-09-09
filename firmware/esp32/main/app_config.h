#ifndef APP_CONFIG_H
#define APP_CONFIG_H

#include <stdint.h>
#include <stdbool.h>

#ifdef __cplusplus
extern "C" {
#endif

// ==========================================
// 1. 设备与网络通信配置
// ==========================================
#define CABINET_DEVICE_ID       "CAB001"
#define MQTT_TOPIC_PREFIX       "kcab"
#define MQTT_DEFAULT_BROKER_URI "mqtt://192.168.1.100:1883"
#define MQTT_USERNAME           "cab_device_CAB001"
#define MQTT_PASSWORD           "cab_secret_mqtt_pass"
#define MQTT_USE_TLS            0   // 置 1 时启用 mqtts:// 与 TLS CA 证书挂载
#define HEARTBEAT_INTERVAL_SEC  25
#define INVENTORY_INTERVAL_SEC  60

// 网络接口模式切换
#define NET_IF_WIFI             1
#define NET_IF_4G_PPP           2
#define ACTIVE_NETWORK_INTERFACE NET_IF_WIFI

// Wi-Fi STA 配置
#define WIFI_STA_SSID           "KeyCabinet_AP"
#define WIFI_STA_PASSWORD       "CabinetPass888"

// 4G Cat.1 / Cat.4 模块串口引脚 (SIM7600CE / A7670C)
#define PIN_4G_UART_TX          17
#define PIN_4G_UART_RX          16
#define PIN_4G_PWR_EN           13  // 独立电源使能 GPIO (避免与 GPIO4 蜂鸣器冲突)

// ==========================================
// 2. 硬件 GPIO 管脚分配
// ==========================================
// RC522 RFID 读卡器 (SPI 总线)
#define PIN_RC522_SCK           18
#define PIN_RC522_MISO          19
#define PIN_RC522_MOSI          23
#define PIN_RC522_CS            5
#define PIN_RC522_RST           22

// 步进电机驱动 (A4988 / TMC2209)
#define PIN_MOTOR_STEP          26
#define PIN_MOTOR_DIR           27
#define PIN_MOTOR_EN            14

// 传感器与限位微动
#define PIN_LIMIT_ORIGIN        32  // 下拉输入：原点归位限位
#define PIN_SLOT1_PRESENCE      33  // 内部上拉：槽位1钥匙在位
#define PIN_SLOT2_PRESENCE      25  // 内部上拉：槽位2钥匙在位
#define PIN_DOOR_MAGNET         35  // 纯输入：柜门磁开关 (硬件外部接10kΩ上拉电阻至3.3V)

// 指示与报警
#define PIN_STATUS_LED          2   // 板载状态指示蓝灯
#define PIN_BUZZER              4   // 蜂鸣器提示

// ==========================================
// 3. 机构控制参数
// ==========================================
#define MOTOR_STEPS_PER_REV     200
#define MOTOR_MAX_TRAVEL_STEPS  3000
#define MOTOR_STEP_DELAY_US     800
#define TOTAL_CABINET_SLOTS     2

// ==========================================
// 4. 事件与指令数据结构
// ==========================================
typedef enum {
    CMD_TYPE_PICKUP = 1,
    CMD_TYPE_RETURN,
    CMD_TYPE_ABORT,
} cabinet_cmd_type_t;

typedef struct {
    cabinet_cmd_type_t type;
    char msg_id[64];
    char operation_id[64];
    int slot_no;
    char key_id[64];
    char expected_rfid[64];
    int timeout_sec;
} cabinet_command_t;

typedef struct {
    int slot_no;
    bool presence;
    char rfid_tag[32];
} slot_status_t;

#ifdef __cplusplus
}
#endif

#endif // APP_CONFIG_H
