# Metrics 数据收集和统计模块实现文档

## 概述

本文档详细说明了为 SmartCI 项目新增的 metrics（任务统计）功能的实现。

## 实现目标

✅ 对每个定时、周期任务收集执行数据
✅ 记录开始时间、结束时间、执行结果等信息
✅ 将数据以结构化格式存储在任务目录下
✅ 提供命令行工具进行数据统计和展示
✅ 支持查询最近一次执行、历史记录、统计信息等

## 架构设计

### 1. 数据收集层 (Executor)

修改了 `BashExecutor` 和 `DockerExecutor`，在任务执行时自动收集元数据：

- 任务开始时创建 `TaskMetadata` 结构
- 记录开始时间、任务配置等信息
- 任务结束时更新结束时间、执行时长、状态和错误信息
- 保存元数据到任务目录的 `metadata.json` 文件

### 2. 数据存储层 (Metrics Package)

创建了 `metrics` 包，提供：

**核心结构：**
- `TaskMetadata`: 任务执行元数据结构
- `TaskStatistics`: 任务统计信息结构

**存储功能：**
- `SaveMetadata()`: 保存元数据到 JSON 文件
- `LoadMetadata()`: 从 JSON 文件加载元数据

**查询功能：**
- `ListAllMetadata()`: 列出所有任务的元数据
- `GetLatestExecution()`: 获取最近一次执行
- `ListExecutions()`: 列出指定时间范围的执行记录
- `GetStatistics()`: 计算统计信息

**过滤功能：**
- `FilterMetadataByTaskName()`: 按任务名称过滤
- `FilterMetadataByTimeRange()`: 按时间范围过滤

### 3. 展示层 (Display)

创建了 `metrics/display.go`，提供格式化输出：

- `DisplayLatestExecution()`: 显示最近执行详情
- `DisplayExecutionList()`: 显示执行历史列表
- `DisplayStatistics()`: 显示统计报告
- `DisplayAllTasksSummary()`: 显示所有任务概览

### 4. 命令行工具 (CLI)

创建了 `cmd/metrics/main.go`，提供独立的命令行工具：

**支持的子命令：**
- `latest`: 查看最近一次执行
- `list`: 列出历史执行记录
- `stats`: 显示统计信息
- `all`: 显示所有任务概览

## 文件结构

```
lite-cicd/
├── metrics/
│   ├── metadata.go          # 核心数据结构和存储逻辑
│   └── display.go           # 格式化输出功能
├── cmd/
│   └── metrics/
│       └── main.go          # 命令行工具入口
├── executor/
│   ├── bash_executor.go     # 修改：添加元数据收集
│   └── docker_executor.go   # 修改：添加元数据收集
├── docs/
│   └── metrics.md           # 用户使用文档
├── demo-metrics.sh          # 功能演示脚本
├── test-metrics.sh          # 测试脚本
└── Makefile                 # 修改：添加 build-metrics 目标
```

## 数据模型

### TaskMetadata 结构

```go
type TaskMetadata struct {
    TaskID     string                 // 唯一任务ID
    TaskName   string                 // 任务名称
    TaskType   string                 // 任务类型 (bash/repo)
    StartTime  time.Time              // 开始时间
    EndTime    time.Time              // 结束时间
    Duration   float64                // 执行时长（秒）
    Status     string                 // 执行状态 (success/failure)
    Error      string                 // 错误信息
    LogFile    string                 // 日志文件路径
    TaskDir    string                 // 任务目录路径
    Config     map[string]interface{} // 任务配置
}
```

### 存储格式

元数据以 JSON 格式存储在任务目录中：

```
logs/
├── 20251203-075158-0002020f/
│   ├── metadata.json        # 元数据文件
│   ├── task.log             # 执行日志
│   └── ai-analysis.md       # AI分析（可选）
└── ...
```

## 关键实现细节

### 1. Executor 集成

在 `BashExecutor.RunBashTask()` 中：

```go
// 创建元数据记录
metadata := &metrics.TaskMetadata{
    TaskID:    taskID,
    TaskName:  task.Name,
    TaskType:  "bash",
    StartTime: time.Now(),
    // ...
}

// 执行任务...

// 更新元数据
metadata.EndTime = time.Now()
metadata.Duration = metadata.EndTime.Sub(metadata.StartTime).Seconds()
metadata.Status = "success" // or "failure"
metrics.SaveMetadata(metadata)
```

