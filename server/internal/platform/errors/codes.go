package errors

type Code string

const (
	// Client errors (4xx)
	CodeInvalidInput  Code = "INVALID_INPUT"
	CodeUnauthorized  Code = "UNAUTHORIZED"
	CodeForbidden     Code = "FORBIDDEN"
	CodeNotFound      Code = "RESOURCE_NOT_FOUND"
	CodeConflict      Code = "CONFLICT"
	CodeInvalidState  Code = "INVALID_STATE"
	CodeTooEarly      Code = "TOO_EARLY"
	CodeExpired       Code = "EXPIRED"
	CodeDuplicate     Code = "DUPLICATE"

	// Server errors (5xx)
	CodeInternalError      Code = "INTERNAL_ERROR"
	CodeServiceUnavailable Code = "SERVICE_UNAVAILABLE"
)
