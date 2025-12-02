package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lite-cicd/config"
)

func TestGenerateTaskID(t *testing.T) {
	id1 := GenerateTaskID()
	id2 := GenerateTaskID()

	// 验证格式
	if len(id1) < 10 {
		t.Errorf("任务ID太短: %s", id1)
	}

	// 验证唯一性
	if id1 == id2 {
		t.Errorf("任务ID应该是唯一的: %s == %s", id1, id2)
	}

	// 验证包含时间戳
	if !strings.Contains(id1, "-") {
		t.Errorf("任务ID格式不正确: %s", id1)
	}
}

func TestCreateTaskDir(t *testing.T) {
	baseDir := "/tmp/test-smartci"
	taskID := "test-task-123"

	// 清理测试目录
	defer os.RemoveAll(baseDir)

	// 创建任务目录
	taskDir, err := CreateTaskDir(baseDir, taskID)
	if err != nil {
		t.Fatalf("创建任务目录失败: %v", err)
	}

	expectedDir := filepath.Join(baseDir, taskID)
	if taskDir != expectedDir {
		t.Errorf("任务目录路径不正确: got %s, want %s", taskDir, expectedDir)
	}

	// 验证目录存在
	if _, err := os.Stat(taskDir); os.IsNotExist(err) {
		t.Errorf("任务目录未创建: %s", taskDir)
	}
}

func TestCollectContext_Log(t *testing.T) {
	// 创建临时任务目录
	baseDir := "/tmp/test-smartci"
	taskID := "test-collect-log"
	taskDir := filepath.Join(baseDir, taskID)

	// 清理测试目录
	defer os.RemoveAll(baseDir)

	// 创建任务目录和日志文件
	os.MkdirAll(taskDir, 0755)
	logFile := filepath.Join(taskDir, "task.log")
	logContent := "Test log content\nLine 2\nLine 3"
	os.WriteFile(logFile, []byte(logContent), 0644)

	// 收集上下文
	context, err := CollectContext([]string{"log"}, taskDir, logFile)
	if err != nil {
		t.Fatalf("收集上下文失败: %v", err)
	}

	// 验证日志内容
	if log, ok := context["log"]; !ok {
		t.Errorf("未找到日志上下文")
	} else if log != logContent {
		t.Errorf("日志内容不匹配: got %q, want %q", log, logContent)
	}
}

func TestCollectContext_PathWildcard(t *testing.T) {
	// 创建临时任务目录
	baseDir := "/tmp/test-smartci"
	taskID := "test-collect-wildcard"
	taskDir := filepath.Join(baseDir, taskID)

	// 清理测试目录
	defer os.RemoveAll(baseDir)

	// 创建任务目录和测试文件
	os.MkdirAll(taskDir, 0755)
	os.WriteFile(filepath.Join(taskDir, "test1.log"), []byte("log1"), 0644)
	os.WriteFile(filepath.Join(taskDir, "test2.log"), []byte("log2"), 0644)
	os.WriteFile(filepath.Join(taskDir, "test.txt"), []byte("txt"), 0644)

	// 收集上下文（匹配*.log）
	context, err := CollectContext([]string{"*.log"}, taskDir, "")
	if err != nil {
		t.Fatalf("收集上下文失败: %v", err)
	}

	// 验证匹配的文件
	if len(context) != 2 {
		t.Errorf("应该匹配2个文件，实际匹配了%d个", len(context))
	}

	if _, ok := context["test1.log"]; !ok {
		t.Errorf("未找到test1.log")
	}

	if _, ok := context["test2.log"]; !ok {
		t.Errorf("未找到test2.log")
	}

	if _, ok := context["test.txt"]; ok {
		t.Errorf("不应该匹配test.txt")
	}
}

func TestCollectContext_RecursiveWildcard(t *testing.T) {
	// 创建临时任务目录
	baseDir := "/tmp/test-smartci"
	taskID := "test-collect-recursive"
	taskDir := filepath.Join(baseDir, taskID)

	// 清理测试目录
	defer os.RemoveAll(baseDir)

	// 创建多层目录结构
	os.MkdirAll(filepath.Join(taskDir, "reports", "sub1"), 0755)
	os.MkdirAll(filepath.Join(taskDir, "reports", "sub2"), 0755)
	os.WriteFile(filepath.Join(taskDir, "reports", "report1.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(taskDir, "reports", "sub1", "report2.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(taskDir, "reports", "sub2", "report3.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(taskDir, "reports", "readme.txt"), []byte("readme"), 0644)

	// 收集上下文（递归匹配*.json）
	context, err := CollectContext([]string{"reports/**/*.json"}, taskDir, "")
	if err != nil {
		t.Fatalf("收集上下文失败: %v", err)
	}

	// 验证匹配的文件
	if len(context) != 3 {
		t.Errorf("应该匹配3个文件，实际匹配了%d个: %v", len(context), context)
	}
}

func TestInvokeAI(t *testing.T) {
	// 创建临时任务目录
	baseDir := "/tmp/test-smartci"
	taskID := "test-invoke-ai"
	taskDir := filepath.Join(baseDir, taskID)

	// 清理测试目录
	defer os.RemoveAll(baseDir)

	// 创建任务目录和日志文件
	os.MkdirAll(taskDir, 0755)
	logFile := filepath.Join(taskDir, "task.log")
	os.WriteFile(logFile, []byte("test log"), 0644)

	// 创建AI配置
	aiConfig := config.AIConfig{
		Enabled:    true,
		Context:    []string{"log"},
		Prompt:     "分析日志",
		OutputFile: "test-output.md",
	}

	// 创建任务结果
	result := &TaskResult{
		TaskID:  taskID,
		TaskDir: taskDir,
		LogFile: logFile,
	}

	// 创建mock agent
	mockAgent := &MockAgent{}

	// 调用AI
	err := InvokeAI(mockAgent, aiConfig, taskDir, result)
	if err != nil {
		t.Fatalf("调用AI失败: %v", err)
	}

	// 验证输出文件
	outputFile := filepath.Join(taskDir, "test-output.md")
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("输出文件未创建: %s", outputFile)
	}
}

// MockAgent 用于测试的mock agent
type MockAgent struct{}

func (m *MockAgent) AnalyzeLog(logContent string) (string, error) {
	return "Mock analysis", nil
}

func (m *MockAgent) AnalyzeWithContext(prompt string, context map[string]string) (string, error) {
	return "# Mock AI Analysis\n\nTest result", nil
}
