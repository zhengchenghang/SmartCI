package ai

import (
	"context"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

type AIAgent struct {
	client *openai.Client
}

func NewAIAgent(apiKey, baseURL string) *AIAgent {
	config := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		config.BaseURL = baseURL
	}
	return &AIAgent{client: openai.NewClientWithConfig(config)}
}

func (a *AIAgent) AnalyzeLog(logPath string) (string, error) {
	content, err := os.ReadFile(logPath)
	if err != nil {
		return "", err
	}

	// 截断日志防止 Token 溢出
	logStr := string(content)
	if len(logStr) > 8000 {
		logStr = logStr[len(logStr)-8000:]
	}

	resp, err := a.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "你是一个资深的 DevOps 专家。请分析这段 CI/CD 失败日志，给出根因分析和修复建议（Markdown格式）。",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: logStr,
				},
			},
		},
	)

	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}
