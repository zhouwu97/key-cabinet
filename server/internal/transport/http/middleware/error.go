package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/zhouwu97/key-cabinet/server/internal/platform/errors"
	"github.com/zhouwu97/key-cabinet/server/internal/transport/http/dto"
	"net/http"
)

func ErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			appErr, ok := errors.IsAppError(err)
			if !ok {
				appErr = errors.New(errors.CodeInternalError, err.Error())
			}

			statusCode := mapErrorToHTTPStatus(appErr.Code)
			c.JSON(statusCode, dto.NewErrorResponse(string(appErr.Code), appErr.Message))
		}
	}
}

func mapErrorToHTTPStatus(code errors.Code) int {
	switch code {
	case errors.CodeInvalidInput:
		return http.StatusBadRequest
	case errors.CodeUnauthorized:
		return http.StatusUnauthorized
	case errors.CodeForbidden:
		return http.StatusForbidden
	case errors.CodeNotFound:
		return http.StatusNotFound
	case errors.CodeConflict, errors.CodeInvalidState, errors.CodeDuplicate:
		return http.StatusConflict
	case errors.CodeTooEarly:
		return http.StatusBadRequest
	case errors.CodeExpired:
		return http.StatusGone
	case errors.CodeServiceUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
