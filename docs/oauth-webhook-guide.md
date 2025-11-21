# OAuth和Webhook使用指南

## OAuth授权

### 支持的平台

- GitHub
- GitLab（待实现）
- Gitea（待实现）

### GitHub OAuth配置

1. 在GitHub上创建OAuth应用：
   - 访问 https://github.com/settings/developers
   - 点击 "New OAuth App"
   - 填写应用信息：
     - Application name: LiteCICD
     - Homepage URL: http://localhost:8080
     - Authorization callback URL: http://localhost:8080/oauth/callback?provider=github
   - 获取 Client ID 和 Client Secret

2. 配置config.yaml：

```yaml
oauth:
  - name: "github"
    client_id: "your_github_client_id"
    client_secret: "your_github_client_secret"
    redirect_url: "http://localhost:8080/oauth/callback?provider=github"
    scopes:
      - "repo"
      - "read:user"
      - "admin:repo_hook"
```

3. 授权流程：
   - 访问 `http://localhost:8080/oauth/authorize?provider=github`
   - 浏览器重定向到GitHub授权页面
   - 用户同意授权后，GitHub重定向回 `/oauth/callback`
   - 系统自动交换访问令牌并获取用户信息

## Webhook监听

### 配置Webhook

```yaml
webhooks:
  - name: "webhook名称"
    path: "/webhook/路径"
    provider: "github"  # 提供商：github, gitlab, gitea
    secret: "webhook密钥"  # 用于验证签名
    events:  # 监听的事件
      - "push"
      - "pull_request"
    filters:  # 过滤条件
      branches:
        - "main"
        - "develop"
      repos:
        - "my-repo"
      actions:
        - "opened"
        - "synchronize"
    actions:  # 触发的动作
      - type: "command"
        command: "echo 'Build triggered'"
        working_dir: "/path/to/project"
        timeout: 300
      - type: "script"
        script: "./scripts/deploy.sh"
        timeout: 600
      - type: "task"
        task: "deploy-app"
```

### 动作类型

#### 1. command - 执行shell命令

```yaml
- type: "command"
  command: "npm test"
  working_dir: "/home/project"
  timeout: 300
  env:
    NODE_ENV: "test"
```

#### 2. script - 执行shell脚本

```yaml
- type: "script"
  script: "./scripts/deploy.sh"
  working_dir: "/home/project"
  timeout: 600
  env:
    DEPLOY_ENV: "production"
```

#### 3. task - 执行已配置的任务

```yaml
- type: "task"
  task: "deploy-app"  # 引用bash_tasks中的任务名称
```

### GitHub Webhook设置

1. 在GitHub仓库设置中添加Webhook：
   - 访问 `https://github.com/用户名/仓库名/settings/hooks`
   - 点击 "Add webhook"
   - Payload URL: `http://your-server:8080/webhook/github/push`
   - Content type: `application/json`
   - Secret: 与config.yaml中的secret一致
   - 选择触发事件（push, pull_request等）

2. 测试Webhook：
   - 推送代码到仓库
   - 查看服务器日志，应看到webhook接收和处理日志

### 过滤条件

#### branches - 分支过滤
只处理指定分支的事件：
```yaml
filters:
  branches:
    - "main"
    - "develop"
```

#### repos - 仓库过滤
只处理指定仓库的事件：
```yaml
filters:
  repos:
    - "my-repo"
    - "another-repo"
```

#### actions - 动作过滤
只处理指定动作的事件（如PR的opened、closed等）：
```yaml
filters:
  actions:
    - "opened"
    - "synchronize"
    - "closed"
```

### 事件类型

#### GitHub支持的事件：
- `push` - 代码推送
- `pull_request` - Pull Request
- `release` - 发布
- `issues` - Issue
- `issue_comment` - Issue评论
- `create` - 创建分支或标签
- `delete` - 删除分支或标签
- `workflow_run` - GitHub Actions工作流运行

### 完整示例

```yaml
# GitHub自动部署
webhooks:
  - name: "auto-deploy"
    path: "/webhook/github/deploy"
    provider: "github"
    secret: "my-secret-key"
    events:
      - "push"
    filters:
      branches:
        - "main"
    actions:
      # 1. 拉取最新代码
      - type: "command"
        command: "git pull origin main"
        working_dir: "/home/project"
        timeout: 60
      # 2. 安装依赖
      - type: "command"
        command: "npm install"
        working_dir: "/home/project"
        timeout: 300
      # 3. 构建
      - type: "command"
        command: "npm run build"
        working_dir: "/home/project"
        timeout: 600
      # 4. 重启服务
      - type: "script"
        script: "./scripts/restart.sh"
        working_dir: "/home/project"
        timeout: 60

  # Pull Request自动测试
  - name: "pr-test"
    path: "/webhook/github/pr"
    provider: "github"
    secret: "my-secret-key"
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

### 环境变量

webhook执行时，以下环境变量会自动设置：

- `WEBHOOK_EVENT` - 事件类型
- `WEBHOOK_REPO` - 仓库名称
- `WEBHOOK_BRANCH` - 分支名称
- `WEBHOOK_COMMIT` - 提交SHA（如果适用）
- `WEBHOOK_SENDER` - 触发者用户名

### 安全建议

1. 始终设置webhook secret并验证签名
2. 使用HTTPS（配置TLS）
3. 限制webhook只能访问必要的资源
4. 设置合理的超时时间
5. 记录所有webhook活动日志
6. 使用环境变量存储敏感信息

