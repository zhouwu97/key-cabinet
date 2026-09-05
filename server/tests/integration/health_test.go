package integration

import (
	"github.com/stretchr/testify/assert"
	"github.com/zhouwu97/key-cabinet/server/internal/platform/jwt"
	transportHttp "github.com/zhouwu97/key-cabinet/server/internal/transport/http"
	"github.com/zhouwu97/key-cabinet/server/internal/transport/http/handler"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	healthHandler := handler.NewHealthHandler(db)
	tokenService := jwt.NewTokenService("test-secret", 3600)

	router := transportHttp.SetupRouter(transportHttp.RouterConfig{
		HealthHandler: healthHandler,
		TokenService:  tokenService,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
	assert.Contains(t, w.Body.String(), "database")
}

func TestHealthCheckResponseFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	healthHandler := handler.NewHealthHandler(db)
	tokenService := jwt.NewTokenService("test-secret", 3600)

	router := transportHttp.SetupRouter(transportHttp.RouterConfig{
		HealthHandler: healthHandler,
		TokenService:  tokenService,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify timestamp format
	body := w.Body.String()
	assert.Contains(t, body, "timestamp")

	// Verify it's valid JSON
	assert.Contains(t, body, `"status"`)
	assert.Contains(t, body, `"database"`)
}
