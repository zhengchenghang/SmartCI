# OAuth和Webhook功能说明

## 概述

本项目实现了OAuth授权和Webhook监听功能，支持与主流代码托管平台集成。

## 功能特性

### 1. OAuth授权框架

- **通用接口设计**：支持扩展多种OAuth提供商
- **GitHub OAuth实现**：完整的GitHub OAuth 2.0授权流程
- **令牌管理**：自动交换和验证访问令牌

### 2. Webhook监听

- **多平台支持**：GitHub、GitLab、Gitea等
- **事件过滤**：支持按分支、仓库、动作过滤
- **签名验证**：自动验证webhook请求签名
- **灵活动作**：支持执行命令、脚本、任务

## 架构设计

```
oauth/
├── provider.go    # OAuth通用接口和基础实现
└── github.go      # GitHub OAuth具体实现

webhook/
└── handler.go     # Webhook处理器

config/
└── config.go      # 配置结构（已扩展支持OAuth和Webhook）

main.go            # 主服务（集成OAuth和Webhook）
```

## 快速开始

### 1. 配置GitHub OAuth

在GitHub创建OAuth应用后，配置`config.yaml`：

```yaml
oauth:
  - name: "github"
    client_id: "your_client_id"
    client_secret: "your_client_secret"
    redirect_url: "http://localhost:8080/oauth/callback?provider=github"
    scopes:
      - "repo"
      - "read:user"
      - "admin:repo_hook"
```

### 2. 配置Webhook

```yaml
webhooks:
  - name: "github-push-deploy"
    path: "/webhook/github/push"
    provider: "github"
    secret: "your_webhook_secret"
    events:
      - "push"
    filters:
      branches:
        - "main"
    actions:
      - type: "task"
        task: "deploy-app"
```

### 3. 启动服务

```bash
./smart-ci-server --config config.yaml
```

### 4. OAuth授权流程

1. 访问授权URL：
   ```
   http://localhost:8080/oauth/authorize?provider=github
   ```

2. 在GitHub上授权应用

3. 授权成功后，系统返回令牌和用户信息

### 5. 设置GitHub Webhook

1. 进入GitHub仓库设置 → Webhooks
2. 添加webhook：
   - Payload URL: `http://your-server:8080/webhook/github/push`
   - Content type: `application/json`
   - Secret: 与config.yaml中一致
   - 选择事件：Push、Pull Request等

## API端点

### OAuth相关

| 端点 | 方法 | 说明 |
|------|------|------|
| `/oauth/authorize` | GET | 发起OAuth授权 |
| `/oauth/callback` | GET | OAuth回调处理 |

#### 参数说明

**GET /oauth/authorize**
- `provider`: OAuth提供商（如：github）
- `state`: 可选，状态参数

**GET /oauth/callback**
- 由OAuth提供商自动调用
- 返回访问令牌和用户信息

### Webhook相关

动态注册的webhook端点根据配置文件中的`path`字段确定。

## 配置详解

### OAuth配置

```yaml
oauth:
  - name: "github"              # 提供商名称
    client_id: "xxx"            # OAuth客户端ID
    client_secret: "xxx"        # OAuth客户端密钥
    redirect_url: "xxx"         # 回调URL
    scopes:                     # 权限范围
      - "repo"
      - "read:user"
```

### Webhook配置

```yaml
webhooks:
  - name: "webhook名称"
    path: "/webhook/路径"        # webhook端点路径
    provider: "github"           # 提供商
    secret: "密钥"               # 签名验证密钥
    events:                      # 监听事件
      - "push"
      - "pull_request"
    filters:                     # 过滤条件
      branches:                  # 分支过滤
        - "main"
      repos:                     # 仓库过滤
        - "my-repo"
      actions:                   # 动作过滤
        - "opened"
    actions:                     # 触发动作
      - type: "command"          # 动作类型
        command: "npm test"      # 执行命令
        working_dir: "/path"     # 工作目录
        timeout: 300             # 超时（秒）
```

### Webhook动作类型

#### 1. command - 执行shell命令

```yaml
- type: "command"
  command: "echo 'Hello'"
  working_dir: "/home/project"
  timeout: 300
  env:
    KEY: "value"
```

#### 2. script - 执行shell脚本

```yaml
- type: "script"
  script: "./scripts/deploy.sh"
  working_dir: "/home/project"
  timeout: 600
```

#### 3. task - 执行已配置任务

