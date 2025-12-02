package executor

import (
    "context"
    "lite-cicd/config"
    "os"
    "path/filepath"
    "testing"
    "time"
)

func TestBashExecutor(t *testing.T) {
    // 创建临时日志目录
    tempDir := t.TempDir()
    
    // 创建bash执行器
    executor, err := NewBashExecutor(tempDir)
    if err != nil {
        t.Fatalf("创建bash执行器失败: %v", err)
    }

    // 测试内联命令
    t.Run("内联命令", func(t *testing.T) {
        task := config.BashTaskConfig{
            Name:        "test-echo",
            Description: "测试echo命令",
            Command:     "echo 'Hello, World!' && echo '测试成功'",
            Timeout:     10,
        }

        result, err := executor.RunBashTask(context.Background(), task)
        if err != nil {
            t.Fatalf("执行bash任务失败: %v", err)
        }

        // 检查结果
        if result == nil {
            t.Fatalf("任务结果为空")
        }

        if result.TaskID == "" {
            t.Fatalf("任务ID为空")
        }

        // 检查日志文件是否存在
        if _, err := os.Stat(result.LogFile); os.IsNotExist(err) {
            t.Fatalf("日志文件不存在: %s", result.LogFile)
        }

        // 读取日志内容
        content, err := os.ReadFile(result.LogFile)
        if err != nil {
            t.Fatalf("读取日志文件失败: %v", err)
        }

        logContent := string(content)
        if !contains(logContent, "Hello, World!") || !contains(logContent, "测试成功") {
            t.Fatalf("日志内容不符合预期: %s", logContent)
        }
    })

    // 测试脚本文件
    t.Run("脚本文件", func(t *testing.T) {
        // 创建临时脚本文件
        scriptFile := filepath.Join(tempDir, "test.sh")
        scriptContent := `#!/bin/bash
echo "脚本开始执行"
date
echo "脚本执行完成"
`
        err := os.WriteFile(scriptFile, []byte(scriptContent), 0755)
        if err != nil {
            t.Fatalf("创建脚本文件失败: %v", err)
        }

        task := config.BashTaskConfig{
            Name:        "test-script",
            Description: "测试脚本文件",
            ScriptFile:  scriptFile,
            Timeout:     10,
        }

        result, err := executor.RunBashTask(context.Background(), task)
        if err != nil {
            t.Fatalf("执行bash任务失败: %v", err)
        }

        // 检查结果
        if result == nil {
            t.Fatalf("任务结果为空")
        }

        // 检查日志文件是否存在
        if _, err := os.Stat(result.LogFile); os.IsNotExist(err) {
            t.Fatalf("日志文件不存在: %s", result.LogFile)
        }

        // 读取日志内容
        content, err := os.ReadFile(result.LogFile)
        if err != nil {
            t.Fatalf("读取日志文件失败: %v", err)
        }

        logContent := string(content)
        if !contains(logContent, "脚本开始执行") || !contains(logContent, "脚本执行完成") {
            t.Fatalf("日志内容不符合预期: %s", logContent)
        }
    })

    // 测试工作目录
    t.Run("工作目录", func(t *testing.T) {
        workDir := tempDir
        task := config.BashTaskConfig{
            Name:        "test-workdir",
            Description: "测试工作目录",
            Command:     "pwd && ls -la",
            WorkingDir:  workDir,
            Timeout:     10,
        }

        result, err := executor.RunBashTask(context.Background(), task)
        if err != nil {
            t.Fatalf("执行bash任务失败: %v", err)
        }

        // 读取日志内容
        content, err := os.ReadFile(result.LogFile)
        if err != nil {
            t.Fatalf("读取日志文件失败: %v", err)
        }

        logContent := string(content)
        if !contains(logContent, workDir) {
            t.Fatalf("工作目录设置失败，预期包含: %s, 实际: %s", workDir, logContent)
        }
    })

    // 测试超时
    t.Run("超时测试", func(t *testing.T) {
        task := config.BashTaskConfig{
            Name:        "test-timeout",
            Description: "测试超时",
            Command:     "sleep 5",
            Timeout:     2, // 2秒超时
        }

        start := time.Now()
        _, err := executor.RunBashTask(context.Background(), task)
        duration := time.Since(start)

        if err == nil {
            t.Fatalf("预期超时错误，但执行成功")
        }

        if duration > 4*time.Second {
            t.Fatalf("超时时间不符合预期，实际耗时: %v", duration)
        }
    })
}

func contains(s, substr string) bool {
    for i := 0; i <= len(s)-len(substr); i++ {
        if s[i:i+len(substr)] == substr {
            return true
        }
    }
    return false
}