package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Provider OAuth提供商通用接口
type Provider interface {
	// GetAuthURL 获取OAuth授权URL
	GetAuthURL(state string) string
	
	// ExchangeToken 用授权码换取访问令牌
	ExchangeToken(ctx context.Context, code string) (*Token, error)
	
	// RefreshToken 刷新访问令牌
	RefreshToken(ctx context.Context, refreshToken string) (*Token, error)
	
	// GetUserInfo 获取用户信息
	GetUserInfo(ctx context.Context, accessToken string) (interface{}, error)
	
	// ValidateWebhook 验证webhook请求签名
	ValidateWebhook(r *http.Request, secret string) error
}

// Token OAuth令牌
type Token struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// Config OAuth配置
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// BaseProvider OAuth基础提供商实现
type BaseProvider struct {
	Config       Config
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
}

// doRequest 执行HTTP请求
func doRequest(ctx context.Context, method, url string, body io.Reader, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))
	}

	return data, nil
}

// exchangeToken 通用令牌交换实现
func (bp *BaseProvider) exchangeToken(ctx context.Context, tokenURL string, params url.Values) (*Token, error) {
	data, err := doRequest(ctx, "POST", tokenURL, 
		nil, // POST form使用params
		map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		})
	if err != nil {
		return nil, err
	}

	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("解析令牌失败: %w", err)
	}

	return &token, nil
}

