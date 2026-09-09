package middleware

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/zhouwu97/key-cabinet/server/internal/platform/jwt"
	"github.com/zhouwu97/key-cabinet/server/internal/repository"
)

type mockDeviceRepository struct {
	devices map[string]*repository.Device
}

func (m *mockDeviceRepository) FindByID(_ context.Context, id string) (*repository.Device, error) {
	return m.devices[id], nil
}

func (m *mockDeviceRepository) FindAll(_ context.Context) ([]*repository.Device, error) {
	return nil, nil
}

func TestCabinetAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secret := "test_secret_key_123"
	repo := &mockDeviceRepository{
		devices: map[string]*repository.Device{
			"CAB001": {
				ID:           "CAB001",
				Status:       "ONLINE",
				DeviceSecret: secret,
			},
			"CAB_OFFLINE": {
				ID:           "CAB_OFFLINE",
				Status:       "OFFLINE",
				DeviceSecret: secret,
			},
			"CAB_NO_SECRET": {
				ID:           "CAB_NO_SECRET",
				Status:       "ONLINE",
				DeviceSecret: "",
			},
		},
	}

	setupTestServer := func() *gin.Engine {
		r := gin.New()
		r.Use(CabinetAuthMiddleware(repo))
		r.POST("/api/v1/cabinet/test", func(c *gin.Context) {
			devID, _ := c.Get("cabinet_device_id")
			c.JSON(http.StatusOK, gin.H{"status": "ok", "deviceId": devID})
		})
		return r
	}

	// 1. 缺失认证头拦截
	t.Run("Missing headers rejected", func(t *testing.T) {
		r := setupTestServer()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cabinet/test", bytes.NewBufferString(`{"test":1}`))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	// 2. 时间戳过期拦截 (超过 300 秒)
	t.Run("Expired timestamp rejected", func(t *testing.T) {
		r := setupTestServer()
		expiredTs := fmt.Sprintf("%d", time.Now().Add(-10*time.Minute).Unix())
		nonce := "nonce_expired_1"
		body := []byte(`{"test":1}`)
		sig := GenerateCabinetSignature(secret, http.MethodPost, "/api/v1/cabinet/test", expiredTs, nonce, body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/cabinet/test", bytes.NewBuffer(body))
		req.Header.Set("X-Cabinet-ID", "CAB001")
		req.Header.Set("X-Timestamp", expiredTs)
		req.Header.Set("X-Nonce", nonce)
		req.Header.Set("X-Signature", sig)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	// 3. 错误签名拦截
	t.Run("Invalid signature rejected", func(t *testing.T) {
		r := setupTestServer()
		ts := fmt.Sprintf("%d", time.Now().Unix())
		nonce := "nonce_invalid_sig"
		body := []byte(`{"test":1}`)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/cabinet/test", bytes.NewBuffer(body))
		req.Header.Set("X-Cabinet-ID", "CAB001")
		req.Header.Set("X-Timestamp", ts)
		req.Header.Set("X-Nonce", nonce)
		req.Header.Set("X-Signature", "wrong_signature_hex")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	// 4. 离线机柜拦截
	t.Run("Offline device rejected", func(t *testing.T) {
		r := setupTestServer()
		ts := fmt.Sprintf("%d", time.Now().Unix())
		nonce := "nonce_offline"
		body := []byte(`{"test":1}`)
		sig := GenerateCabinetSignature(secret, http.MethodPost, "/api/v1/cabinet/test", ts, nonce, body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/cabinet/test", bytes.NewBuffer(body))
		req.Header.Set("X-Cabinet-ID", "CAB_OFFLINE")
		req.Header.Set("X-Timestamp", ts)
		req.Header.Set("X-Nonce", nonce)
		req.Header.Set("X-Signature", sig)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusForbidden, w.Code)
	})

	// 5. 正确签名与时间戳通过认证
	t.Run("Valid HMAC signature accepted", func(t *testing.T) {
		r := setupTestServer()
		ts := fmt.Sprintf("%d", time.Now().Unix())
		nonce := "nonce_valid_123"
		body := []byte(`{"studentNo":"20230001"}`)
		sig := GenerateCabinetSignature(secret, http.MethodPost, "/api/v1/cabinet/test", ts, nonce, body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/cabinet/test", bytes.NewBuffer(body))
		req.Header.Set("X-Cabinet-ID", "CAB001")
		req.Header.Set("X-Timestamp", ts)
		req.Header.Set("X-Nonce", nonce)
		req.Header.Set("X-Signature", sig)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		// 6. 重放攻击测试 (相同 Nonce 第二次请求被拒绝)
		wReplay := httptest.NewRecorder()
		reqReplay := httptest.NewRequest(http.MethodPost, "/api/v1/cabinet/test", bytes.NewBuffer(body))
		reqReplay.Header.Set("X-Cabinet-ID", "CAB001")
		reqReplay.Header.Set("X-Timestamp", ts)
		reqReplay.Header.Set("X-Nonce", nonce)
		reqReplay.Header.Set("X-Signature", sig)
		r.ServeHTTP(wReplay, reqReplay)
		require.Equal(t, http.StatusUnauthorized, wReplay.Code)

		// 7. 数据库未配置密钥时直接拒绝 (杜绝默认 fallback)
		wNoSec := httptest.NewRecorder()
		reqNoSec := httptest.NewRequest(http.MethodPost, "/api/v1/cabinet/test", bytes.NewBuffer(body))
		reqNoSec.Header.Set("X-Cabinet-ID", "CAB_NO_SECRET")
		reqNoSec.Header.Set("X-Timestamp", ts)
		reqNoSec.Header.Set("X-Nonce", "nonce_no_sec")
		reqNoSec.Header.Set("X-Signature", "some_sig")
		r.ServeHTTP(wNoSec, reqNoSec)
		require.Equal(t, http.StatusUnauthorized, wNoSec.Code)
		require.Contains(t, wNoSec.Body.String(), "DEVICE_SECRET_NOT_CONFIGURED")
	})
}

func TestFaceSessionMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenSvc := jwt.NewTokenService("test_secret_32_bytes_super_secure", 86400)

	r := gin.New()
	r.Use(FaceSessionMiddleware(tokenSvc))
	r.POST("/api/v1/cabinet/direct-dispense", func(c *gin.Context) {
		uid, _ := c.Get("user_id")
		devID, _ := c.Get("cabinet_device_id")
		c.JSON(http.StatusOK, gin.H{"userId": uid, "deviceId": devID})
	})

	// 1. 普通 USER Token 无法通过 FaceSessionMiddleware
	userToken, _ := tokenSvc.Generate("u1", "USER")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cabinet/direct-dispense", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)

	// 2. FACE_SESSION Token 成功通过
	faceToken, _ := tokenSvc.GenerateFaceSession("u1", "USER", "CAB001", 5*time.Minute)
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/cabinet/direct-dispense", nil)
	req2.Header.Set("Authorization", "Bearer "+faceToken)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)

	// 3. 跨机柜盗用校验：上下文认证设备为 CAB002，但 Token 为 CAB001 时拒绝通过
	rMismatched := gin.New()
	rMismatched.Use(func(c *gin.Context) {
		c.Set("cabinet_device_id", "CAB002") // 模拟由 HMAC 认证出的现场设备为 CAB002
		c.Next()
	})
	rMismatched.Use(FaceSessionMiddleware(tokenSvc))
	rMismatched.POST("/api/v1/cabinet/direct-dispense", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/cabinet/direct-dispense", nil)
	req3.Header.Set("Authorization", "Bearer "+faceToken)
	w3 := httptest.NewRecorder()
	rMismatched.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusForbidden, w3.Code)
	require.Contains(t, w3.Body.String(), "DEVICE_MISMATCH")
}
