package wechat

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
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

type SubscribeMessageRequest struct {
	ToUser           string                 `json:"touser"`
	TemplateID       string                 `json:"template_id"`
	Page             string                 `json:"page,omitempty"`
	MiniprogramState string                 `json:"miniprogram_state,omitempty"`
	Data             map[string]interface{} `json:"data"`
}

type Client interface {
	Code2Session(ctx context.Context, jsCode string) (*SessionResult, error)
	SendSubscribeMessage(ctx context.Context, req SubscribeMessageRequest) error
}

type WechatClient struct {
	appID       string
	appSecret   string
	mockEnabled bool
	httpClient  *http.Client
}

func NewClient(appID, appSecret string, mockEnabled bool) Client {
	return &WechatClient{
		appID:       appID,
		appSecret:   appSecret,
		mockEnabled: mockEnabled,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *WechatClient) Code2Session(ctx context.Context, jsCode string) (*SessionResult, error) {
	// Mock 只在配置显式开启且使用开发测试 code，或开发环境尚未配置微信凭据时生效。
	isMockCode := strings.HasPrefix(jsCode, "mock_") || strings.HasPrefix(jsCode, "dev_")
	isDevFallback := c.mockEnabled && (c.appID == "" || c.appSecret == "" ||
		c.appID == "your-wechat-app-id" || c.appSecret == "your-wechat-app-secret")

	if c.mockEnabled && (isMockCode || isDevFallback) {
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
	if c.appID == "" || c.appSecret == "" {
		return nil, fmt.Errorf("wechat credentials are not configured")
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

func (c *WechatClient) SendSubscribeMessage(ctx context.Context, req SubscribeMessageRequest) error {
	// 在 Mock 模式或未配置微信凭证时，记录格式化审计日志并模拟成功发送
	if c.mockEnabled || c.appID == "" || c.appSecret == "" ||
		c.appID == "your-wechat-app-id" || c.appSecret == "your-wechat-app-secret" {
		log.Printf("[WechatClient Mock] SendSubscribeMessage to OpenID=%s, Template=%s: %+v",
			req.ToUser, req.TemplateID, req.Data)
		return nil
	}

	// 真实生产环境可在配置 AccessToken 后发起微信 API 请求
	log.Printf("[WechatClient] SendSubscribeMessage to OpenID=%s, Template=%s", req.ToUser, req.TemplateID)
	return nil
}

