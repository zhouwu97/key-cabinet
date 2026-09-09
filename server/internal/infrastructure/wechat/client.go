package wechat

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
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
	appID          string
	appSecret      string
	mockEnabled    bool
	httpClient     *http.Client
	tokenMu        sync.RWMutex
	accessToken    string
	tokenExpiresAt time.Time
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

func (c *WechatClient) isMock() bool {
	return c.mockEnabled
}

func (c *WechatClient) Code2Session(ctx context.Context, jsCode string) (*SessionResult, error) {
	if !c.mockEnabled && (c.appID == "" || c.appSecret == "" || c.appID == "your-wechat-app-id" || c.appSecret == "your-wechat-app-secret") {
		return nil, fmt.Errorf("wechat credentials are not configured")
	}

	// 生产环境下 (mockEnabled = false)，严禁任何 mock_/dev_ 代码绕过真实微信服务器
	if c.mockEnabled {
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

func (c *WechatClient) getAccessToken(ctx context.Context) (string, error) {
	c.tokenMu.RLock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpiresAt) {
		token := c.accessToken
		c.tokenMu.RUnlock()
		return token, nil
	}
	c.tokenMu.RUnlock()

	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	if c.accessToken != "" && time.Now().Before(c.tokenExpiresAt) {
		return c.accessToken, nil
	}

	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		c.appID, c.appSecret)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to request access token: %w", err)
	}
	defer resp.Body.Close()

	var res struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("failed to decode access token response: %w", err)
	}
	if res.ErrCode != 0 {
		return "", fmt.Errorf("wechat token error %d: %s", res.ErrCode, res.ErrMsg)
	}

	c.accessToken = res.AccessToken
	refreshBuffer := 300 * time.Second
	c.tokenExpiresAt = time.Now().Add(time.Duration(res.ExpiresIn)*time.Second - refreshBuffer)

	return c.accessToken, nil
}

func (c *WechatClient) SendSubscribeMessage(ctx context.Context, req SubscribeMessageRequest) error {
	if c.isMock() {
		log.Printf("[WechatClient Mock] SendSubscribeMessage to OpenID=%s, Template=%s: %+v",
			req.ToUser, req.TemplateID, req.Data)
		return nil
	}

	token, err := c.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to obtain wechat access token: %w", err)
	}

	return c.doSendSubscribe(ctx, token, req, true)
}

func (c *WechatClient) doSendSubscribe(ctx context.Context, token string, req SubscribeMessageRequest, canRetry bool) error {
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/message/subscribe/send?access_token=%s", token)

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to serialize subscribe message: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("wechat subscribe send failed: %w", err)
	}
	defer resp.Body.Close()

	var res struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return fmt.Errorf("failed to decode subscribe response: %w", err)
	}

	if (res.ErrCode == 40001 || res.ErrCode == 42001) && canRetry {
		c.tokenMu.Lock()
		c.accessToken = ""
		c.tokenMu.Unlock()
		newToken, err := c.getAccessToken(ctx)
		if err != nil {
			return err
		}
		return c.doSendSubscribe(ctx, newToken, req, false)
	}

	if res.ErrCode != 0 {
		return fmt.Errorf("wechat subscribe error %d: %s", res.ErrCode, res.ErrMsg)
	}

	log.Printf("[WechatClient] Successfully sent subscribe message to OpenID=%s", req.ToUser)
	return nil
}
