package wechat

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type SessionResult struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

type Client interface {
	Code2Session(ctx context.Context, jsCode string) (*SessionResult, error)
}

type WechatClient struct {
	appID      string
	appSecret  string
	httpClient *http.Client
}

func NewClient(appID, appSecret string) Client {
	return &WechatClient{
		appID:     appID,
		appSecret: appSecret,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *WechatClient) Code2Session(ctx context.Context, jsCode string) (*SessionResult, error) {
	// Dev/Mock fallback:
	// If AppID/Secret is not configured, or if the code starts with mock_ or dev_
	isDevMode := c.appID == "" || c.appSecret == "" ||
		c.appSecret == "your-wechat-app-secret" ||
		strings.HasPrefix(jsCode, "mock_") ||
		strings.HasPrefix(jsCode, "dev_")

	if isDevMode {
		h := md5.Sum([]byte(jsCode))
		mockOpenID := fmt.Sprintf("wx_mock_%s", hex.EncodeToString(h[:8]))
		return &SessionResult{
			OpenID:     mockOpenID,
			SessionKey: "mock_session_key",
			UnionID:    "",
			ErrCode:    0,
			ErrMsg:     "ok",
		}, nil
	}

	url := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		c.appID, c.appSecret, jsCode,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("wechat api request failed: %w", err)
	}
	defer resp.Body.Close()

	var result SessionResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode wechat response: %w", err)
	}

	if result.ErrCode != 0 {
		return nil, fmt.Errorf("wechat error %d: %s", result.ErrCode, result.ErrMsg)
	}

	return &result, nil
}
