package main

import (
    "bytes"
    "encoding/json"
    "flag"
    "fmt"
    "io"
    "net/http"
    "os"
    "strings"

    "lite-cicd/config"
)

// Client 客户端结构
type Client struct {
    serverURL string
    authToken string
}

// CommandRequest 命令请求结构
type CommandRequest struct {
    Command string                 `json:"command"`
    Args    map[string]interface{} `json:"args"`
}

// CommandResponse 命令响应结构
type CommandResponse struct {
    Success bool        `json:"success"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

// NewClient 创建新的客户端
func NewClient(serverURL, authToken string) *Client {
    return &Client{
        serverURL: serverURL,
        authToken: authToken,
    }
}

// sendCommand 发送命令到服务器
func (c *Client) sendCommand(command string, args map[string]interface{}) (*CommandResponse, error) {
    req := CommandRequest{
        Command: command,
        Args:    args,
    }

    jsonData, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("序列化请求失败: %v", err)
    }

    httpReq, err := http.NewRequest("POST", c.serverURL+"/api/command", bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, fmt.Errorf("创建请求失败: %v", err)
    }

    httpReq.Header.Set("Content-Type", "application/json")
    if c.authToken != "" {
        httpReq.Header.Set("Authorization", "Bearer "+c.authToken)
    }

    client := &http.Client{}
    resp, err := client.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("发送请求失败: %v", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("读取响应失败: %v", err)
    }

    var response CommandResponse
    if err := json.Unmarshal(body, &response); err != nil {
        return nil, fmt.Errorf("解析响应失败: %v", err)
    }

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("服务器错误: %s", response.Message)
    }

    return &response, nil
}

func main() {
    var configFile = flag.String("config", "config.yaml", "配置文件路径")
    var server = flag.String("server", "", "服务器地址 (格式: host:port)")
    var command = flag.String("command", "", "要执行的命令")
    flag.Parse()

    if *command == "" {
        fmt.Println("请指定要执行的命令")
        printUsage()
        os.Exit(1)
    }

    // 加载配置获取服务器地址
    serverURL := *server
    if serverURL == "" {
        cfg, err := loadConfig(*configFile)
        if err != nil {
            fmt.Printf("加载配置文件失败: %v\n", err)
            os.Exit(1)
        }
        serverURL = fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
    }

    // 确保URL格式正确
    if !strings.HasPrefix(serverURL, "http://") && !strings.HasPrefix(serverURL, "https://") {
        serverURL = "http://" + serverURL
    }

    // 加载配置获取认证令牌
    cfg, _ := loadConfig(*configFile)
    authToken := ""
    if cfg != nil {
        authToken = cfg.Server.AuthToken
    }

    client := NewClient(serverURL, authToken)

    // 解析命令和参数
    cmd, args := parseCommand(*command, flag.Args())

    // 发送命令
    response, err := client.sendCommand(cmd, args)
    if err != nil {
        fmt.Printf("❌ 命令执行失败: %v\n", err)
        os.Exit(1)
    }

    if !response.Success {
        fmt.Printf("❌ 服务器返回错误: %s\n", response.Message)
        os.Exit(1)
    }

    fmt.Printf("✅ %s\n", response.Message)
    if response.Data != nil {
        printData(response.Data)
    }
}

func printUsage() {
    fmt.Println("SmartCI Client - 远程CI/CD管理工具")
    fmt.Println("")
    fmt.Println("基本用法:")
    fmt.Println("  ./client -command <命令> [参数]")
    fmt.Println("")
    fmt.Println("可用命令:")
    fmt.Println("  server-up [port] [host]     - 启动服务器（可覆盖配置文件中的端口和主机）")
    fmt.Println("  server-down                 - 停止服务器")
    fmt.Println("  run <task_name>             - 运行一次指定任务")
    fmt.Println("  start <task_name>           - 启动指定任务（周期调度）")
    fmt.Println("  stop <task_name>            - 停止指定任务")
    fmt.Println("  status [task_name]          - 查看任务状态（不指定任务名则显示所有）")
    fmt.Println("  logs <task_name> [lines]    - 查看任务日志")
    fmt.Println("  config                      - 查看当前配置")
    fmt.Println("  reload                      - 重新加载配置文件")
    fmt.Println("  list                        - 列出所有可用任务")
    fmt.Println("  health                      - 检查服务器健康状态")
    fmt.Println("")
    fmt.Println("参数:")
    fmt.Println("  -config string    配置文件路径 (默认: config.yaml)")
    fmt.Println("  -server string    服务器地址 (格式: host:port)")
    fmt.Println("  -command string   要执行的命令")
    fmt.Println("")
    fmt.Println("示例:")
    fmt.Println("  ./client -command \"run backup-database\"")
    fmt.Println("  ./client -command \"start system-monitor\"")
    fmt.Println("  ./client -command \"status\"")
    fmt.Println("  ./client -command \"logs backup-database 100\"")
    fmt.Println("  ./client -command \"server-up 9090\"")
}

func parseCommand(command string, args []string) (string, map[string]interface{}) {
    parts := strings.Fields(command)
    if len(parts) == 0 {
        return "", nil
    }

    cmd := parts[0]
    cmdArgs := make(map[string]interface{})

    switch cmd {
    case "server-up":
        if len(parts) > 1 {
            if port, err := parseInt(parts[1]); err == nil {
                cmdArgs["port"] = port
            }
        }
        if len(parts) > 2 {
            cmdArgs["host"] = parts[2]
        }
    case "run", "start", "stop":
        if len(parts) > 1 {
            cmdArgs["task_name"] = parts[1]
        }
    case "status":
        if len(parts) > 1 {
            cmdArgs["task_name"] = parts[1]
        }
    case "logs":
        if len(parts) > 1 {
            cmdArgs["task_name"] = parts[1]
        }
        if len(parts) > 2 {
            if lines, err := parseInt(parts[2]); err == nil {
                cmdArgs["lines"] = lines
            }
        }
    }

    // 添加额外的命令行参数
    for i, arg := range args {
        cmdArgs[fmt.Sprintf("arg_%d", i)] = arg
    }

    return cmd, cmdArgs
}

func parseInt(s string) (int, error) {
    var result int
    _, err := fmt.Sscanf(s, "%d", &result)
    return result, err
}

func printData(data interface{}) {
    switch v := data.(type) {
    case map[string]interface{}:
        for key, value := range v {
            fmt.Printf("  %s: %v\n", key, value)
        }
    case []interface{}:
        for i, item := range v {
            fmt.Printf("  [%d] %v\n", i, item)
        }
    default:
        fmt.Printf("  %v\n", v)
    }
}

func loadConfig(configFile string) (*config.Config, error) {
    cfg, err := config.LoadConfig(configFile)
    if err != nil {
        return nil, err
    }
    return &cfg, nil
}