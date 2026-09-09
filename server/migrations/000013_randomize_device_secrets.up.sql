-- 000013_randomize_device_secrets.up.sql
-- 重置机柜通信密钥为高熵不可预测的 32 字节 (64 位十六进制) 真实 CSPRNG 字符串，杜绝可预测伪随机风险

CREATE EXTENSION IF NOT EXISTS pgcrypto;

UPDATE devices
SET device_secret = encode(gen_random_bytes(32), 'hex')
WHERE device_secret IS NULL 
   OR device_secret = '' 
   OR device_secret LIKE 'cab_sec_%';

