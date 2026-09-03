package jwt

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestGenerateAndValidate(t *testing.T) {
	service := NewTokenService("test-secret", 3600)

	token, err := service.Generate("U001", "STUDENT")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := service.Validate(token)
	assert.NoError(t, err)
	assert.Equal(t, "U001", claims.UserID)
	assert.Equal(t, "STUDENT", claims.Role)
}

func TestValidateInvalidToken(t *testing.T) {
	service := NewTokenService("test-secret", 3600)

	_, err := service.Validate("invalid.token.here")
	assert.Error(t, err)
}

func TestValidateWrongSecret(t *testing.T) {
	service1 := NewTokenService("secret1", 3600)
	service2 := NewTokenService("secret2", 3600)

	token, err := service1.Generate("U001", "STUDENT")
	assert.NoError(t, err)

	_, err = service2.Validate(token)
	assert.Error(t, err)
}
