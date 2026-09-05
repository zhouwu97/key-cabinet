package wechat

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientRequiresExplicitMockSwitch(t *testing.T) {
	client := NewClient("", "", false)

	_, err := client.Code2Session(context.Background(), "mock_code")
	assert.ErrorContains(t, err, "wechat credentials are not configured")
}

func TestClientUsesMockOnlyWhenEnabled(t *testing.T) {
	client := NewClient("", "", true)

	session, err := client.Code2Session(context.Background(), "mock_code")
	require.NoError(t, err)
	assert.NotEmpty(t, session.OpenID)
	assert.Equal(t, "mock_session_key", session.SessionKey)
}
