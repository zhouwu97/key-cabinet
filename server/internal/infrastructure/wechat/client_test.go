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

func TestClientRejectsMockCodeWhenMockDisabledWithCredentials(t *testing.T) {
	// 当 mockEnabled = false 且凭据已配置时，任何 mock_ 代码都必须进入真实请求而不能绕过
	client := NewClient("wx_real_appid_12345", "wx_real_secret_12345", false)

	session, err := client.Code2Session(context.Background(), "mock_attack_code")
	if err == nil {
		assert.NotEqual(t, "mock_session_key", session.SessionKey)
	} else {
		assert.NotContains(t, err.Error(), "wechat credentials are not configured")
	}
}
