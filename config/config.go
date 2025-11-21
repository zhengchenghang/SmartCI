package config

// ================= 配置定义 =================

type Config struct {
	Server    ServerConfig      `yaml:"server"`     // 服务器配置
	OAuth     []OAuthConfig     `yaml:"oauth"`      // OAuth配置
	Webhooks  []WebhookConfig   `yaml:"webhooks"`   // Webhook配置
	LLMKey    string            `yaml:"llm_key"`    // 大模型 API Key
	LLMBase   string            `yaml:"llm_base"`   // 大模型 Base URL
	Schedule  string            `yaml:"schedule"`   // 全局定时
	Repos     []RepoConfig      `yaml:"repos"`      // 仓库配置
	BashTasks []BashTaskConfig  `yaml:"bash_tasks"` // Bash任务配置
}

// ServerConfig 服务器配置
type ServerConfig struct {
    Host      string   `yaml:"host"`       // 服务器主机
    Port      int      `yaml:"port"`       // 服务器端口
    AuthToken string   `yaml:"auth_token"` // 认证令牌
    TLS       TLSConfig `yaml:"tls"`       // TLS配置
}

// TLSConfig TLS配置
type TLSConfig struct {
    Enabled  bool   `yaml:"enabled"`  // 是否启用TLS
    CertFile string `yaml:"cert_file"` // 证书文件路径
    KeyFile  string `yaml:"key_file"`  // 私钥文件路径
}

type RepoConfig struct {
    Name        string   `yaml:"name"`
    URL         string   `yaml:"url"`
    Branches    []string `yaml:"branches"`
    Dockerfile  string   `yaml:"dockerfile"`
    TestCmd     string   `yaml:"test_cmd"`
    AutoAnalyze bool     `yaml:"auto_analyze"` // 是否开启 AI 自动失败分析
}

// BashTaskConfig 定义Bash任务配置
type BashTaskConfig struct {
	Name        string `yaml:"name"`         // 任务名称
	Description string `yaml:"description"`  // 任务描述
	Schedule    string `yaml:"schedule"`     // Cron表达式，如 "0 */2 * * *"
	Command     string `yaml:"command"`      // Bash命令（内联）
	ScriptFile  string `yaml:"script_file"`  // Bash脚本文件路径
	WorkingDir  string `yaml:"working_dir"`  // 工作目录，可选
	Timeout     int    `yaml:"timeout"`      // 超时时间（秒），默认300
	AutoAnalyze bool   `yaml:"auto_analyze"` // 是否开启 AI 自动失败分析
}

// OAuthConfig OAuth配置
type OAuthConfig struct {
	Name         string   `yaml:"name"`          // 提供商名称：github, gitlab, gitea等
	ClientID     string   `yaml:"client_id"`     // OAuth客户端ID
	ClientSecret string   `yaml:"client_secret"` // OAuth客户端密钥
	RedirectURL  string   `yaml:"redirect_url"`  // OAuth回调URL
	Scopes       []string `yaml:"scopes"`        // OAuth权限范围
}

// WebhookConfig Webhook配置
type WebhookConfig struct {
	Name      string            `yaml:"name"`      // webhook名称
	Path      string            `yaml:"path"`      // webhook路径，如 /webhook/github
	Provider  string            `yaml:"provider"`  // 提供商：github, gitlab, gitea等
	Secret    string            `yaml:"secret"`    // webhook密钥
	Events    []string          `yaml:"events"`    // 监听的事件类型
	Actions   []WebhookAction   `yaml:"actions"`   // 触发的动作
	Filters   WebhookFilter     `yaml:"filters"`   // 过滤条件
}

// WebhookAction webhook触发的动作
type WebhookAction struct {
	Type       string            `yaml:"type"`        // 动作类型：command, script, task
	Command    string            `yaml:"command"`     // shell命令
	Script     string            `yaml:"script"`      // shell脚本路径
	Task       string            `yaml:"task"`        // 已配置的任务名称
	WorkingDir string            `yaml:"working_dir"` // 工作目录
	Timeout    int               `yaml:"timeout"`     // 超时时间（秒）
	Env        map[string]string `yaml:"env"`         // 环境变量
}

// WebhookFilter webhook过滤条件
type WebhookFilter struct {
	Branches []string `yaml:"branches"` // 分支过滤
	Repos    []string `yaml:"repos"`    // 仓库过滤
	Actions  []string `yaml:"actions"`  // 动作过滤（如：opened, closed等）
}
