#include "rc522.h"
#include "app_config.h"
#include <string.h>
#include <stdio.h>
#include "driver/spi_master.h"
#include "driver/gpio.h"
#include "esp_log.h"
#include "rom/ets_sys.h"

static const char *TAG = "RC522";
static spi_device_handle_t s_spi_handle = NULL;

// MFRC522 核心寄存器
#define REG_COMMAND         0x01
#define REG_COM_IEN         0x02
#define REG_DIV_IEN         0x03
#define REG_COM_IRQ         0x04
#define REG_DIV_IRQ         0x05
#define REG_ERROR           0x06
#define REG_STATUS1         0x07
#define REG_STATUS2         0x08
#define REG_FIFO_DATA       0x09
#define REG_FIFO_LEVEL      0x0A
#define REG_CONTROL         0x0C
#define REG_BIT_FRAMING     0x0D
#define REG_COLL            0x0E
#define REG_MODE            0x11
#define REG_TX_MODE         0x12
#define REG_RX_MODE         0x13
#define REG_TX_CONTROL      0x14
#define REG_TX_ASK          0x15
#define REG_T_MODE          0x2A
#define REG_T_PRESCALER     0x2B
#define REG_T_RELOAD_H      0x2C
#define REG_T_RELOAD_L      0x2D
#define REG_VERSION         0x37

// MFRC522 常用指令
#define CMD_IDLE            0x00
#define CMD_TRANSCEIVE      0x0C
#define CMD_SOFT_RESET      0x0F

// PICC 寻卡防冲突指令
#define PICC_CMD_REQA       0x26
#define PICC_CMD_SEL_CL1    0x93

static esp_err_t rc522_write_reg(uint8_t reg, uint8_t val) {
    uint8_t tx_data[2] = { (uint8_t)((reg << 1) & 0x7E), val };
    spi_transaction_t t = {
        .length = 16,
        .tx_buffer = tx_data,
    };
    return spi_device_transmit(s_spi_handle, &t);
}

static uint8_t rc522_read_reg(uint8_t reg) {
    uint8_t tx_data[2] = { (uint8_t)(((reg << 1) & 0x7E) | 0x80), 0x00 };
    uint8_t rx_data[2] = { 0 };
    spi_transaction_t t = {
        .length = 16,
        .tx_buffer = tx_data,
        .rx_buffer = rx_data,
    };
    spi_device_transmit(s_spi_handle, &t);
    return rx_data[1];
}

static void rc522_set_bit_mask(uint8_t reg, uint8_t mask) {
    uint8_t current = rc522_read_reg(reg);
    rc522_write_reg(reg, current | mask);
}

static void rc522_clear_bit_mask(uint8_t reg, uint8_t mask) {
    uint8_t current = rc522_read_reg(reg);
    rc522_write_reg(reg, current & (~mask));
}

static void rc522_antenna_on(void) {
    uint8_t val = rc522_read_reg(REG_TX_CONTROL);
    if ((val & 0x03) != 0x03) {
        rc522_set_bit_mask(REG_TX_CONTROL, 0x03);
    }
}

void rc522_antenna_off(void) {
    rc522_clear_bit_mask(REG_TX_CONTROL, 0x03);
}

esp_err_t rc522_init(void) {
    // 1. 配置 RST 复位引脚
    gpio_config_t rst_conf = {
        .pin_bit_mask = (1ULL << PIN_RC522_RST),
        .mode = GPIO_MODE_OUTPUT,
        .pull_down_en = GPIO_PULLDOWN_DISABLE,
        .pull_up_en = GPIO_PULLUP_ENABLE,
    };
    gpio_config(&rst_conf);
    gpio_set_level(PIN_RC522_RST, 0);
    ets_delay_us(100);
    gpio_set_level(PIN_RC522_RST, 1);
    ets_delay_us(500);

    // 2. 初始化 SPI 总线配置
    spi_bus_config_t bus_cfg = {
        .miso_io_num = PIN_RC522_MISO,
        .mosi_io_num = PIN_RC522_MOSI,
        .sclk_io_num = PIN_RC522_SCK,
        .quadwp_io_num = -1,
        .quadhd_io_num = -1,
        .max_transfer_sz = 64,
    };
    esp_err_t ret = spi_bus_initialize(SPI2_HOST, &bus_cfg, SPI_DMA_CH_AUTO);
    if (ret != ESP_OK && ret != ESP_ERR_INVALID_STATE) {
        ESP_LOGE(TAG, "Failed to init SPI bus: %s", esp_err_to_name(ret));
        return ret;
    }

    spi_device_interface_config_t dev_cfg = {
        .clock_speed_hz = 5 * 1000 * 1000, // 5 MHz
        .mode = 0,
        .spics_io_num = PIN_RC522_CS,
        .queue_size = 4,
    };
    ret = spi_bus_add_device(SPI2_HOST, &dev_cfg, &s_spi_handle);
    if (ret != ESP_OK) {
        ESP_LOGE(TAG, "Failed to attach RC522 SPI device: %s", esp_err_to_name(ret));
        return ret;
    }

    // 3. 复位与寄存器初始化
    rc522_write_reg(REG_COMMAND, CMD_SOFT_RESET);
    ets_delay_us(50000); // 50ms 等待复位完成

    rc522_write_reg(REG_T_MODE, 0x8D);
    rc522_write_reg(REG_T_PRESCALER, 0x3E);
    rc522_write_reg(REG_T_RELOAD_L, 30);
    rc522_write_reg(REG_T_RELOAD_H, 0);
    rc522_write_reg(REG_TX_ASK, 0x40); // 强制 100% ASK 调制
    rc522_write_reg(REG_MODE, 0x3D);   // CRC 初始值 0x6363

    rc522_antenna_on();

    uint8_t version = rc522_read_reg(REG_VERSION);
    ESP_LOGI(TAG, "RC522 射频芯片初始化成功, 芯片版本固件: 0x%02X", version);
    return ESP_OK;
}

