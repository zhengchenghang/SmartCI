package config

import (
	"os"
	"gopkg.in/yaml.v3"
)

// LoadConfig 从YAML文件加载配置
func LoadConfig(filename string) (Config, error) {
	var cfg Config
	
	// 设置默认值
	cfg = Config{
		Schedule: "@every 1h",
		LLMBase:  "https://api.openai.com/v1",
	}
	
	// 如果文件存在，则加载
	if _, err := os.Stat(filename); err == nil {
		data, err := os.ReadFile(filename)
		if err != nil {
			return cfg, err
		}
		
		err = yaml.Unmarshal(data, &cfg)
		if err != nil {
			return cfg, err
		}
	}
	
	// 从环境变量覆盖配置
	if llmKey := os.Getenv("OPENAI_API_KEY"); llmKey != "" {
		cfg.LLMKey = llmKey
	}
	if llmBase := os.Getenv("LLM_BASE_URL"); llmBase != "" {
		cfg.LLMBase = llmBase
	}
	
	return cfg, nil
}

// SaveConfig 保存配置到YAML文件
func SaveConfig(cfg Config, filename string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	
	return os.WriteFile(filename, data, 0644)
}