package core

import (
    "context"
    "lite-cicd/config"
)

// Executor 定义构建能力的接口，方便扩展非 Docker 环境
type Executor interface {
    Run(ctx context.Context, repo config.RepoConfig, branch string) (string, error)
}

// BashExecutor 定义Bash任务执行接口
type BashExecutor interface {
    RunBashTask(ctx context.Context, task config.BashTaskConfig) (string, error)
}

// Agent 定义 AI 能力接口
type Agent interface {
    AnalyzeLog(logContent string) (string, error)
}
