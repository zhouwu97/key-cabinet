package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/zhouwu97/key-cabinet/server/internal/repository"
	"github.com/zhouwu97/key-cabinet/server/internal/service"
	"github.com/zhouwu97/key-cabinet/server/internal/transport/http/dto"
	"github.com/zhouwu97/key-cabinet/server/internal/transport/http/middleware"
)

type fakeCabinetService struct {
	matchRoomFn      func(ctx context.Context, deviceID, roomNo string) ([]*service.MatchedKeyInfo, error)
	directDispenseFn func(ctx context.Context, params service.CabinetDirectDispenseParams) (*repository.DeviceOperation, *repository.BorrowRecord, *repository.Slot, *repository.Key, error)
	faceAuthFn       func(ctx context.Context, params service.FaceAuthParams) (*service.FaceAuthResult, error)
}

func (s *fakeCabinetService) MatchRoom(ctx context.Context, deviceID, roomNo string) ([]*service.MatchedKeyInfo, error) {
	if s.matchRoomFn != nil {
		return s.matchRoomFn(ctx, deviceID, roomNo)
	}
	return nil, nil
}

func (s *fakeCabinetService) DirectDispense(ctx context.Context, params service.CabinetDirectDispenseParams) (*repository.DeviceOperation, *repository.BorrowRecord, *repository.Slot, *repository.Key, error) {
	if s.directDispenseFn != nil {
		return s.directDispenseFn(ctx, params)
	}
	return nil, nil, nil, nil, nil
}

func (s *fakeCabinetService) FaceAuth(ctx context.Context, params service.FaceAuthParams) (*service.FaceAuthResult, error) {
	if s.faceAuthFn != nil {
		return s.faceAuthFn(ctx, params)
	}
	return nil, nil
}

func TestCabinetHandler_MatchRoom(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fakeSvc := &fakeCabinetService{
		matchRoomFn: func(ctx context.Context, deviceID, roomNo string) ([]*service.MatchedKeyInfo, error) {
			return []*service.MatchedKeyInfo{
				{
					Key:          &repository.Key{ID: "k1", Name: "101室钥匙", RoomNo: "101", DeviceID: "CAB001", Status: "AVAILABLE"},
					Slot:         &repository.Slot{ID: "s1", SlotNo: 1, Presence: "PRESENT"},
					IsBorrowable: true,
				},
			}, nil
		},
	}

	h := NewCabinetHandler(fakeSvc)
	r := gin.New()
	r.Use(middleware.ErrorMiddleware())
	r.GET("/api/v1/cabinet/keys/match-room", h.MatchRoom)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cabinet/keys/match-room?deviceId=CAB001&roomNo=101", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp dto.SuccessResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
}

func TestCabinetHandler_DirectDispense(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fakeSvc := &fakeCabinetService{
		directDispenseFn: func(ctx context.Context, params service.CabinetDirectDispenseParams) (*repository.DeviceOperation, *repository.BorrowRecord, *repository.Slot, *repository.Key, error) {
			return &repository.DeviceOperation{
					ID:       "op_123",
					DeviceID: params.DeviceID,
					SlotID:   "s1",
					Status:   "EXECUTING",
				},
				&repository.BorrowRecord{ID: "bor_123"},
				&repository.Slot{ID: "s1", SlotNo: 2},
				&repository.Key{ID: "k1", Name: "101室主钥匙", RoomNo: "101"},
				nil
		},
	}

	h := NewCabinetHandler(fakeSvc)
	r := gin.New()
	r.Use(middleware.ErrorMiddleware())
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "u1")
		c.Set("cabinet_device_id", "CAB001")
		c.Next()
	})
	r.POST("/api/v1/cabinet/direct-dispense", h.DirectDispense)

	body, _ := json.Marshal(dto.CabinetDirectDispenseRequest{
		RequestID: "req_test",
		DeviceID:  "CAB001",
		RoomNo:    "101",
		StudentNo: "20230001",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cabinet/direct-dispense", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusAccepted, w.Code)
	var resp dto.SuccessResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
}

func TestCabinetHandler_FaceAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fakeSvc := &fakeCabinetService{
		faceAuthFn: func(ctx context.Context, params service.FaceAuthParams) (*service.FaceAuthResult, error) {
			return &service.FaceAuthResult{
				User:             &repository.User{ID: "u1", Name: "张三", StudentNo: "20230001"},
				FaceSessionToken: "mock_jwt_token",
				CabinetToken:     "mock_jwt_token",
			}, nil
		},
	}

	h := NewCabinetHandler(fakeSvc)
	r := gin.New()
	r.Use(middleware.ErrorMiddleware())
	r.Use(func(c *gin.Context) {
		c.Set("cabinet_device_id", "CAB001")
		c.Next()
	})
	r.POST("/api/v1/cabinet/auth/face", h.FaceAuth)

	body, _ := json.Marshal(dto.FaceAuthRequest{
		DeviceID:       "CAB001",
		StudentNo:      "20230001",
		Confidence:     0.96,
		LivenessPassed: true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cabinet/auth/face", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp dto.SuccessResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
}
