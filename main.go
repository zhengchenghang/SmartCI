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
    cfg         config.Config
    executor    core.Executor
    bashExecutor core.BashExecutor
    agent       core.Agent
    mu          sync.Mutex
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

func (e *Engine) TriggerBashTask(taskName string) {
    // 查找bash任务配置
    var targetTask config.BashTaskConfig
    found := false
    for _, t := range e.cfg.BashTasks {
        if t.Name == taskName {
            targetTask = t
            found = true
            break
        }
    }
    if !found {
        log.Printf("❌ 未找到Bash任务配置: %s", taskName)
        return
    }

    log.Printf("⚙️ 触发Bash任务: %s", taskName)
    logFile, err := e.bashExecutor.RunBashTask(context.Background(), targetTask)

    if err != nil {
        log.Printf("❌ Bash任务失败: %v", err)
        // AI 介入分析
        if targetTask.AutoAnalyze && e.agent != nil && logFile != "" {
            e.analyzeFailure(logFile)
        }
    } else {
        log.Printf("✅ Bash任务成功，日志: %s", logFile)
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
                Name:        "trigger_bash_task",
                Description: "Trigger a bash task by name",
                InputSchema: map[string]any{
                    "type": "object",
                    "properties": map[string]any{
                        "task": map[string]string{"type": "string"},
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
        case "trigger_bash_task":
            go s.engine.TriggerBashTask(req.Args["task"])
            fmt.Fprintf(w, "Bash task triggered for %s", req.Args["task"])
        case "get_build_logs":
            // 实现获取日志逻辑
            fmt.Fprintf(w, "Logs content...")
        }
    }
}

func main() {
    // 1. 加载配置
    configFile := "config.yaml"
    if len(os.Args) > 1 {
        configFile = os.Args[1]
    }
    
    cfg, err := config.LoadConfig(configFile)
    if err != nil {
        log.Printf("⚠️  加载配置文件失败: %v，使用默认配置", err)
        // 使用默认配置
        cfg = config.Config{
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
            BashTasks: []config.BashTaskConfig{
                {
                    Name:        "backup-database",
                    Description: "备份数据库",
                    Schedule:    "0 2 * * *", // 每天凌晨2点
                    Command:     "pg_dump mydb > backup_$(date +%Y%m%d_%H%M%S).sql",
                    WorkingDir:  "/backups",
                    Timeout:     600,
                    AutoAnalyze: true,
                },
                {
                    Name:        "cleanup-logs",
                    Description: "清理旧日志文件",
                    Schedule:    "0 0 * * 0", // 每周日午夜
                    Command:     "find ./logs -name '*.log' -mtime +7 -delete",
                    WorkingDir:  "/home/engine/project",
                    Timeout:     300,
                    AutoAnalyze: false,
                },
            },
            Schedule: "@every 1h",
            LLMKey:   os.Getenv("OPENAI_API_KEY"),
        }
    }

    // 创建日志目录
    os.MkdirAll("./logs", 0755)

    dockerExecutor, _ := executor.NewDockerExecutor("./logs")
    bashExecutor, _ := executor.NewBashExecutor("./logs")
    aiAgent := ai.NewAIAgent(cfg.LLMKey, cfg.LLMBase)

    engine := &Engine{
        cfg:          cfg,
        executor:     dockerExecutor,
        bashExecutor: bashExecutor,
        agent:        aiAgent,
    }

    // 2. 启动 Cron - 全局调度
    c := cron.New()
    
    // 全局仓库任务调度
    c.AddFunc(cfg.Schedule, func() {
        for _, r := range cfg.Repos {
            engine.Trigger(r.Name, r.Branches[0])
        }
    })

    // Bash任务独立调度
    for _, task := range cfg.BashTasks {
        if task.Schedule != "" {
            taskName := task.Name // 创建局部变量避免闭包问题
            c.AddFunc(task.Schedule, func() {
                engine.TriggerBashTask(taskName)
            })
            log.Printf("📅 已注册Bash任务: %s (%s)", taskName, task.Schedule)
        }
    }
    
    c.Start()

    // 3. 启动 MCP / Webhook 服务器
    mcpServer := &MCPServer{
        engine: engine,
    }
    http.Handle("/mcp/", mcpServer)
    http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
        // 简单的 Webhook 触发逻辑
        repo := r.URL.Query().Get("repo")
        branch := r.URL.Query().Get("branch")
        if repo == "" {
            repo = "backend-go"
        }
        if branch == "" {
            branch = "main"
        }
        engine.Trigger(repo, branch)
        w.Write([]byte("OK"))
    })

    // 添加bash任务webhook触发
    http.HandleFunc("/webhook/bash", func(w http.ResponseWriter, r *http.Request) {
        taskName := r.URL.Query().Get("task")
        if taskName == "" {
            http.Error(w, "Missing task parameter", http.StatusBadRequest)
            return
        }
        engine.TriggerBashTask(taskName)
        w.Write([]byte("Bash task triggered"))
    })

    // 添加配置查看端点
    http.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        // 返回配置摘要，不包含敏感信息
        summary := map[string]interface{}{
            "repos_count":      len(cfg.Repos),
            "bash_tasks_count": len(cfg.BashTasks),
            "schedule":         cfg.Schedule,
            "llm_configured":   cfg.LLMKey != "",
        }
        json.NewEncoder(w).Encode(summary)
    })

    log.Printf("SmartCI is running on :8080 (Cron + Webhook + MCP + Bash Tasks)")
    log.Printf("配置文件: %s", configFile)
    log.Printf("仓库数量: %d, Bash任务数量: %d", len(cfg.Repos), len(cfg.BashTasks))
    http.ListenAndServe(":8080", nil)
}
