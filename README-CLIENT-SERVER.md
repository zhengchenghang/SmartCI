# SmartCI - Client-Server 架构的CI/CD工具

SmartCI 是一个轻量级的CI/CD引擎，现在支持client-server架构，可以远程管理CI/CD任务。

## 架构概述

SmartCI 现在采用client-server架构：
- **Server**: 负责执行CI/CD任务，调度Bash任务，提供HTTP API
- **Client**: 发送命令到Server，远程管理CI/CD任务

## 快速开始

### 1. 构建项目

```bash
# 构建所有组件
make build

# 或者分别构建
make build-server  # 构建服务器
make build-client  # 构建客户端
```

### 2. 启动服务器

```bash
# 使用默认配置启动
./smart-ci-server -mode server

# 指定配置文件
./smart-ci-server -mode server -config config.yaml

# 覆盖配置文件中的服务器设置
./smart-ci-server -mode server -host 0.0.0.0 -port 9090
```

### 3. 使用客户端

```bash
# 查看帮助
./smart-ci-client -command help

# 查看所有可用任务
./smart-ci-client -command "list"

# 运行一次任务
./smart-ci-client -command "run backup-database"

# 启动周期任务
./smart-ci-client -command "start system-monitor"

# 查看任务状态
./smart-ci-client -command "status"

# 查看特定任务状态
./smart-ci-client -command "status backup-database"

# 查看任务日志
./smart-ci-client -command "logs backup-database 50"

# 查看服务器配置
./smart-ci-client -command "config"

# 检查服务器健康状态
./smart-ci-client -command "health"
```

## 配置文件

配置文件 `config.yaml` 现在包含服务器配置：

```yaml
# 服务器配置
server:
  host: "localhost"
  port: 8080
  auth_token: ""  # 可选：认证密钥
  tls:
    enabled: false
    cert_file: ""
    key_file: ""

# 大模型配置
llm_key: "${OPENAI_API_KEY}"
llm_base: "https://api.openai.com/v1"

# 全局定时调度
schedule: "@every 30m"

# 仓库CI/CD配置
repos:
  - name: "backend-go"
    url: "https://github.com/user/backend"
    branches: ["main", "develop"]
    dockerfile: "Dockerfile"
    test_cmd: "go test ./..."
    auto_analyze: true

# Bash任务调度配置
bash_tasks:
  - name: "backup-database"
    description: "每日凌晨备份数据库"
    schedule: "0 2 * * *"
    command: "pg_dump mydb > backup_$(date +%Y%m%d_%H%M%S).sql"
    working_dir: "/backups"
    timeout: 1800
    auto_analyze: true
```

## 可用命令

### 服务器管理命令

- `server-up [port] [host]` - 启动服务器（可覆盖配置文件中的端口和主机）
- `server-down` - 停止服务器

### 任务管理命令

- `run <task_name>` - 运行一次指定任务
- `start <task_name>` - 启动指定任务（周期调度）
- `stop <task_name>` - 停止指定任务
- `status [task_name]` - 查看任务状态（不指定任务名则显示所有）
- `logs <task_name> [lines]` - 查看任务日志

### 信息查询命令

- `list` - 列出所有可用任务
- `config` - 查看当前配置
- `health` - 检查服务器健康状态
- `reload` - 重新加载配置文件

## API 接口

服务器提供以下HTTP API端点：

- `POST /api/command` - 执行命令
- `GET /health` - 健康检查
- `GET /config` - 获取配置信息
- `GET /mcp/tools` - MCP工具列表（兼容性）
- `POST /mcp/call` - MCP工具调用（兼容性）
- `GET /webhook` - Webhook触发（兼容性）
- `GET /webhook/bash` - Bash任务Webhook触发（兼容性）

### API 请求示例

```bash
# 执行命令
curl -X POST http://localhost:8080/api/command \
  -H "Content-Type: application/json" \
  -d '{"command": "run", "args": {"task_name": "backup-database"}}'

# 健康检查
curl http://localhost:8080/health

# 获取配置
curl http://localhost:8080/config
```

## 认证

如果配置了 `auth_token`，客户端需要在请求头中提供认证：

```bash
# 使用认证令牌
curl -X POST http://localhost:8080/api/command \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-token" \
  -d '{"command": "list"}'
```

## 开发

### 开发模式启动

```bash
# 开发模式启动服务器（默认localhost:8080）
make dev

# 或者手动启动
./smart-ci-server -mode server -config config.yaml -host localhost -port 8080
```

### 运行测试

```bash
make test
```

### 安装到系统

```bash
make install
```

## 迁移说明

从旧版本迁移到新的client-server架构：

1. 现有的 `main.go` 现在是server模式
2. 新增 `client.go` 作为客户端
3. 配置文件新增 `server` 配置段
4. 所有原有功能保持兼容

### 兼容性

- 旧的webhook端点仍然可用
- MCP接口保持兼容
- 配置文件向后兼容（server配置有默认值）

## 故障排除

### 服务器无法启动

1. 检查端口是否被占用
2. 检查配置文件格式是否正确
3. 查看日志输出

### 客户端连接失败

1. 检查服务器是否正在运行
2. 检查网络连接
3. 验证服务器地址和端口配置
4. 如果启用了认证，检查auth_token配置

### 任务执行失败

1. 检查任务配置是否正确
2. 查看日志文件了解详细错误信息
3. 验证脚本文件权限和路径