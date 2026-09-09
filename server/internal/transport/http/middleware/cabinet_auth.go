package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zhouwu97/key-cabinet/server/internal/platform/jwt"
	"github.com/zhouwu97/key-cabinet/server/internal/repository"
	"github.com/zhouwu97/key-cabinet/server/internal/transport/http/dto"
)

var (
	nonceMu    sync.Mutex
	nonceStore = make(map[string]time.Time)
)

// GenerateCabinetSignature calculates canonical HMAC-SHA256 signature for cabinet edge devices.
// canonical format: method + "\n" + path + "\n" + timestamp + "\n" + nonce + "\n" + sha256(body)
func GenerateCabinetSignature(secret, method, path, timestamp, nonce string, body []byte) string {
	sha := sha256.Sum256(body)
	bodyHash := hex.EncodeToString(sha[:])
	canonical := fmt.Sprintf("%s\n%s\n%s\n%s\n%s", strings.ToUpper(method), path, timestamp, nonce, bodyHash)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(canonical))
	return hex.EncodeToString(mac.Sum(nil))
}

// CabinetAuthMiddleware authenticates cabinet edge hardware via HMAC-SHA256 signature,
// timestamp window check, and nonce replay prevention.
func CabinetAuthMiddleware(deviceRepo repository.DeviceRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := strings.TrimSpace(c.GetHeader("X-Cabinet-ID"))
		if deviceID == "" {
			deviceID = strings.TrimSpace(c.GetHeader("X-Cabinet-Device-ID"))
		}
		timestampStr := strings.TrimSpace(c.GetHeader("X-Timestamp"))
		if timestampStr == "" {
			timestampStr = strings.TrimSpace(c.GetHeader("X-Cabinet-Timestamp"))
		}
		nonceStr := strings.TrimSpace(c.GetHeader("X-Nonce"))
		if nonceStr == "" {
			nonceStr = strings.TrimSpace(c.GetHeader("X-Cabinet-Nonce"))
		}
		sigStr := strings.TrimSpace(c.GetHeader("X-Signature"))
		if sigStr == "" {
			sigStr = strings.TrimSpace(c.GetHeader("X-Cabinet-Signature"))
		}

		if deviceID == "" || timestampStr == "" || nonceStr == "" || sigStr == "" {
			c.JSON(http.StatusUnauthorized, dto.NewErrorResponse(
				"UNAUTHORIZED_CABINET",
				"Missing cabinet authentication headers (X-Cabinet-ID, X-Timestamp, X-Nonce, X-Signature required)",
			))
			c.Abort()
			return
		}

		// 1. 校验时间戳 (允许 ±300 秒 / 5 分钟的时间窗口)
		tsInt, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusUnauthorized, dto.NewErrorResponse(
				"INVALID_TIMESTAMP",
				"Invalid timestamp format",
			))
			c.Abort()
			return
		}
		// 支持毫秒或秒时间戳
		if tsInt > 1e12 {
			tsInt /= 1000
		}
		nowSec := time.Now().Unix()
		diffSec := nowSec - tsInt
		if diffSec < -300 || diffSec > 300 {
			c.JSON(http.StatusUnauthorized, dto.NewErrorResponse(
				"TIMESTAMP_EXPIRED",
				"Request timestamp is outside the allowed 5-minute window",
			))
			c.Abort()
			return
		}

		// 2. 校验 Nonce 防重放攻击
		nonceKey := deviceID + ":" + nonceStr
		now := time.Now()
		nonceMu.Lock()
		// 清理过期 Nonce
		for k, exp := range nonceStore {
			if now.After(exp) {
				delete(nonceStore, k)
			}
		}
		if _, exists := nonceStore[nonceKey]; exists {
			nonceMu.Unlock()
			c.JSON(http.StatusUnauthorized, dto.NewErrorResponse(
				"NONCE_REPLAYED",
				"Nonce has already been used within the window",
			))
			c.Abort()
			return
		}
		nonceStore[nonceKey] = now.Add(5 * time.Minute)
		nonceMu.Unlock()

		// 3. 查验机柜设备与密钥
		device, err := deviceRepo.FindByID(c.Request.Context(), deviceID)
		if err != nil || device == nil {
			c.JSON(http.StatusUnauthorized, dto.NewErrorResponse(
				"DEVICE_NOT_FOUND",
				"Cabinet hardware is not registered in the system",
			))
			c.Abort()
			return
		}

		if device.Status != "ONLINE" {
			c.JSON(http.StatusForbidden, dto.NewErrorResponse(
				"DEVICE_OFFLINE",
				"Cabinet device is offline or unavailable",
			))
			c.Abort()
			return
		}

		secret := strings.TrimSpace(device.DeviceSecret)
		if secret == "" {
			// 若数据库未配置特定密钥，使用系统默认规则
			secret = "cab_sec_" + device.ID
		}

		// 4. 读取 Body 并计算 HMAC 签名
		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, err = io.ReadAll(c.Request.Body)
			if err != nil {
				c.JSON(http.StatusBadRequest, dto.NewErrorResponse("BAD_REQUEST", "Failed to read request body"))
				c.Abort()
				return
			}
			// 还原 Body 供下游 Handler 读取
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		expectedSig := GenerateCabinetSignature(secret, c.Request.Method, c.Request.URL.Path, timestampStr, nonceStr, bodyBytes)
		if !hmac.Equal([]byte(strings.ToLower(sigStr)), []byte(strings.ToLower(expectedSig))) {
			c.JSON(http.StatusUnauthorized, dto.NewErrorResponse(
				"INVALID_SIGNATURE",
				"Cabinet device HMAC-SHA256 signature verification failed",
			))
			c.Abort()
			return
		}

		c.Set("cabinet_device_id", deviceID)
		c.Next()
	}
}

// FaceSessionMiddleware validates that the request carries a short-lived face session token
// generated after successful edge biometric verification on the specific cabinet.
func FaceSessionMiddleware(tokenService *jwt.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, dto.NewErrorResponse(
				"UNAUTHORIZED",
				"Missing face session token",
			))
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, dto.NewErrorResponse(
				"UNAUTHORIZED",
				"Invalid authorization header format",
			))
			c.Abort()
			return
		}

		claims, err := tokenService.Validate(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, dto.NewErrorResponse(
				"INVALID_TOKEN",
				"Face session has expired or is invalid: "+err.Error(),
			))
			c.Abort()
			return
		}

		if claims.TokenType != "FACE_SESSION" {
			c.JSON(http.StatusForbidden, dto.NewErrorResponse(
				"INVALID_TOKEN_TYPE",
				"Operation requires a valid FACE_SESSION token",
			))
			c.Abort()
			return
		}

		if claims.DeviceID == "" || claims.UserID == "" {
			c.JSON(http.StatusForbidden, dto.NewErrorResponse(
				"INVALID_TOKEN",
				"Face session token missing user or device binding",
			))
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Set("cabinet_device_id", claims.DeviceID)
		c.Next()
	}
}
