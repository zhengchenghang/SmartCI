package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"lite-cicd/config"
	"lite-cicd/oauth"
)

// Handler webhook处理器
type Handler struct {
	config    config.WebhookConfig
	provider  oauth.Provider
	executor  func(ctx context.Context, action config.WebhookAction, payload interface{}) error
}

// NewHandler 创建webhook处理器
func NewHandler(cfg config.WebhookConfig, provider oauth.Provider, executor func(ctx context.Context, action config.WebhookAction, payload interface{}) error) *Handler {
	return &Handler{
		config:   cfg,
		provider: provider,
		executor: executor,
	}
}

// ServeHTTP 处理webhook请求
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 验证签名
	if h.config.Secret != "" && h.provider != nil {
		if err := h.provider.ValidateWebhook(r, h.config.Secret); err != nil {
			log.Printf("❌ Webhook签名验证失败: %v", err)
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ 读取webhook请求体失败: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// 解析payload
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("❌ 解析webhook payload失败: %v", err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// 获取事件类型
	event := r.Header.Get("X-GitHub-Event")
	if event == "" {
		event = r.Header.Get("X-Gitlab-Event")
	}
	if event == "" {
		event = r.Header.Get("X-Gitea-Event")
	}

	log.Printf("📥 收到webhook: %s, 事件: %s", h.config.Name, event)

	// 检查事件过滤
	if !h.shouldProcess(event, payload) {
		log.Printf("⏭️ Webhook事件被过滤: %s", event)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Event filtered"))
		return
	}

	// 执行动作
	go func() {
		ctx := context.Background()
		for _, action := range h.config.Actions {
			if err := h.executor(ctx, action, payload); err != nil {
				log.Printf("❌ 执行webhook动作失败: %v", err)
			}
		}
	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Webhook processed"))
}

// shouldProcess 判断是否应该处理该webhook
func (h *Handler) shouldProcess(event string, payload map[string]interface{}) bool {
	// 检查事件类型
	if len(h.config.Events) > 0 {
		found := false
		for _, e := range h.config.Events {
			if strings.EqualFold(e, event) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// 检查分支过滤
	if len(h.config.Filters.Branches) > 0 {
		branch := extractBranch(payload)
		if branch != "" {
			found := false
			for _, b := range h.config.Filters.Branches {
				if b == branch {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	// 检查仓库过滤
	if len(h.config.Filters.Repos) > 0 {
		repo := extractRepo(payload)
		if repo != "" {
			found := false
			for _, r := range h.config.Filters.Repos {
				if r == repo {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	// 检查动作过滤
	if len(h.config.Filters.Actions) > 0 {
		action := extractAction(payload)
		if action != "" {
			found := false
			for _, a := range h.config.Filters.Actions {
				if a == action {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	return true
}

// extractBranch 从payload中提取分支名
func extractBranch(payload map[string]interface{}) string {
	// GitHub: ref字段
	if ref, ok := payload["ref"].(string); ok {
		if strings.HasPrefix(ref, "refs/heads/") {
			return strings.TrimPrefix(ref, "refs/heads/")
		}
	}

	// Pull Request: base.ref
	if pr, ok := payload["pull_request"].(map[string]interface{}); ok {
		if base, ok := pr["base"].(map[string]interface{}); ok {
			if ref, ok := base["ref"].(string); ok {
				return ref
			}
		}
	}

	return ""
}

// extractRepo 从payload中提取仓库名
func extractRepo(payload map[string]interface{}) string {
	if repository, ok := payload["repository"].(map[string]interface{}); ok {
		if name, ok := repository["name"].(string); ok {
			return name
		}
		if fullName, ok := repository["full_name"].(string); ok {
			return fullName
		}
	}
	return ""
}

// extractAction 从payload中提取动作类型
func extractAction(payload map[string]interface{}) string {
	if action, ok := payload["action"].(string); ok {
		return action
	}
	return ""
}

// GitHubPushPayload GitHub push事件payload
type GitHubPushPayload struct {
	Ref        string `json:"ref"`
	Before     string `json:"before"`
	After      string `json:"after"`
	Repository struct {
		Name     string `json:"name"`
		FullName string `json:"full_name"`
	} `json:"repository"`
	Pusher struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"pusher"`
	Commits []struct {
		ID      string `json:"id"`
		Message string `json:"message"`
		Author  struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"author"`
	} `json:"commits"`
}

// ParseGitHubPush 解析GitHub push事件
func ParseGitHubPush(body []byte) (*GitHubPushPayload, error) {
	var payload GitHubPushPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("解析GitHub push payload失败: %w", err)
	}
	return &payload, nil
}

