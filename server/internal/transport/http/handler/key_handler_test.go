package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/zhouwu97/key-cabinet/server/internal/platform/jwt"
	"github.com/zhouwu97/key-cabinet/server/internal/repository"
	"github.com/zhouwu97/key-cabinet/server/internal/service"
	transporthttp "github.com/zhouwu97/key-cabinet/server/internal/transport/http"
	"github.com/zhouwu97/key-cabinet/server/internal/transport/http/handler"
)

type keyServiceStub struct {
	keys  []*repository.Key
	last  service.KeyListFilter
	slots []*repository.Slot
}

func (s *keyServiceStub) ListKeys(_ context.Context, filter service.KeyListFilter) ([]*repository.Key, error) {
	s.last = filter
	return s.keys, nil
}

func (s *keyServiceStub) GetKey(_ context.Context, _ string) (*repository.Key, error) {
	return s.keys[0], nil
}

func (s *keyServiceStub) GetKeySlot(_ context.Context, _ string) (*repository.Slot, error) {
	return s.slots[0], nil
}

func newTestRouter(keyService service.KeyService) *gin.Engine {
	tokenService := jwt.NewTokenService("test-secret", 3600)
	return transporthttp.SetupRouter(transporthttp.RouterConfig{
		HealthHandler: handler.NewHealthHandler(nil),
		KeyHandler:    handler.NewKeyHandler(keyService),
		TokenService:  tokenService,
	})
}

func TestKeyHandlerListUsesContractEnvelopeAndFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &keyServiceStub{keys: []*repository.Key{{ID: "KEY-1", Name: "101室钥匙"}}}
	router := newTestRouter(stub)
	token, err := jwt.NewTokenService("test-secret", 3600).Generate("USER-1", "USER")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/keys?keyword=101&deviceId=CAB-1&status=available&enabled=true", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	require.Equal(t, http.StatusOK, res.Code)
	require.Contains(t, res.Body.String(), `"code":0`)
	require.Contains(t, res.Body.String(), `"id":"KEY-1"`)
	require.Equal(t, "101", stub.last.Keyword)
	require.Equal(t, "CAB-1", stub.last.DeviceID)
	require.Equal(t, "AVAILABLE", stub.last.Status)
	require.NotNil(t, stub.last.Enabled)
	require.True(t, *stub.last.Enabled)
}

func TestKeyHandlerRejectsInvalidEnabledQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := newTestRouter(&keyServiceStub{})
	token, err := jwt.NewTokenService("test-secret", 3600).Generate("USER-1", "USER")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/keys?enabled=maybe", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	require.Equal(t, http.StatusBadRequest, res.Code)
	require.Contains(t, res.Body.String(), "enabled must be true or false")
}
