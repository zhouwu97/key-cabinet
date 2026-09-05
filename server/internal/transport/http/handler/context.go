package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/zhouwu97/key-cabinet/server/internal/platform/errors"
)

func currentUserID(c *gin.Context) (string, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		c.Error(errors.New(errors.CodeUnauthorized, "missing user identity in context"))
		return "", false
	}
	userID, ok := value.(string)
	if !ok || userID == "" {
		c.Error(errors.New(errors.CodeUnauthorized, "invalid user identity in context"))
		return "", false
	}
	return userID, true
}
