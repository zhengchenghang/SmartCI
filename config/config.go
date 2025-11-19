package config

// ================= 配置定义 =================

type Config struct {
    LLMKey    string         `yaml:"llm_key"`    // 大模型 API Key
    LLMBase   string         `yaml:"llm_base"`   // 大模型 Base URL
    Schedule  string         `yaml:"schedule"`   // 全局定时
    Repos     []RepoConfig   `yaml:"repos"`      // 仓库配置
    BashTasks []BashTaskConfig `yaml:"bash_tasks"` // Bash任务配置
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
