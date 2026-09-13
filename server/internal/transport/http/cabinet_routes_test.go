package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	stdhttp "net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/zhouwu97/key-cabinet/server/internal/platform/jwt"
	"github.com/zhouwu97/key-cabinet/server/internal/repository"
	"github.com/zhouwu97/key-cabinet/server/internal/service"
	"github.com/zhouwu97/key-cabinet/server/internal/transport/http/handler"
	"github.com/zhouwu97/key-cabinet/server/internal/transport/http/middleware"
)

type cabinetRouteDevices struct{ repository.DeviceRepository }

func (cabinetRouteDevices) FindByID(_ context.Context, id string) (*repository.Device, error) {
	return &repository.Device{ID: id, Status: "ONLINE", DeviceSecret: "local-route-test-secret"}, nil
}

type cabinetRouteAuth struct {
	service.CabinetService
	tokens *jwt.TokenService
}

func (s cabinetRouteAuth) FaceAuth(_ context.Context, params service.FaceAuthParams) (*service.FaceAuthResult, error) {
	token, err := s.tokens.GenerateFaceSession("u1", "USER", params.DeviceID, time.Minute)
	return &service.FaceAuthResult{FaceSessionToken: token, ExpiresIn: 60}, err
}

type cabinetRouteOperations struct {
	service.OperationService
	device, called string
}

func (s *cabinetRouteOperations) StartCabinetPickup(_ context.Context, user, reservation, cabinet, request string) (*repository.DeviceOperation, error) {
	s.called = fmt.Sprintf("pickup:%s:%s:%s:%s", user, reservation, cabinet, request)
	return &repository.DeviceOperation{ID: "op1", UserID: user, DeviceID: cabinet, Status: "EXECUTING"}, nil
}
func (s *cabinetRouteOperations) StartReturn(_ context.Context, user, borrow, cabinet, request string) (*repository.DeviceOperation, error) {
	s.called = fmt.Sprintf("return:%s:%s:%s:%s", user, borrow, cabinet, request)
	return &repository.DeviceOperation{ID: "op2", UserID: user, DeviceID: cabinet, Status: "EXECUTING"}, nil
}
func (s *cabinetRouteOperations) GetOperation(_ context.Context, user, id string) (*repository.DeviceOperation, error) {
	return &repository.DeviceOperation{ID: id, UserID: user, DeviceID: s.device, Status: "EXECUTING"}, nil
}
func (s *cabinetRouteOperations) GetActiveOperation(ctx context.Context, user string) (*repository.DeviceOperation, error) {
	return s.GetOperation(ctx, user, "op1")
}
func (s *cabinetRouteOperations) CancelOperation(_ context.Context, user, id string) error {
	s.called = "cancel:" + user + ":" + id
	return nil
}

func TestCabinetFaceRoutesRequireBothCredentialsAndDeviceScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokens := jwt.NewTokenService("local-face-route-secret", 300)
	ops := &cabinetRouteOperations{device: "CAB001"}
	router := SetupRouter(RouterConfig{HealthHandler: &handler.HealthHandler{}, TokenService: tokens,
		DeviceRepo: cabinetRouteDevices{}, CabinetHandler: handler.NewCabinetHandler(cabinetRouteAuth{tokens: tokens}),
		OperationHandler: handler.NewOperationHandler(ops)})
	sequence := 0
	request := func(method, path, body, token, cabinet string, signed bool) *httptest.ResponseRecorder {
		sequence++
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if signed {
			timestamp := strconv.FormatInt(time.Now().Unix(), 10)
			nonce := fmt.Sprintf("%s-%d-%d", t.Name(), time.Now().UnixNano(), sequence)
			req.Header.Set("X-Cabinet-ID", cabinet)
			req.Header.Set("X-Timestamp", timestamp)
			req.Header.Set("X-Nonce", nonce)
			req.Header.Set("X-Signature", middleware.GenerateCabinetSignature("local-route-test-secret", method, path, timestamp, nonce, []byte(body)))
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		return response
	}
	response := request("POST", "/api/v1/cabinet/auth/face", `{"studentNo":"20230001","confidence":0.95,"livenessPassed":true}`, "", "CAB001", true)
	require.Equal(t, stdhttp.StatusOK, response.Code)
	var auth struct {
		Data struct {
			FaceSessionToken string `json:"faceSessionToken"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &auth))
	faceToken := auth.Data.FaceSessionToken
	require.NotEmpty(t, faceToken)
	userToken, err := tokens.Generate("u1", "USER")
	require.NoError(t, err)
	path := "/api/v1/cabinet/device-operations/pickup"
	body := `{"reservationId":"r1","clientRequestId":"req1"}`
	require.Equal(t, 401, request("POST", path, body, faceToken, "CAB001", false).Code)
	require.Equal(t, 401, request("POST", path, body, "", "CAB001", true).Code)
	require.Equal(t, 403, request("POST", path, body, userToken, "CAB001", true).Code)
	require.Equal(t, 403, request("POST", path, body, faceToken, "CAB002", true).Code)
	expiredToken, err := tokens.GenerateFaceSession("u1", "USER", "CAB001", time.Nanosecond)
	require.NoError(t, err)
	require.Equal(t, 401, request("POST", path, body, expiredToken, "CAB001", true).Code)
	require.Empty(t, ops.called)
	require.Equal(t, 202, request("POST", path, body, faceToken, "CAB001", true).Code)
	require.Equal(t, "pickup:u1:r1:CAB001:req1", ops.called)
	path = "/api/v1/cabinet/device-operations/return"
	require.Equal(t, 403, request("POST", path, `{"borrowRecordId":"b1","clientRequestId":"req2","deviceId":"CAB002"}`, faceToken, "CAB001", true).Code)
	require.Equal(t, 202, request("POST", path, `{"borrowRecordId":"b1","clientRequestId":"req2"}`, faceToken, "CAB001", true).Code)
	require.Equal(t, "return:u1:b1:CAB001:req2", ops.called)
	path = "/api/v1/cabinet/device-operations/op1"
	require.Equal(t, 200, request("GET", path, "", faceToken, "CAB001", true).Code)
	ops.device = "CAB002"
	require.Equal(t, 403, request("GET", path, "", faceToken, "CAB001", true).Code)
	ops.called = ""
	require.Equal(t, 403, request("POST", path+"/cancel", "", faceToken, "CAB001", true).Code)
	require.Empty(t, ops.called)
	require.Contains(t, request("GET", "/api/v1/cabinet/device-operations/active", "", faceToken, "CAB001", true).Body.String(), `"data":null`)
	ops.device = "CAB001"
	require.Equal(t, 200, request("POST", path+"/cancel", "", faceToken, "CAB001", true).Code)
	require.Equal(t, "cancel:u1:op1", ops.called)
}

func TestCabinetRoutesAreNotRegisteredWithoutDeviceAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := SetupRouter(RouterConfig{HealthHandler: &handler.HealthHandler{},
		CabinetHandler: handler.NewCabinetHandler(cabinetRouteAuth{}), TokenService: jwt.NewTokenService("local-test", 300)})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("POST", "/api/v1/cabinet/auth/face", nil))
	require.Equal(t, 404, response.Code)
}
