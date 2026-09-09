-- 000013_randomize_device_secrets.up.sql
-- 重置机柜通信密钥为高熵不可预测的 64 位 CSPRNG 十六进制字符串，杜绝 cab_sec_CAB001 可预测风险

UPDATE devices
SET device_secret = encode(sha256((random()::text || clock_timestamp()::text || id || 'cabinet_device_secret_salt')::bytea), 'hex')
WHERE device_secret IS NULL 
   OR device_secret = '' 
   OR device_secret LIKE 'cab_sec_%';