### 2. 统计计算

统计信息通过遍历元数据列表计算：

```go
func GetStatistics(logDir, taskName string, hours, days int) (*TaskStatistics, error) {
    executions, _ := ListExecutions(logDir, taskName, hours, days)
    
    stats := &TaskStatistics{
        TaskName:   taskName,
        TotalCount: len(executions),
    }
    
    for _, exec := range executions {
        if exec.Status == "success" {
            stats.SuccessCount++
        } else {
            stats.FailureCount++
        }
        // 计算平均时长、最短、最长等...
    }
    
    stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalCount) * 100
    return stats, nil
}
```

### 3. 时间范围过滤

支持灵活的时间范围查询：

```go
// 支持按小时和天数组合查询
duration := time.Duration(days*24+hours) * time.Hour
start := time.Now().Add(-duration)
filtered := FilterMetadataByTimeRange(metadata, start, time.Now())
```

## 使用示例

### 构建

```bash
make build-metrics
# 或
go build -o smart-ci-metrics ./cmd/metrics/main.go
```

### 查看最近执行

```bash
./smart-ci-metrics latest -task backup-database
```

### 查看历史记录

```bash
# 最近7天
./smart-ci-metrics list -task backup-database -days 7

# 最近24小时
./smart-ci-metrics list -task backup-database -hours 24
```

### 查看统计信息

```bash
./smart-ci-metrics stats -task backup-database -days 30
```

### 查看所有任务

```bash
./smart-ci-metrics all
```

## 测试

### 自动化测试

运行提供的测试脚本：

```bash
./test-metrics.sh
```

这将：
1. 构建所有组件
2. 启动测试服务器
3. 执行成功和失败任务
4. 生成元数据
5. 展示统计信息

### 演示脚本

```bash
./demo-metrics.sh
```

显示现有任务的统计信息。

## 性能考虑

1. **扫描优化**: 只读取包含 `metadata.json` 的目录
2. **排序优化**: 按修改时间排序，最新记录优先
3. **分页支持**: `list` 命令支持 `-limit` 参数限制返回数量
4. **内存效率**: 流式处理，避免一次性加载所有数据

## 扩展性设计

### 1. 易于添加新字段

元数据使用 JSON 格式，容易扩展：

```go
Config: map[string]interface{}{
    "command":     task.Command,
    "working_dir": task.WorkingDir,
    // 可轻松添加新字段
    "custom_field": value,
}
```

### 2. 支持自定义过滤器

可以轻松添加新的过滤条件：

```go
func FilterByStatus(list []*TaskMetadata, status string) []*TaskMetadata {
    // 实现...
}
```

### 3. 多种输出格式

当前支持终端格式化输出，未来可以添加：
- JSON 输出（用于 API）
- CSV 输出（用于 Excel）
- HTML 输出（用于报告）

## 未来改进

1. **实时监控**
   - WebSocket 推送实时执行数据
   - 图形化仪表板

2. **高级分析**
   - 趋势分析和预测
   - 异常检测
   - 性能基准

3. **数据导出**
   - 支持 CSV/Excel 导出
   - 生成 PDF 报告

4. **告警集成**
   - 失败率阈值告警
   - 执行时长异常告警
   - 集成钉钉、企业微信等

5. **数据库存储**
   - 支持 SQLite/PostgreSQL 存储
   - 提高大数据量查询性能

6. **API 接口**
   - RESTful API
   - GraphQL 支持

## 兼容性

- ✅ 向后兼容：老版本任务目录无 metadata.json 不影响使用
- ✅ 独立工具：metrics 工具独立运行，不依赖服务器
- ✅ 配置无关：无需修改现有配置文件

## 总结

本次实现完成了一个完整的任务数据收集和统计系统，具有：

- 🎯 **自动化**: 无需手动配置，自动收集数据
- 📊 **全面性**: 涵盖所有重要指标
- 🔍 **灵活性**: 支持多种查询方式
- 💡 **易用性**: 简洁的命令行界面
- 🚀 **可扩展**: 架构设计便于未来扩展

该模块为 SmartCI 提供了强大的任务监控和分析能力，帮助用户更好地了解任务执行情况，及时发现和解决问题。
