package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	cron "github.com/robfig/cron/v3"

	"lite-cicd/ai"
	"lite-cicd/config"
	"lite-cicd/core"
	"lite-cicd/executor"
)

type Engine struct {
	cfg      config.Config
	executor core.Executor
	agent    core.Agent
	mu       sync.Mutex
}

func (e *Engine) Trigger(repoName, branch string) {
	// 查找配置
	var targetRepo config.RepoConfig
	found := false
	for _, r := range e.cfg.Repos {
		if r.Name == repoName {
			targetRepo = r
			found = true
			break
		}
	}
	if !found {
		log.Printf("❌ 未找到仓库配置: %s", repoName)
		return
	}

	log.Printf("⚙️ 触发流水线: %s/%s", repoName, branch)
	logFile, err := e.executor.Run(context.Background(), targetRepo, branch)

	if err != nil {
		log.Printf("❌ 流水线失败: %v", err)
		// AI 介入分析
		if targetRepo.AutoAnalyze && e.agent != nil && logFile != "" {
			e.analyzeFailure(logFile)
		}
	} else {
		log.Printf("✅ 流水线成功，日志: %s", logFile)
	}
}

func (e *Engine) analyzeFailure(logPath string) {
	log.Println("🤖 正在请求 AI 分析失败原因...")
	analysis, err := e.agent.AnalyzeLog(logPath)
	if err != nil {
		log.Printf("AI 分析失败: %v", err)
		return
	}

	// 将分析结果写入同目录的 .analysis.md 文件
	analysisFile := logPath + ".analysis.md"
	os.WriteFile(analysisFile, []byte(analysis), 0644)
	log.Printf("🤖 AI 分析报告已生成: %s", analysisFile)
}

// MCPServer 暴露工具给外部 AI (如 Cursor, Claude)
type MCPServer struct {
	engine *Engine
}

// 模拟 MCP 的 Tool 定义结构
type MCPTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"input_schema"`
}

func NewMcpServer(engine *Engine) MCPServer {
	svr := MCPServer{
		engine: engine,
	}
	svr.engine = engine
	return svr
}

func (s *MCPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 这是一个极简的 MCP / JSON-RPC 实现
	// 实际 MCP 需要处理 SSE 或 Stdio，这里用 HTTP 模拟 Tool Call 接口

	if r.URL.Path == "/mcp/tools" {
		// 列出可用工具
		tools := []MCPTool{
			{
				Name:        "trigger_pipeline",
				Description: "Trigger a CI/CD pipeline for a specific repo and branch",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"repo":   map[string]string{"type": "string"},
						"branch": map[string]string{"type": "string"},
					},
				},
			},
			{
				Name:        "get_build_logs",
				Description: "Get the latest build logs and AI analysis",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"repo": map[string]string{"type": "string"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(tools)
		return
	}

	if r.URL.Path == "/mcp/call" {
		// 执行工具调用
		var req struct {
			Tool string            `json:"tool"`
			Args map[string]string `json:"args"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		switch req.Tool {
		case "trigger_pipeline":
			go s.engine.Trigger(req.Args["repo"], req.Args["branch"])
			fmt.Fprintf(w, "Pipeline triggered for %s", req.Args["repo"])
		case "get_build_logs":
			// 实现获取日志逻辑
			fmt.Fprintf(w, "Logs content...")
		}
	}
}

func main() {
	// 1. 初始化组件
	cfg := config.Config{
		Repos: []config.RepoConfig{
			{
				Name:        "backend-go",
				URL:         "https://github.com/user/backend",
				Branches:    []string{"main"},
				Dockerfile:  "Dockerfile",
				TestCmd:     "go test ./...",
				AutoAnalyze: true,
			},
		},
		Schedule: "@every 1h",
		LLMKey:   os.Getenv("OPENAI_API_KEY"),
	}

	// 创建日志目录
	os.MkdirAll("./logs", 0755)

	executor, _ := executor.NewDockerExecutor("./logs")
	aiAgent := ai.NewAIAgent(cfg.LLMKey, "")

	engine := &Engine{
		cfg:      cfg,
		executor: executor,
		agent:    aiAgent,
	}

	// 2. 启动 Cron
	c := cron.New()
	c.AddFunc(cfg.Schedule, func() {
		for _, r := range cfg.Repos {
			engine.Trigger(r.Name, r.Branches[0])
		}
	})
	c.Start()

	// 3. 启动 MCP / Webhook 服务器
	mcpServer := &MCPServer{
		engine: engine,
	}
	http.Handle("/mcp/", mcpServer)
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		// 简单的 Webhook 触发逻辑
		engine.Trigger("backend-go", "main")
		w.Write([]byte("OK"))
	})

	log.Println("SmartCI is running on :8080 (Cron + Webhook + MCP)")
	http.ListenAndServe(":8080", nil)
}
