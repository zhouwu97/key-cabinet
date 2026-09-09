package jwt

import "github.com/golang-jwt/jwt/v5"

type Claims struct {
	UserID    string `json:"user_id"`
	Role      string `json:"role"`
	DeviceID  string `json:"device_id,omitempty"`
	TokenType string `json:"token_type,omitempty"` // "USER" or "FACE_SESSION"
	jwt.RegisteredClaims
}
