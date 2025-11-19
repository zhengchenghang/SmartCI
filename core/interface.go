package core

import (
	"context"
	"lite-cicd/config"
)

// Executor 定义构建能力的接口，方便扩展非 Docker 环境
type Executor interface {
	Run(ctx context.Context, repo config.RepoConfig, branch string) (string, error)
}

// Agent 定义 AI 能力接口
type Agent interface {
	AnalyzeLog(logContent string) (string, error)
}
