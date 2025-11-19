package config

// ================= 配置定义 =================

type Config struct {
	LLMKey   string       `yaml:"llm_key"`  // 大模型 API Key
	LLMBase  string       `yaml:"llm_base"` // 大模型 Base URL
	Schedule string       `yaml:"schedule"` // 全局定时
	Repos    []RepoConfig `yaml:"repos"`
}

type RepoConfig struct {
	Name        string   `yaml:"name"`
	URL         string   `yaml:"url"`
	Branches    []string `yaml:"branches"`
	Dockerfile  string   `yaml:"dockerfile"`
	TestCmd     string   `yaml:"test_cmd"`
	AutoAnalyze bool     `yaml:"auto_analyze"` // 是否开启 AI 自动失败分析
}
