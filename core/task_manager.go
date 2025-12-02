package core

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lite-cicd/config"
)

// GenerateTaskID 生成唯一的任务ID
func GenerateTaskID() string {
	timestamp := time.Now().Format("20060102-150405")
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	randomStr := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("%s-%s", timestamp, randomStr)
}

// CreateTaskDir 创建任务目录
func CreateTaskDir(baseDir, taskID string) (string, error) {
	taskDir := filepath.Join(baseDir, taskID)
	err := os.MkdirAll(taskDir, 0755)
	if err != nil {
		return "", fmt.Errorf("创建任务目录失败: %v", err)
	}
	return taskDir, nil
}

// CollectContext 收集任务上下文
// contextConfig: 上下文配置列表（预定义类型或路径通配符）
// taskDir: 任务目录
// logFile: 日志文件路径
func CollectContext(contextConfig []string, taskDir, logFile string) (map[string]string, error) {
	context := make(map[string]string)

	for _, item := range contextConfig {
		switch item {
		case "log":
			// 预定义类型：读取日志文件
			if logFile != "" {
				content, err := ioutil.ReadFile(logFile)
				if err != nil {
					return nil, fmt.Errorf("读取日志文件失败: %v", err)
				}
				context["log"] = string(content)
			}
		default:
			// 路径通配符：收集匹配的文件
			if err := collectPathContext(item, taskDir, context); err != nil {
				return nil, err
			}
		}
	}

	return context, nil
}

// collectPathContext 收集路径匹配的文件作为上下文
func collectPathContext(pattern, taskDir string, context map[string]string) error {
	// 将模式转换为绝对路径
	var searchPath string
	if filepath.IsAbs(pattern) {
		searchPath = pattern
	} else {
		searchPath = filepath.Join(taskDir, pattern)
	}

	// 支持递归通配符 **
	if strings.Contains(pattern, "**") {
		return collectRecursivePattern(pattern, taskDir, context)
	}

	// 使用filepath.Glob进行匹配
	matches, err := filepath.Glob(searchPath)
	if err != nil {
		return fmt.Errorf("路径匹配失败 [%s]: %v", pattern, err)
	}

	// 读取匹配的文件
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue
		}

		content, err := ioutil.ReadFile(match)
		if err != nil {
			continue
		}

		// 使用相对路径作为key
		relPath, _ := filepath.Rel(taskDir, match)
		if relPath == "" {
			relPath = filepath.Base(match)
		}
		context[relPath] = string(content)
	}

	return nil
}

// collectRecursivePattern 收集支持 ** 递归通配符的文件
func collectRecursivePattern(pattern, taskDir string, context map[string]string) error {
	// 解析模式：分离路径前缀和文件模式
	parts := strings.Split(pattern, "**")
	if len(parts) != 2 {
		return fmt.Errorf("不支持的递归模式: %s", pattern)
	}

	prefix := strings.TrimSuffix(parts[0], "/")
	suffix := strings.TrimPrefix(parts[1], "/")

	// 确定搜索根目录
	var searchRoot string
	if filepath.IsAbs(prefix) {
		searchRoot = prefix
	} else if prefix == "" {
		searchRoot = taskDir
	} else {
		searchRoot = filepath.Join(taskDir, prefix)
	}

	// 递归遍历目录
	err := filepath.Walk(searchRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 忽略错误继续遍历
		}

		if info.IsDir() {
			return nil
		}

		// 检查文件是否匹配后缀模式
		relPath, err := filepath.Rel(searchRoot, path)
		if err != nil {
			return nil
		}

		matched, err := filepath.Match(suffix, filepath.Base(path))
		if err != nil {
			return nil
		}

		if matched {
			content, err := ioutil.ReadFile(path)
			if err != nil {
				return nil
			}

			// 使用相对于任务目录的路径作为key
			taskRelPath, _ := filepath.Rel(taskDir, path)
			if taskRelPath == "" {
				taskRelPath = relPath
			}
			context[taskRelPath] = string(content)
		}

		return nil
	})

	return err
}

// InvokeAI 调用AI分析
func InvokeAI(agent Agent, aiConfig config.AIConfig, taskDir string, result *TaskResult) error {
	if !aiConfig.Enabled {
		return nil
	}

	// 收集上下文
	context, err := CollectContext(aiConfig.Context, taskDir, result.LogFile)
	if err != nil {
		return fmt.Errorf("收集上下文失败: %v", err)
	}

	// 使用默认Prompt如果未配置
	prompt := aiConfig.Prompt
	if prompt == "" {
		prompt = "请分析以下内容，找出问题并给出建议："
	}

	// 调用AI分析（实现留空）
	analysis, err := agent.AnalyzeWithContext(prompt, context)
	if err != nil {
		return fmt.Errorf("AI分析失败: %v", err)
	}

	// 确定输出文件路径
	outputFile := aiConfig.OutputFile
	if outputFile == "" {
		outputFile = "ai-analysis.md"
	}
	
	// 如果是相对路径，则相对于任务目录
	if !filepath.IsAbs(outputFile) {
		outputFile = filepath.Join(taskDir, outputFile)
	}

	// 写入分析结果
	err = ioutil.WriteFile(outputFile, []byte(analysis), 0644)
	if err != nil {
		return fmt.Errorf("写入AI分析结果失败: %v", err)
	}

	return nil
}
