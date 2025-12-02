package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "sync"
    "syscall"
    "time"

    cron "github.com/robfig/cron/v3"

    "lite-cicd/ai"
    "lite-cicd/config"
    "lite-cicd/core"
    "lite-cicd/executor"
    "lite-cicd/oauth"
    "lite-cicd/webhook"
)

type Engine struct {
    cfg          config.Config
    executor     core.Executor
    bashExecutor core.BashExecutor
    agent        core.Agent
    cron         *cron.Cron
    mu           sync.Mutex
    running      bool
    taskStatus   map[string]bool         // 任务运行状态
    taskEntries  map[string]cron.EntryID // 任务cron entry ID映射
    shutdownChan chan struct{}           // 服务器关闭信号
}

type Server struct {
    engine          *Engine
    cfg             *config.Config
    server          *http.Server
    oauthProviders  map[string]oauth.Provider
    webhookHandlers map[string]*webhook.Handler
}

// APIRequest API请求结构
type APIRequest struct {
    Command string                 `json:"command"`
    Args    map[string]interface{} `json:"args"`
}

// APIResponse API响应结构
type APIResponse struct {
    Success bool        `json:"success"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

func NewEngine(cfg config.Config) *Engine {
    dockerExecutor, _ := executor.NewDockerExecutor("./logs")
    bashExecutor, _ := executor.NewBashExecutor("./logs")
    aiAgent := ai.NewAIAgent(cfg.LLMKey, cfg.LLMBase)

    return &Engine{
        cfg:          cfg,
        executor:     dockerExecutor,
        bashExecutor: bashExecutor,
        agent:        aiAgent,
        cron:         cron.New(),
        taskStatus:   make(map[string]bool),
        taskEntries:  make(map[string]cron.EntryID),
        shutdownChan: make(chan struct{}),
    }
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
    result, err := e.executor.Run(context.Background(), targetRepo, branch)

    if err != nil {
        log.Printf("❌ 流水线失败: %v", err)
        // 兼容旧的AutoAnalyze配置或使用新的AI配置
        if result != nil && e.agent != nil {
            if targetRepo.AutoAnalyze && result.LogFile != "" {
                e.analyzeFailure(result.LogFile)
            }
            // 使用新的AI配置
            if targetRepo.AI.Enabled {
                e.invokeAI(targetRepo.AI, result)
            }
        }
    } else {
        if result != nil {
            log.Printf("✅ 流水线成功，任务ID: %s, 日志: %s", result.TaskID, result.LogFile)
            // 即使成功也可能需要AI分析（根据配置）
            if targetRepo.AI.Enabled && e.agent != nil {
                e.invokeAI(targetRepo.AI, result)
            }
        } else {
            log.Printf("✅ 流水线成功")
        }
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

    e.mu.Lock()
    e.taskStatus[taskName] = true
    e.mu.Unlock()

    log.Printf("⚙️ 触发Bash任务: %s", taskName)
    result, err := e.bashExecutor.RunBashTask(context.Background(), targetTask)

    e.mu.Lock()
    e.taskStatus[taskName] = false
    e.mu.Unlock()

    if err != nil {
        log.Printf("❌ Bash任务失败: %v", err)
        // 兼容旧的AutoAnalyze配置或使用新的AI配置
        if result != nil && e.agent != nil {
            if targetTask.AutoAnalyze && result.LogFile != "" {
                e.analyzeFailure(result.LogFile)
            }
            // 使用新的AI配置
            if targetTask.AI.Enabled {
                e.invokeAI(targetTask.AI, result)
            }
        }
    } else {
        if result != nil {
            log.Printf("✅ Bash任务成功，任务ID: %s, 日志: %s", result.TaskID, result.LogFile)
            // 即使成功也可能需要AI分析（根据配置）
            if targetTask.AI.Enabled && e.agent != nil {
                e.invokeAI(targetTask.AI, result)
            }
        } else {
            log.Printf("✅ Bash任务成功")
        }
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

// invokeAI 调用AI分析（使用新的AI配置）
func (e *Engine) invokeAI(aiConfig config.AIConfig, result *core.TaskResult) {
    log.Println("🤖 正在调用 AI 分析...")
    
    err := core.InvokeAI(e.agent, aiConfig, result.TaskDir, result)
    if err != nil {
        log.Printf("❌ AI 分析失败: %v", err)
        return
    }
    
    log.Printf("✅ AI 分析完成，任务ID: %s", result.TaskID)
}

func (e *Engine) StartCron() {
    // 全局仓库任务调度
    e.cron.AddFunc(e.cfg.Schedule, func() {
        for _, r := range e.cfg.Repos {
            e.Trigger(r.Name, r.Branches[0])
        }
    })

    // Bash任务独立调度
    for _, task := range e.cfg.BashTasks {
        if task.Schedule != "" {
            taskName := task.Name // 创建局部变量避免闭包问题
            entryID, err := e.cron.AddFunc(task.Schedule, func() {
                e.TriggerBashTask(taskName)
            })
            if err != nil {
                log.Printf("❌ 注册Bash任务失败: %s, 错误: %v", taskName, err)
                continue
            }
            e.taskEntries[taskName] = entryID
            log.Printf("📅 已注册Bash任务: %s (%s) [ID: %d]", taskName, task.Schedule, entryID)
        }
    }

    e.cron.Start()
    e.running = true
    log.Printf("✅ Cron调度器已启动，共注册 %d 个周期性Bash任务", len(e.taskEntries))
}

func (e *Engine) StopCron() {
    if e.cron != nil {
        ctx := e.cron.Stop()
        select {
        case <-ctx.Done():
            log.Printf("✅ Cron调度器已停止")
        case <-time.After(time.Second * 10):
            log.Printf("⚠️ Cron调度器停止超时")
        }
        e.running = false
    }
}

// StopBashTask 停止指定的周期性Bash任务
func (e *Engine) StopBashTask(taskName string) error {
    e.mu.Lock()
    defer e.mu.Unlock()

    entryID, exists := e.taskEntries[taskName]
    if !exists {
        return fmt.Errorf("任务 '%s' 没有注册周期性调度", taskName)
    }

    // 从cron中移除任务
    e.cron.Remove(entryID)
    delete(e.taskEntries, taskName)

    log.Printf("🛑 已停止周期性Bash任务: %s [ID: %d]", taskName, entryID)
    return nil
}

// StartBashTask 启动指定的周期性Bash任务
func (e *Engine) StartBashTask(taskName string) error {
    e.mu.Lock()
    defer e.mu.Unlock()

    // 检查任务是否已经在运行
    if _, exists := e.taskEntries[taskName]; exists {
        return fmt.Errorf("任务 '%s' 已经在周期性运行中", taskName)
    }

    // 查找任务配置
    var targetTask config.BashTaskConfig
    found := false
    for _, task := range e.cfg.BashTasks {
        if task.Name == taskName {
            targetTask = task
            found = true
            break
        }
    }
    if !found {
        return fmt.Errorf("未找到Bash任务配置: %s", taskName)
    }

    if targetTask.Schedule == "" {
        return fmt.Errorf("任务 '%s' 没有配置周期性调度", taskName)
    }

    // 添加到cron调度
    entryID, err := e.cron.AddFunc(targetTask.Schedule, func() {
        e.TriggerBashTask(taskName)
    })
    if err != nil {
        return fmt.Errorf("注册Bash任务失败: %v", err)
    }

    e.taskEntries[taskName] = entryID
    log.Printf("📅 已启动周期性Bash任务: %s (%s) [ID: %d]", taskName, targetTask.Schedule, entryID)
    return nil
}

func (e *Engine) GetTaskStatus(taskName string) map[string]interface{} {
    e.mu.Lock()
    defer e.mu.Unlock()

    if taskName != "" {
        status, exists := e.taskStatus[taskName]
        isScheduled, scheduled := e.taskEntries[taskName]
        return map[string]interface{}{
            "task_name":   taskName,
            "running":     exists && status,
            "scheduled":   scheduled,
            "schedule_id": isScheduled,
        }
    }

    // 返回所有任务状态
    status := make(map[string]bool)
    scheduled := make(map[string]bool)
    scheduleIds := make(map[string]int)

    for name, running := range e.taskStatus {
        status[name] = running
    }

    for name, entryID := range e.taskEntries {
        scheduled[name] = true
        scheduleIds[name] = int(entryID)
    }

    return map[string]interface{}{
        "tasks":        status,
        "scheduled":    scheduled,
        "schedule_ids": scheduleIds,
        "cron_running": e.running,
    }
}

// NewServer 创建新的服务器实例
func NewServer(cfg *config.Config) *Server {
    engine := NewEngine(*cfg)
    server := &Server{
        engine:          engine,
        cfg:             cfg,
        oauthProviders:  make(map[string]oauth.Provider),
        webhookHandlers: make(map[string]*webhook.Handler),
    }

    // 初始化OAuth提供商
    server.initOAuthProviders()

    // 初始化Webhook处理器
    server.initWebhookHandlers()

    return server
}

// initOAuthProviders 初始化OAuth提供商
func (s *Server) initOAuthProviders() {
    for _, oauthCfg := range s.cfg.OAuth {
        var provider oauth.Provider

        switch oauthCfg.Name {
        case "github":
            provider = oauth.NewGitHubProvider(
                oauthCfg.ClientID,
                oauthCfg.ClientSecret,
                oauthCfg.RedirectURL,
                oauthCfg.Scopes,
            )
        default:
            log.Printf("⚠️ 未知的OAuth提供商: %s", oauthCfg.Name)
            continue
        }

        s.oauthProviders[oauthCfg.Name] = provider
        log.Printf("✅ 已初始化OAuth提供商: %s", oauthCfg.Name)
    }
}

// initWebhookHandlers 初始化Webhook处理器
func (s *Server) initWebhookHandlers() {
    for _, webhookCfg := range s.cfg.Webhooks {
        provider := s.oauthProviders[webhookCfg.Provider]

        handler := webhook.NewHandler(webhookCfg, provider, s.executeWebhookAction)
        s.webhookHandlers[webhookCfg.Path] = handler

        log.Printf("✅ 已注册Webhook: %s -> %s", webhookCfg.Path, webhookCfg.Name)
    }
}

// executeWebhookAction 执行webhook动作
func (s *Server) executeWebhookAction(ctx context.Context, action config.WebhookAction, payload interface{}) error {
    log.Printf("⚙️ 执行Webhook动作: %s", action.Type)

    switch action.Type {
    case "command":
        // 执行shell命令
        if action.Command == "" {
            return fmt.Errorf("command类型的action必须指定command字段")
        }

        // 创建临时任务配置
        taskCfg := config.BashTaskConfig{
            Name:       "webhook-command",
            Command:    action.Command,
            WorkingDir: action.WorkingDir,
            Timeout:    action.Timeout,
        }
        if taskCfg.Timeout == 0 {
            taskCfg.Timeout = 300
        }

        _, err := s.engine.bashExecutor.RunBashTask(ctx, taskCfg)
        return err

    case "script":
        // 执行shell脚本
        if action.Script == "" {
            return fmt.Errorf("script类型的action必须指定script字段")
        }

        taskCfg := config.BashTaskConfig{
            Name:       "webhook-script",
            ScriptFile: action.Script,
            WorkingDir: action.WorkingDir,
            Timeout:    action.Timeout,
        }
        if taskCfg.Timeout == 0 {
            taskCfg.Timeout = 300
        }

        _, err := s.engine.bashExecutor.RunBashTask(ctx, taskCfg)
        return err

    case "task":
        // 执行已配置的任务
        if action.Task == "" {
            return fmt.Errorf("task类型的action必须指定task字段")
        }

        go s.engine.TriggerBashTask(action.Task)
        return nil

    default:
        return fmt.Errorf("未知的action类型: %s", action.Type)
    }
}

// Start 启动服务器
func (s *Server) Start(host string, port int) error {
    // 创建日志目录
    os.MkdirAll("./logs", 0755)

    // 启动Cron调度器
    s.engine.StartCron()

    // 创建HTTP服务器
    addr := fmt.Sprintf("%s:%d", host, port)
    s.server = &http.Server{
        Addr: addr,
    }

    // 注册路由
    s.setupRoutes()

    log.Printf("🚀 SmartCI服务器启动在 %s", addr)
    log.Printf("📋 配置文件加载完成，仓库数量: %d, Bash任务数量: %d", len(s.cfg.Repos), len(s.cfg.BashTasks))

    return s.server.ListenAndServe()
}

// Stop 停止服务器
func (s *Server) Stop() error {
    s.engine.mu.Lock()
    defer s.engine.mu.Unlock()

    log.Printf("🛑 正在停止SmartCI服务器...")

    // 停止Cron调度器
    s.engine.StopCron()

    // 停止HTTP服务器
    var err error
    if s.server != nil {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        err = s.server.Shutdown(ctx)
    }

    // 发送关闭信号给主程序（防止重复关闭）
    if s.engine.shutdownChan != nil {
        select {
        case <-s.engine.shutdownChan:
            // channel已经关闭
        default:
            close(s.engine.shutdownChan)
        }
        s.engine.shutdownChan = nil
    }

    return err
}

// setupRoutes 设置HTTP路由
func (s *Server) setupRoutes() {
    // API命令路由
    http.HandleFunc("/api/command", s.handleCommand)

    // OAuth路由
    http.HandleFunc("/oauth/authorize", s.handleOAuthAuthorize)
    http.HandleFunc("/oauth/callback", s.handleOAuthCallback)

    // Webhook路由（动态注册）
    for path, handler := range s.webhookHandlers {
        http.Handle(path, handler)
    }

    // 兼容性路由
    http.HandleFunc("/mcp/", s.handleMCP)
    http.HandleFunc("/webhook", s.handleWebhook)
    http.HandleFunc("/webhook/bash", s.handleBashWebhook)
    http.HandleFunc("/config", s.handleConfig)
    http.HandleFunc("/health", s.handleHealth)
}

// handleCommand 处理API命令请求
func (s *Server) handleCommand(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // 检查认证
    if s.cfg.Server.AuthToken != "" {
        authHeader := r.Header.Get("Authorization")
        if authHeader != "Bearer "+s.cfg.Server.AuthToken {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
    }

    var req APIRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response := APIResponse{
            Success: false,
            Message: "解析请求失败: " + err.Error(),
        }
        json.NewEncoder(w).Encode(response)
        return
    }

    response := s.executeCommand(req.Command, req.Args)
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// executeCommand 执行命令
func (s *Server) executeCommand(command string, args map[string]interface{}) APIResponse {
    switch command {
    case "server-up":
        return APIResponse{
            Success: true,
            Message: "服务器已在运行",
        }
    case "server-down":
        go func() {
            time.Sleep(1 * time.Second)
            s.Stop()
        }()
        return APIResponse{
            Success: true,
            Message: "服务器正在停止...",
        }
    case "run":
        taskName, ok := args["task_name"].(string)
        if !ok {
            return APIResponse{
                Success: false,
                Message: "缺少任务名称参数",
            }
        }
        go s.engine.TriggerBashTask(taskName)
        return APIResponse{
            Success: true,
            Message: fmt.Sprintf("任务 '%s' 已启动", taskName),
        }
    case "start":
        taskName, ok := args["task_name"].(string)
        if !ok {
            return APIResponse{
                Success: false,
                Message: "缺少任务名称参数",
            }
        }
        // 启动任务的周期性调度
        err := s.engine.StartBashTask(taskName)
        if err != nil {
            return APIResponse{
                Success: false,
                Message: err.Error(),
            }
        }
        return APIResponse{
            Success: true,
            Message: fmt.Sprintf("任务 '%s' 的周期性调度已启动", taskName),
        }
    case "stop":
        taskName, ok := args["task_name"].(string)
        if !ok {
            return APIResponse{
                Success: false,
                Message: "缺少任务名称参数",
            }
        }
        // 停止任务的周期性调度
        err := s.engine.StopBashTask(taskName)
        if err != nil {
            return APIResponse{
                Success: false,
                Message: err.Error(),
            }
        }
        return APIResponse{
            Success: true,
            Message: fmt.Sprintf("任务 '%s' 的周期性调度已停止", taskName),
        }
    case "status":
        taskName, _ := args["task_name"].(string)
        status := s.engine.GetTaskStatus(taskName)
        return APIResponse{
            Success: true,
            Message: "任务状态查询成功",
            Data:    status,
        }
    case "logs":
        taskName, ok := args["task_name"].(string)
        if !ok {
            return APIResponse{
                Success: false,
                Message: "缺少任务名称参数",
            }
        }
        lines, _ := args["lines"].(int)
        if lines == 0 {
            lines = 100 // 默认显示100行
        }
        // 这里可以实现日志读取逻辑
        return APIResponse{
            Success: true,
            Message: fmt.Sprintf("显示任务 '%s' 的最近 %d 行日志", taskName, lines),
            Data: map[string]interface{}{
                "task_name": taskName,
                "lines":     lines,
                "content":   "日志内容待实现...",
            },
        }
    case "config":
        return APIResponse{
            Success: true,
            Message: "配置信息",
            Data: map[string]interface{}{
                "repos_count":      len(s.cfg.Repos),
                "bash_tasks_count": len(s.cfg.BashTasks),
                "schedule":         s.cfg.Schedule,
                "llm_configured":   s.cfg.LLMKey != "",
                "server":           s.cfg.Server,
            },
        }
    case "reload":
        // 这里可以实现配置重新加载逻辑
        return APIResponse{
            Success: true,
            Message: "配置重新加载功能待实现",
        }
    case "list":
        tasks := make([]string, 0, len(s.cfg.BashTasks))
        for _, task := range s.cfg.BashTasks {
            tasks = append(tasks, task.Name)
        }
        return APIResponse{
            Success: true,
            Message: "可用任务列表",
            Data: map[string]interface{}{
                "bash_tasks": tasks,
                "repos":      getRepoNames(s.cfg.Repos),
            },
        }
    case "health":
        return APIResponse{
            Success: true,
            Message: "服务器运行正常",
            Data: map[string]interface{}{
                "status":       "healthy",
                "uptime":       "运行时间待实现",
                "version":      "1.0.0",
                "cron_running": s.engine.running,
            },
        }
    default:
        return APIResponse{
            Success: false,
            Message: fmt.Sprintf("未知命令: %s", command),
        }
    }
}

func getRepoNames(repos []config.RepoConfig) []string {
    names := make([]string, len(repos))
    for i, repo := range repos {
        names[i] = repo.Name
    }
    return names
}

// handleMCP 处理MCP兼容请求
func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
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
            fmt.Fprintf(w, "Logs content...")
        }
    }
}

// handleWebhook 处理webhook请求
func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
    repo := r.URL.Query().Get("repo")
    branch := r.URL.Query().Get("branch")
    if repo == "" {
        repo = "backend-go"
    }
    if branch == "" {
        branch = "main"
    }
    s.engine.Trigger(repo, branch)
    w.Write([]byte("OK"))
}

// handleBashWebhook 处理bash任务webhook请求
func (s *Server) handleBashWebhook(w http.ResponseWriter, r *http.Request) {
    taskName := r.URL.Query().Get("task")
    if taskName == "" {
        http.Error(w, "Missing task parameter", http.StatusBadRequest)
        return
    }
    s.engine.TriggerBashTask(taskName)
    w.Write([]byte("Bash task triggered"))
}

// handleConfig 处理配置查看请求
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    summary := map[string]interface{}{
        "repos_count":      len(s.cfg.Repos),
        "bash_tasks_count": len(s.cfg.BashTasks),
        "schedule":         s.cfg.Schedule,
        "llm_configured":   s.cfg.LLMKey != "",
        "server":           s.cfg.Server,
    }
    json.NewEncoder(w).Encode(summary)
}

// handleHealth 处理健康检查请求
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    status := map[string]interface{}{
        "status":       "healthy",
        "version":      "1.0.0",
        "cron_running": s.engine.running,
        "uptime":       "运行时间待实现",
    }
    json.NewEncoder(w).Encode(status)
}

// handleOAuthAuthorize 处理OAuth授权请求
func (s *Server) handleOAuthAuthorize(w http.ResponseWriter, r *http.Request) {
    provider := r.URL.Query().Get("provider")
    if provider == "" {
        http.Error(w, "Missing provider parameter", http.StatusBadRequest)
        return
    }

    oauthProvider, exists := s.oauthProviders[provider]
    if !exists {
        http.Error(w, "Unknown OAuth provider", http.StatusBadRequest)
        return
    }

    state := r.URL.Query().Get("state")
    if state == "" {
        state = "random-state-" + fmt.Sprint(time.Now().Unix())
    }

    authURL := oauthProvider.GetAuthURL(state)
    http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// handleOAuthCallback 处理OAuth回调请求
func (s *Server) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
    provider := r.URL.Query().Get("provider")
    code := r.URL.Query().Get("code")

    if provider == "" || code == "" {
        http.Error(w, "Missing parameters", http.StatusBadRequest)
        return
    }

    oauthProvider, exists := s.oauthProviders[provider]
    if !exists {
        http.Error(w, "Unknown OAuth provider", http.StatusBadRequest)
        return
    }

    // 交换访问令牌
    token, err := oauthProvider.ExchangeToken(r.Context(), code)
    if err != nil {
        log.Printf("❌ OAuth令牌交换失败: %v", err)
        http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
        return
    }

    // 获取用户信息
    userInfo, err := oauthProvider.GetUserInfo(r.Context(), token.AccessToken)
    if err != nil {
        log.Printf("❌ 获取用户信息失败: %v", err)
        http.Error(w, "Failed to get user info", http.StatusInternalServerError)
        return
    }

    log.Printf("✅ OAuth授权成功: %s", provider)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "success":   true,
        "provider":  provider,
        "token":     token,
        "user_info": userInfo,
    })
}

// MCPTool MCP工具定义结构
type MCPTool struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    InputSchema any    `json:"input_schema"`
}

func main() {
    // 解析命令行参数
    var (
        configFile = flag.String("config", "config.yaml", "配置文件路径")
        mode       = flag.String("mode", "server", "运行模式: server 或 client")
        host       = flag.String("host", "", "服务器主机地址（覆盖配置文件）")
        port       = flag.Int("port", 0, "服务器端口（覆盖配置文件）")
    )
    flag.Parse()

    // 加载配置
    cfg, err := config.LoadConfig(*configFile)
    if err != nil {
        log.Printf("⚠️  加载配置文件失败: %v，使用默认配置", err)
        // 使用默认配置
        cfg = config.Config{
            Server: config.ServerConfig{
                Host: "localhost",
                Port: 8080,
            },
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
                    Schedule:    "0 2 * * *",
                    Command:     "pg_dump mydb > backup_$(date +%Y%m%d_%H%M%S).sql",
                    WorkingDir:  "/backups",
                    Timeout:     600,
                    AutoAnalyze: true,
                },
                {
                    Name:        "cleanup-logs",
                    Description: "清理旧日志文件",
                    Schedule:    "0 0 * * 0",
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

    // 覆盖配置文件中的服务器设置
    if *host != "" {
        cfg.Server.Host = *host
    }
    if *port != 0 {
        cfg.Server.Port = *port
    }

    switch *mode {
    case "server":
        runServer(cfg)
    case "client":
        log.Printf("❌ 客户端模式请使用 ./client 可执行文件")
        os.Exit(1)
    default:
        log.Printf("❌ 未知模式: %s，支持的模式: server, client", *mode)
        os.Exit(1)
    }
}

func runServer(cfg config.Config) {
    // 创建服务器实例
    server := NewServer(&cfg)

    // 设置信号处理
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    // 在goroutine中启动服务器
    go func() {
        if err := server.Start(cfg.Server.Host, cfg.Server.Port); err != nil && err != http.ErrServerClosed {
            log.Printf("❌ 服务器启动失败: %v", err)
            os.Exit(1)
        }
    }()

    // 等待信号或服务器关闭信号
    select {
    case <-sigChan:
        log.Printf("📡 接收到系统停止信号，正在优雅关闭服务器...")
    case <-server.engine.shutdownChan:
        log.Printf("📡 接收到服务器关闭命令，正在优雅关闭服务器...")
    }

    // 停止服务器
    if err := server.Stop(); err != nil {
        log.Printf("❌ 服务器停止失败: %v", err)
    } else {
        log.Printf("✅ 服务器已安全停止")
    }

    // 退出程序
    os.Exit(0)
}
