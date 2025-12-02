# AI集成功能文档

## 概述

SmartCI现在支持强大的AI集成功能，可以在任务执行结束或失败时自动调用AI进行分析。通过灵活的上下文配置和自定义Prompt，您可以让AI深入了解任务执行情况并提供智能建议。

## 特性

### 1. 任务ID管理

每次任务运行都会生成唯一的任务ID，格式为 `YYYYMMDD-HHMMSS-XXXX`，其中：
- 前半部分是时间戳
- 后半部分是随机字符串

所有任务相关的文件都存储在以任务ID命名的目录中：
```
logs/
  └── 20231215-143025-a1b2c3d4/
      ├── task.log              # 任务执行日志
      ├── ai-analysis.md        # AI分析报告
      └── other-files...        # 其他任务文件
```

### 2. AI配置

在任务配置中添加 `ai` 字段：

```yaml
ai:
  enabled: true              # 是否启用AI分析
  context:                   # 上下文配置列表
    - "log"                  # 预定义类型
    - "*.log"                # 路径通配符
    - "reports/**/*.json"    # 递归通配符
  prompt: "自定义提示词"      # AI分析提示词
  output_file: "report.md"   # 输出文件名（可选）
```

### 3. 上下文配置

#### 预定义类型

- `log`: 自动读取任务日志文件内容

#### 路径通配符

支持标准的文件路径通配符：

- `*.log`: 匹配任务目录下所有 `.log` 文件
- `*.{log,txt}`: 匹配任务目录下所有 `.log` 和 `.txt` 文件
- `reports/*.json`: 匹配 `reports` 目录下的所有 `.json` 文件

#### 递归通配符

支持 `**` 递归匹配：

- `**/*.log`: 匹配任务目录下所有子目录中的 `.log` 文件
- `coverage/**/*`: 匹配 `coverage` 目录下的所有文件

#### 绝对路径

也可以使用绝对路径引用任务目录外的文件：

```yaml
context:
  - "log"
  - "/var/log/app/**/*.log"
```

### 4. Prompt配置

您可以为每个任务自定义AI分析的提示词，例如：

```yaml
prompt: "分析这次Go项目的测试结果，找出失败原因并给出修复建议"
```

如果未配置，将使用默认提示词：
```
请分析以下内容，找出问题并给出建议：
```

### 5. 输出文件

AI分析结果默认输出到任务目录下的 `ai-analysis.md` 文件。您可以通过 `output_file` 字段自定义输出文件名：

```yaml
output_file: "test-report.md"
```

## 使用示例

### 示例1：Bash任务AI分析

```yaml
bash_tasks:
  - name: "run-tests"
    description: "运行单元测试"
    command: |
      npm test -- --coverage
      echo "测试完成"
    ai:
      enabled: true
      context:
        - "log"                    # 任务日志
        - "coverage/**/*.json"     # 覆盖率报告
        - "test-results/*.xml"     # 测试结果
      prompt: "分析测试执行结果和代码覆盖率，给出改进建议"
      output_file: "test-analysis.md"
```

### 示例2：Docker CI/CD任务AI分析

```yaml
repos:
  - name: "backend-api"
    url: "https://github.com/user/backend"
    branches: ["main"]
    dockerfile: "Dockerfile"
    test_cmd: "go test -v ./... -coverprofile=coverage.out"
    ai:
      enabled: true
      context:
        - "log"
        - "*.out"               # 覆盖率文件
      prompt: "分析Go项目的构建和测试结果，找出失败原因"
      output_file: "ci-report.md"
```

### 示例3：数据库备份任务分析

```yaml
bash_tasks:
  - name: "backup-db"
    command: |
      pg_dump mydb > backup.sql
      gzip backup.sql
    ai:
      enabled: true
      context:
        - "log"
        - "*.sql.gz"            # 备份文件（用于检查大小）
      prompt: "检查数据库备份是否成功，如果失败分析原因"
```

## 向后兼容

旧的 `auto_analyze: true` 配置仍然支持，会使用默认的AI分析行为（仅在失败时分析日志）。建议迁移到新的 `ai` 配置以获得更多灵活性。

```yaml
# 旧配置（仍然支持）
auto_analyze: true

# 新配置（推荐）
ai:
  enabled: true
  context: ["log"]
  prompt: "分析失败原因"
```

## 工作流程

1. **任务开始**：生成唯一任务ID
2. **创建目录**：在logs下创建以任务ID命名的目录
3. **执行任务**：执行任务并将日志写入任务目录
4. **任务结束**：
   - 检查AI配置是否启用
   - 收集配置的上下文内容
   - 调用AI分析（实现待完成）
   - 将分析结果写入输出文件

## 注意事项

1. **上下文大小**：注意上下文文件的总大小，避免超出AI模型的token限制
2. **路径安全**：使用绝对路径时注意权限和安全性
3. **异步执行**：AI分析不会阻塞任务的完成状态
4. **错误处理**：如果AI分析失败，不会影响任务本身的状态

## 后续开发

当前AI调用的实现是占位的，具体实现需要：

1. 在 `ai/agent.go` 中完善 `AnalyzeWithContext` 方法
2. 将收集的上下文和prompt组织成合适的格式
3. 调用OpenAI API进行分析
4. 处理API响应和错误

示例实现框架：

```go
func (a *AIAgent) AnalyzeWithContext(prompt string, context map[string]string) (string, error) {
    // 1. 组织上下文为消息
    var contextStr strings.Builder
    for key, value := range context {
        contextStr.WriteString(fmt.Sprintf("### %s\n\n```\n%s\n```\n\n", key, value))
    }
    
    // 2. 调用OpenAI API
    resp, err := a.client.CreateChatCompletion(
        context.Background(),
        openai.ChatCompletionRequest{
            Model: openai.GPT4,
            Messages: []openai.ChatCompletionMessage{
                {
                    Role:    openai.ChatMessageRoleSystem,
                    Content: "你是一个DevOps专家，擅长分析CI/CD任务执行情况。",
                },
                {
                    Role:    openai.ChatMessageRoleUser,
                    Content: prompt + "\n\n" + contextStr.String(),
                },
            },
        },
    )
    
    // 3. 返回分析结果
    if err != nil {
        return "", err
    }
    return resp.Choices[0].Message.Content, nil
}
```

## API支持

任务执行结果现在包含任务ID信息，可以通过API查询：

```bash
# 触发任务
curl -X POST http://localhost:8080/webhook/bash?task=my-task

# 查看任务目录
ls -la logs/20231215-143025-a1b2c3d4/

# 查看AI分析结果
cat logs/20231215-143025-a1b2c3d4/ai-analysis.md
```
