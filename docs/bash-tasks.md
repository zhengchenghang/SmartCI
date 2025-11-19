# Bash任务调度功能

SmartCI 现在支持 Bash 任务的定时调度执行，可以在配置文件中配置 bash 代码或者指定 bash 文件，然后按配置的周期和时间自动触发执行。

## 功能特性

- ✅ 支持内联 Bash 命令和外部脚本文件
- ✅ 灵活的 Cron 表达式调度
- ✅ 可配置工作目录和超时时间
- ✅ 实时日志输出和持久化
- ✅ AI 失败分析支持
- ✅ Webhook 和 MCP 接口触发

## 配置说明

### 基本配置结构

```yaml
bash_tasks:
  - name: "任务名称"
    description: "任务描述"
    schedule: "Cron表达式"
    command: "Bash命令"           # 内联命令
    # 或者
    script_file: "脚本文件路径"    # 外部脚本文件
    working_dir: "工作目录"       # 可选
    timeout: 300                 # 超时时间(秒)，默认300
    auto_analyze: true           # 是否开启AI分析
```

### Cron 表达式示例

| 表达式 | 说明 |
|--------|------|
| `0 2 * * *` | 每天凌晨2点 |
| `*/5 * * * *` | 每5分钟 |
| `0 0 * * 0` | 每周日午夜 |
| `0 6 * * 1,3,5` | 周一、三、五早上6点 |
| `0 */4 * * *` | 每4小时 |

## 使用示例

### 1. 数据库备份任务

```yaml
- name: "backup-database"
  description: "每日凌晨备份数据库"
  schedule: "0 2 * * *"
  command: |
    pg_dump mydb > backup_$(date +%Y%m%d_%H%M%S).sql
    gzip backup_*.sql
    find ./backups -name "*.sql.gz" -mtime +7 -delete
  working_dir: "/backups"
  timeout: 1800
  auto_analyze: true
```

### 2. 使用外部脚本文件

```yaml
- name: "deploy-app"
  description: "应用部署脚本"
  schedule: "0 6 * * 1,3,5"
  script_file: "./scripts/deploy.sh"
  working_dir: "/home/engine/project"
  timeout: 600
  auto_analyze: true
```

### 3. 系统监控任务

```yaml
- name: "system-monitor"
  description: "每5分钟检查系统状态"
  schedule: "*/5 * * * *"
  command: |
    df -h | awk '$5+0 > 80 {print "磁盘警告: " $0}'
    free | grep Mem | awk '$3/$2 > 0.8 {print "内存警告: " $0}'
  timeout: 60
  auto_analyze: true
```

## API 触发

### Webhook 触发

```bash
# 触发指定的bash任务
curl "http://localhost:8080/webhook/bash?task=backup-database"
```

### MCP 接口

```json
{
  "tool": "trigger_bash_task",
  "args": {
    "task": "backup-database"
  }
}
```

## 日志和监控

- 所有 bash 任务的执行日志都会保存在 `./logs/` 目录下
- 日志文件命名格式：`bash-{任务名}-{时间戳}.log`
- 如果开启了 `auto_analyze`，失败时会生成 AI 分析报告

## 最佳实践

1. **脚本安全性**
   - 使用 `set -e` 确保脚本遇到错误时退出
   - 验证输入参数和环境变量
   - 避免在脚本中硬编码敏感信息

2. **超时设置**
   - 根据任务执行时间合理设置超时
   - 长时间运行的任务建议设置较长的超时时间

3. **工作目录**
   - 明确指定工作目录，避免路径问题
   - 使用绝对路径确保脚本可移植性

4. **错误处理**
   - 在脚本中添加适当的错误检查
   - 开启 AI 分析帮助诊断失败原因

5. **资源管理**
   - 定期清理旧的日志文件
   - 监控任务执行时间和资源消耗

## 故障排除

### 常见问题

1. **任务不执行**
   - 检查 Cron 表达式是否正确
   - 确认服务正在运行
   - 查看系统日志

2. **脚本执行失败**
   - 检查脚本文件权限
   - 验证工作目录是否存在
   - 查看详细日志输出

3. **超时问题**
   - 增加 timeout 配置值
   - 优化脚本执行效率
   - 检查系统资源状况

### 调试方法

1. 查看实时日志：
   ```bash
   tail -f ./logs/bash-任务名-*.log
   ```

2. 手动触发测试：
   ```bash
   curl "http://localhost:8080/webhook/bash?task=任务名"
   ```

3. 检查任务状态：
   ```bash
   # 查看 MCP 工具列表
   curl http://localhost:8080/mcp/tools
   ```