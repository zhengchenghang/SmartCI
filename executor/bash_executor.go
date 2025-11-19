package executor

import (
    "context"
    "fmt"
    "io"
    "lite-cicd/config"
    "log"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"
)

type BashExecutor struct {
    logDir string
}

func NewBashExecutor(logDir string) (*BashExecutor, error) {
    return &BashExecutor{logDir: logDir}, nil
}

func (e *BashExecutor) RunBashTask(ctx context.Context, task config.BashTaskConfig) (string, error) {
    // 生成日志文件路径
    logFile := filepath.Join(e.logDir, fmt.Sprintf("bash-%s-%d.log", task.Name, time.Now().Unix()))
    
    // 确定要执行的命令
    var command string
    var err error
    
    if task.ScriptFile != "" {
        // 从文件读取脚本
        command, err = e.readScriptFile(task.ScriptFile)
        if err != nil {
            return "", fmt.Errorf("读取脚本文件失败: %v", err)
        }
    } else if task.Command != "" {
        // 使用内联命令
        command = task.Command
    } else {
        return "", fmt.Errorf("未指定命令或脚本文件")
    }

    // 设置超时
    timeout := time.Duration(task.Timeout) * time.Second
    if task.Timeout == 0 {
        timeout = 300 * time.Second // 默认5分钟
    }

    // 创建带超时的context
    if timeout > 0 {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, timeout)
        defer cancel()
    }

    // 执行bash命令
    log.Printf("🔧 [Bash] 执行任务: %s", task.Name)
    log.Printf("📝 [Bash] 命令: %s", strings.TrimSpace(command))
    
    if task.WorkingDir != "" {
        log.Printf("📁 [Bash] 工作目录: %s", task.WorkingDir)
    }

    err = e.runBashCommand(ctx, command, task.WorkingDir, logFile)
    if err != nil {
        return logFile, fmt.Errorf("bash任务执行失败: %v", err)
    }

    log.Printf("✅ [Bash] 任务完成: %s", task.Name)
    return logFile, nil
}

func (e *BashExecutor) readScriptFile(scriptFile string) (string, error) {
    // 检查文件是否存在
    if _, err := os.Stat(scriptFile); os.IsNotExist(err) {
        return "", fmt.Errorf("脚本文件不存在: %s", scriptFile)
    }

    // 读取文件内容
    content, err := os.ReadFile(scriptFile)
    if err != nil {
        return "", fmt.Errorf("读取脚本文件失败: %v", err)
    }

    return string(content), nil
}

func (e *BashExecutor) runBashCommand(ctx context.Context, command, workingDir, logFile string) error {
    // 创建日志文件
    logF, err := os.Create(logFile)
    if err != nil {
        return fmt.Errorf("创建日志文件失败: %v", err)
    }
    defer logF.Close()

    // 创建bash命令
    cmd := exec.CommandContext(ctx, "bash", "-c", command)
    
    // 设置工作目录
    if workingDir != "" {
        cmd.Dir = workingDir
    }

    // 设置环境变量
    cmd.Env = os.Environ()

    // 创建管道来捕获输出
    stdoutPipe, err := cmd.StdoutPipe()
    if err != nil {
        return fmt.Errorf("创建stdout管道失败: %v", err)
    }
    stderrPipe, err := cmd.StderrPipe()
    if err != nil {
        return fmt.Errorf("创建stderr管道失败: %v", err)
    }

    // 启动命令
    if err := cmd.Start(); err != nil {
        return fmt.Errorf("启动命令失败: %v", err)
    }

    // 实时写入日志
    go func() {
        io.Copy(logF, stdoutPipe)
    }()
    go func() {
        io.Copy(logF, stderrPipe)
    }()

    // 等待命令完成
    err = cmd.Wait()
    
    // 写入执行结果
    if err != nil {
        logF.WriteString(fmt.Sprintf("\n\n=== 命令执行失败 ===\n错误: %v\n", err))
    } else {
        logF.WriteString(fmt.Sprintf("\n\n=== 命令执行成功 ===\n退出码: %d\n", cmd.ProcessState.ExitCode()))
    }

    return err
}