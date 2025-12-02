package core

import (
    "context"
    "lite-cicd/config"
)

// TaskResult 任务执行结果
type TaskResult struct {
    TaskID  string // 任务ID
    TaskDir string // 任务目录
    LogFile string // 日志文件路径
    Error   error  // 执行错误
}

// Executor 定义构建能力的接口，方便扩展非 Docker 环境
type Executor interface {
    Run(ctx context.Context, repo config.RepoConfig, branch string) (*TaskResult, error)
}

// BashExecutor 定义Bash任务执行接口
type BashExecutor interface {
    RunBashTask(ctx context.Context, task config.BashTaskConfig) (*TaskResult, error)
}

// Agent 定义 AI 能力接口
type Agent interface {
    AnalyzeLog(logContent string) (string, error)
    // AnalyzeWithContext 使用自定义上下文和Prompt进行AI分析
    AnalyzeWithContext(prompt string, context map[string]string) (string, error)
}