static bool rc522_to_card(uint8_t command, const uint8_t *send_data, size_t send_len,
                          uint8_t *back_data, size_t *back_len, uint8_t *valid_bits) {
    uint8_t irq_en = 0x00;
    uint8_t wait_irq = 0x00;

    if (command == CMD_TRANSCEIVE) {
        irq_en = 0x77;
        wait_irq = 0x30; // RxIRQ | IdleIRQ
    }

    rc522_write_reg(REG_COM_IEN, irq_en | 0x80);
    rc522_clear_bit_mask(REG_COM_IRQ, 0x80);
    rc522_set_bit_mask(REG_FIFO_LEVEL, 0x80); // 清空 FIFO
    rc522_write_reg(REG_COMMAND, CMD_IDLE);

    for (size_t i = 0; i < send_len; i++) {
        rc522_write_reg(REG_FIFO_DATA, send_data[i]);
    }

    rc522_write_reg(REG_COMMAND, command);
    if (command == CMD_TRANSCEIVE) {
        rc522_set_bit_mask(REG_BIT_FRAMING, 0x80); // 启动发送
    }

    // 等待传输完成或超时 (最多 25ms)
    uint32_t count = 2000;
    uint8_t n = 0;
    do {
        n = rc522_read_reg(REG_COM_IRQ);
        count--;
        ets_delay_us(10);
    } while ((count > 0) && !(n & 0x01) && !(n & wait_irq));

    rc522_clear_bit_mask(REG_BIT_FRAMING, 0x80);

    if (count == 0 || (n & 0x01)) { // 超时或发生错误
        return false;
    }

    uint8_t error_reg = rc522_read_reg(REG_ERROR);
    if (error_reg & 0x1B) { // BufferOvfl, ParityErr, ProtocolErr
        return false;
    }

    if (back_data && back_len) {
        uint8_t fifo_len = rc522_read_reg(REG_FIFO_LEVEL);
        if (fifo_len > *back_len) {
            fifo_len = *back_len;
        }
        for (uint8_t i = 0; i < fifo_len; i++) {
            back_data[i] = rc522_read_reg(REG_FIFO_DATA);
        }
        *back_len = fifo_len;
    }

    if (valid_bits) {
        *valid_bits = rc522_read_reg(REG_CONTROL) & 0x07;
    }

    return true;
}

bool rc522_read_uid(char *out_uid_hex, size_t max_len) {
    if (!out_uid_hex || max_len < 9) {
        return false;
    }

    // 1. 发送 REQA 寻卡 (ISO14443A)
    rc522_write_reg(REG_BIT_FRAMING, 0x07);
    uint8_t req_cmd = PICC_CMD_REQA;
    uint8_t atqa[2] = {0};
    size_t atqa_len = sizeof(atqa);
    if (!rc522_to_card(CMD_TRANSCEIVE, &req_cmd, 1, atqa, &atqa_len, NULL)) {
        return false;
    }

    // 2. 防冲突循环读取 UID Cascade Level 1
    rc522_write_reg(REG_BIT_FRAMING, 0x00);
    uint8_t anticoll_cmd[2] = { PICC_CMD_SEL_CL1, 0x20 };
    uint8_t uid_buffer[5] = {0};
    size_t uid_len = sizeof(uid_buffer);
    if (!rc522_to_card(CMD_TRANSCEIVE, anticoll_cmd, 2, uid_buffer, &uid_len, NULL)) {
        return false;
    }

    // 3. 校验 XOR 校验和
    uint8_t checksum = uid_buffer[0] ^ uid_buffer[1] ^ uid_buffer[2] ^ uid_buffer[3];
    if (checksum != uid_buffer[4]) {
        ESP_LOGW(TAG, "RFID 卡片校验和错误 (XOR 校验失败)");
        return false;
    }

    // 格式化输出为 8 字符大写十六进制字符串 (如 "04A1B2C3")
    snprintf(out_uid_hex, max_len, "%02X%02X%02X%02X",
             uid_buffer[0], uid_buffer[1], uid_buffer[2], uid_buffer[3]);
    return true;
}