```yaml
- type: "task"
  task: "deploy-app"  # 引用bash_tasks中的任务
```

## 使用示例

### 示例1：自动部署

当代码推送到main分支时，自动执行部署：

```yaml
webhooks:
  - name: "auto-deploy"
    path: "/webhook/github/deploy"
    provider: "github"
    secret: "${GITHUB_WEBHOOK_SECRET}"
    events:
      - "push"
    filters:
      branches:
        - "main"
    actions:
      - type: "command"
        command: "git pull origin main"
        working_dir: "/home/project"
      - type: "task"
        task: "deploy-app"
```

### 示例2：PR自动测试

当创建或更新PR时，自动运行测试：

```yaml
webhooks:
  - name: "pr-test"
    path: "/webhook/github/pr"
    provider: "github"
    secret: "${GITHUB_WEBHOOK_SECRET}"
    events:
      - "pull_request"
    filters:
      actions:
        - "opened"
        - "synchronize"
    actions:
      - type: "command"
        command: "npm test"
        working_dir: "/home/project"
        timeout: 600
```

### 示例3：发布自动化

当创建release时，执行发布脚本：

```yaml
webhooks:
  - name: "auto-release"
    path: "/webhook/github/release"
    provider: "github"
    secret: "${GITHUB_WEBHOOK_SECRET}"
    events:
      - "release"
    filters:
      actions:
        - "published"
    actions:
      - type: "script"
        script: "./scripts/release.sh"
        working_dir: "/home/project"
        timeout: 1800
```

## 扩展其他平台

要添加新的OAuth提供商，实现`oauth.Provider`接口：

```go
type Provider interface {
    GetAuthURL(state string) string
    ExchangeToken(ctx context.Context, code string) (*Token, error)
    RefreshToken(ctx context.Context, refreshToken string) (*Token, error)
    GetUserInfo(ctx context.Context, accessToken string) (interface{}, error)
    ValidateWebhook(r *http.Request, secret string) error
}
```

示例：添加GitLab支持

```go
// oauth/gitlab.go
type GitLabProvider struct {
    BaseProvider
}

func NewGitLabProvider(clientID, clientSecret, redirectURL string, scopes []string) *GitLabProvider {
    return &GitLabProvider{
        BaseProvider: BaseProvider{
            Config: Config{
                ClientID:     clientID,
                ClientSecret: clientSecret,
                RedirectURL:  redirectURL,
                Scopes:       scopes,
            },
            AuthURL:     "https://gitlab.com/oauth/authorize",
            TokenURL:    "https://gitlab.com/oauth/token",
            UserInfoURL: "https://gitlab.com/api/v4/user",
        },
    }
}

// 实现其他接口方法...
```

然后在`main.go`的`initOAuthProviders`中添加：

```go
case "gitlab":
    provider = oauth.NewGitLabProvider(
        oauthCfg.ClientID,
        oauthCfg.ClientSecret,
        oauthCfg.RedirectURL,
        oauthCfg.Scopes,
    )
```

## 安全建议

1. **使用环境变量**存储敏感信息（client_secret、webhook secret）
2. **启用TLS**保护通信安全
3. **验证webhook签名**防止伪造请求
4. **限制OAuth scope**只申请必要权限
5. **设置合理超时**避免长时间执行
6. **记录审计日志**追踪所有操作

## 日志查看

服务运行时会输出详细日志：

```
✅ 已初始化OAuth提供商: github
✅ 已注册Webhook: /webhook/github/push -> github-push-deploy
📥 收到webhook: github-push-deploy, 事件: push
⚙️ 执行Webhook动作: task
✅ Bash任务成功，日志: ./logs/deploy-app-xxx.log
```

## 故障排查

### OAuth授权失败

1. 检查client_id和client_secret是否正确
2. 确认redirect_url与OAuth应用配置一致
3. 查看服务器日志获取详细错误信息

### Webhook未触发

1. 检查webhook URL是否可访问
2. 验证webhook secret是否一致
3. 检查事件和过滤条件是否正确
4. 查看GitHub webhook delivery详情

### 动作执行失败

1. 检查命令或脚本路径是否正确
2. 确认工作目录权限
3. 查看日志文件获取错误详情
4. 验证超时设置是否合理

## 更多文档

- [OAuth和Webhook使用指南](docs/oauth-webhook-guide.md)
- [Bash任务配置](docs/bash-tasks.md)

