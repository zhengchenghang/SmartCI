# Bash任务调度功能实现总结

## 功能概述

成功为SmartCI系统添加了完整的Bash任务定时调度执行功能，支持在配置文件中配置bash代码或指定bash文件，实现定时、周期性触发执行。

## 实现的功能

### ✅ 核心功能
- **内联Bash命令支持**: 可以直接在配置文件中编写bash命令
- **外部脚本文件支持**: 支持指定bash脚本文件路径
- **灵活的Cron调度**: 支持标准cron表达式定义执行时间
- **工作目录配置**: 可为每个任务指定独立的工作目录
- **超时控制**: 可配置任务执行超时时间，防止无限等待
- **AI失败分析**: 集成现有的AI分析功能，自动分析任务失败原因

### ✅ 集成功能
- **统一日志系统**: 所有bash任务日志统一存储在`./logs/`目录
- **Webhook触发**: 支持通过HTTP API手动触发bash任务
- **MCP接口集成**: 通过MCP协议暴露bash任务触发能力
- **配置文件支持**: 支持YAML配置文件，便于管理大量任务

## 技术实现

### 1. 配置扩展 (`config/config.go`)
```go
type BashTaskConfig struct {
    Name        string `yaml:"name"`
    Description string `yaml:"description"`
    Schedule    string `yaml:"schedule"`
    Command     string `yaml:"command"`
    ScriptFile  string `yaml:"script_file"`
    WorkingDir  string `yaml:"working_dir"`
    Timeout     int    `yaml:"timeout"`
    AutoAnalyze bool   `yaml:"auto_analyze"`
}
```

### 2. Bash执行器 (`executor/bash_executor.go`)
- 实现了`core.BashExecutor`接口
- 支持内联命令和脚本文件两种执行方式
- 提供超时控制和工作目录设置
- 实时捕获输出并写入日志文件

### 3. 接口扩展 (`core/interface.go`)
```go
type BashExecutor interface {
    RunBashTask(ctx context.Context, task config.BashTaskConfig) (string, error)
}
```

### 4. 引擎集成 (`main.go`)
- 扩展`Engine`结构体，添加`bashExecutor`字段
- 实现`TriggerBashTask`方法处理bash任务执行
- 集成cron调度，为每个bash任务创建独立的调度器
- 添加webhook端点`/webhook/bash`支持手动触发

### 5. 配置加载器 (`config/loader.go`)
- 支持从YAML文件加载配置
- 环境变量覆盖机制
- 默认值处理

## 使用示例

### 配置文件示例 (`config.yaml`)
```yaml
bash_tasks:
  - name: "backup-database"
    description: "每日凌晨备份数据库"
    schedule: "0 2 * * *"
    command: |
      pg_dump mydb > backup_$(date +%Y%m%d_%H%M%S).sql
      gzip backup_*.sql
    working_dir: "/backups"
    timeout: 1800
    auto_analyze: true

  - name: "deploy-app"
    description: "应用部署脚本"
    schedule: "0 6 * * 1,3,5"
    script_file: "./scripts/deploy.sh"
    timeout: 600
    auto_analyze: true
```

### API触发示例
```bash
# 手动触发bash任务
curl "http://localhost:8080/webhook/bash?task=backup-database"

# 查看MCP工具列表
curl http://localhost:8080/mcp/tools

# 查看配置摘要
curl http://localhost:8080/config
```

## 测试验证

### 单元测试 (`executor/bash_executor_test.go`)
- ✅ 内联命令执行测试
- ✅ 脚本文件执行测试  
- ✅ 工作目录设置测试
- ✅ 超时控制测试

所有测试通过，功能验证完整。

## 文件结构

```
/home/engine/project/
├── config/
│   ├── config.go          # 配置结构定义
│   └── loader.go          # 配置文件加载器
├── core/
│   └── interface.go       # 接口定义
├── executor/
│   ├── bash_executor.go   # Bash任务执行器
│   └── bash_executor_test.go # 单元测试
├── scripts/
│   └── deploy.sh          # 示例部署脚本
├── docs/
│   └── bash-tasks.md      # 详细文档
├── config.yaml            # 示例配置文件
├── demo.sh                # 演示脚本
└── main.go               # 主程序入口
```

## 关键特性

### 🔧 灵活性
- 支持内联命令和外部脚本两种方式
- 每个任务独立配置工作目录和超时时间
- 支持任意复杂的bash脚本逻辑

### ⏰ 调度能力
- 标准cron表达式支持
- 每个任务独立的调度周期
- 支持手动即时触发

### 📊 监控集成
- 统一的日志收集和存储
- AI驱动的失败分析
- 实时执行状态监控

### 🌐 API集成
- RESTful API接口
- MCP协议支持
- Webhook触发机制

## 最佳实践建议

1. **脚本安全**: 使用`set -e`确保错误时退出
2. **资源管理**: 合理设置超时时间，避免资源泄漏
3. **日志规范**: 在脚本中添加适当的日志输出
4. **错误处理**: 开启AI分析帮助诊断问题
5. **路径管理**: 使用绝对路径确保可移植性

这个实现完全满足了用户需求，提供了强大而灵活的bash任务调度功能，同时保持了与现有系统的良好集成。