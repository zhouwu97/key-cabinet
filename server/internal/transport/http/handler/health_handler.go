package handler

import (
	"net/http"
	"time"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"github.com/zhouwu97/key-cabinet/server/internal/infrastructure/postgres"
)

type HealthHandler struct {
	db *gorm.DB
}

func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

type HealthResponse struct {
	Status    string `json:"status"`
	Database  string `json:"database"`
	Timestamp string `json:"timestamp"`
}

func (h *HealthHandler) Check(c *gin.Context) {
	dbStatus := "ok"
	if err := postgres.HealthCheck(h.db); err != nil {
		dbStatus = "error"
	}

	status := "ok"
	if dbStatus != "ok" {
		status = "degraded"
	}

	c.JSON(http.StatusOK, HealthResponse{
		Status:    status,
		Database:  dbStatus,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}
