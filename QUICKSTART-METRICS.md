# Metrics 快速入门指南

## 5分钟上手

### 1️⃣ 构建工具
```bash
make build-metrics
```

### 2️⃣ 运行任务（生成数据）
```bash
# 启动服务器
./smart-ci-server -config config.yaml

# 手动触发任务（另一个终端）
curl "http://localhost:8080/webhook/bash?task=your-task-name"
```

### 3️⃣ 查看统计
```bash
# 所有任务概览
./smart-ci-metrics all

# 查看特定任务
./smart-ci-metrics latest -task your-task-name
./smart-ci-metrics list -task your-task-name -days 7
./smart-ci-metrics stats -task your-task-name -days 30
```

## 常用命令

| 命令 | 用途 | 示例 |
|------|------|------|
| `all` | 查看所有任务 | `./smart-ci-metrics all` |
| `latest` | 最近一次执行 | `./smart-ci-metrics latest -task backup` |
| `list` | 执行历史 | `./smart-ci-metrics list -task backup -days 7` |
| `stats` | 统计信息 | `./smart-ci-metrics stats -task backup -days 30` |

## 参数说明

- `-task <name>` - 任务名称（必需，除了 all 命令）
- `-logdir <path>` - 日志目录（默认：./logs）
- `-days <N>` - 最近N天
- `-hours <N>` - 最近N小时
- `-limit <N>` - 最多显示N条（默认：20）

## 测试

运行测试脚本验证功能：
```bash
./test-metrics.sh
```

这将：
1. 构建所有组件
2. 启动测试服务器
3. 执行测试任务
4. 展示统计结果

## 数据位置

元数据存储在每个任务目录中：
```
logs/
└── 20251203-075207-328f77c8/
    ├── metadata.json    ← 元数据
    ├── task.log         ← 日志
    └── ai-analysis.md   ← AI分析（可选）
```

## 示例场景

### 监控任务健康
```bash
# 每小时检查任务成功率
0 * * * * /path/to/smart-ci-metrics stats -task critical-task -days 1
```

### 故障排查
```bash
# 查看最近失败的任务
./smart-ci-metrics list -task failing-task -hours 24
./smart-ci-metrics latest -task failing-task
cat logs/<task-id>/task.log
```

### 性能分析
```bash
# 查看任务执行时长趋势
./smart-ci-metrics stats -task slow-task -days 30
```

## 更多文档

- 📖 [完整文档](docs/metrics.md)
- 🔧 [实现细节](METRICS_IMPLEMENTATION.md)
- 📝 [变更日志](CHANGES.md)

## 获取帮助

```bash
./smart-ci-metrics
```

显示所有可用命令和选项。
