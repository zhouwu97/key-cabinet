package dto

import "time"

type SuccessResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func NewSuccessResponse(data interface{}) SuccessResponse {
	return SuccessResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}
}

type ErrorResponse struct {
	Code      int         `json:"code"`
	ErrorCode string      `json:"errorCode"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	Timestamp string      `json:"timestamp"`
	Error     ErrorDetail `json:"error"` // legacy compatibility
}

type ErrorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

func NewErrorResponse(code, message string) ErrorResponse {
	now := time.Now().Format(time.RFC3339)
	return ErrorResponse{
		Code:      400,
		ErrorCode: code,
		Message:   message,
		Data:      nil,
		Timestamp: now,
		Error: ErrorDetail{
			Code:      code,
			Message:   message,
			Timestamp: now,
		},
	}
}
