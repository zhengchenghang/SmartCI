package oauth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	githubAuthURL     = "https://github.com/login/oauth/authorize"
	githubTokenURL    = "https://github.com/login/oauth/access_token"
	githubUserInfoURL = "https://api.github.com/user"
)

// GitHubProvider GitHub OAuth提供商
type GitHubProvider struct {
	BaseProvider
}

// GitHubUser GitHub用户信息
type GitHubUser struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// NewGitHubProvider 创建GitHub OAuth提供商
func NewGitHubProvider(clientID, clientSecret, redirectURL string, scopes []string) *GitHubProvider {
	return &GitHubProvider{
		BaseProvider: BaseProvider{
			Config: Config{
				ClientID:     clientID,
				ClientSecret: clientSecret,
				RedirectURL:  redirectURL,
				Scopes:       scopes,
			},
			AuthURL:     githubAuthURL,
			TokenURL:    githubTokenURL,
			UserInfoURL: githubUserInfoURL,
		},
	}
}

// GetAuthURL 获取GitHub授权URL
func (g *GitHubProvider) GetAuthURL(state string) string {
	params := url.Values{}
	params.Set("client_id", g.Config.ClientID)
	params.Set("redirect_uri", g.Config.RedirectURL)
	params.Set("scope", strings.Join(g.Config.Scopes, " "))
	params.Set("state", state)
	
	return fmt.Sprintf("%s?%s", g.AuthURL, params.Encode())
}

// ExchangeToken 用授权码换取访问令牌
func (g *GitHubProvider) ExchangeToken(ctx context.Context, code string) (*Token, error) {
	params := url.Values{}
	params.Set("client_id", g.Config.ClientID)
	params.Set("client_secret", g.Config.ClientSecret)
	params.Set("code", code)
	params.Set("redirect_uri", g.Config.RedirectURL)

	req, err := http.NewRequestWithContext(ctx, "POST", g.TokenURL, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

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
		return nil, fmt.Errorf("获取令牌失败: HTTP %d: %s", resp.StatusCode, string(data))
	}

	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("解析令牌失败: %w", err)
	}

	return &token, nil
}

// RefreshToken GitHub不支持刷新令牌
func (g *GitHubProvider) RefreshToken(ctx context.Context, refreshToken string) (*Token, error) {
	return nil, fmt.Errorf("GitHub OAuth不支持刷新令牌")
}

// GetUserInfo 获取GitHub用户信息
func (g *GitHubProvider) GetUserInfo(ctx context.Context, accessToken string) (interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", g.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

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
		return nil, fmt.Errorf("获取用户信息失败: HTTP %d: %s", resp.StatusCode, string(data))
	}

	var user GitHubUser
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("解析用户信息失败: %w", err)
	}

	return &user, nil
}

// ValidateWebhook 验证GitHub webhook签名
func (g *GitHubProvider) ValidateWebhook(r *http.Request, secret string) error {
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		return fmt.Errorf("缺少webhook签名")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("读取请求体失败: %w", err)
	}

	// 计算HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedMAC := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedMAC)) {
		return fmt.Errorf("webhook签名验证失败")
	}

	return nil
}

