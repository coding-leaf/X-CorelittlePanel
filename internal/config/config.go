package config

import (
	"encoding/json"
	"os"

	"xray-panel/internal/types"
)

// Load 读取 config.json 并返回结构化的 Config 实例
func Load() types.Config {
	cfg := types.Config{
		ListenAddr:      "127.0.0.1:8880",
		XrayAPI:         "127.0.0.1:10085",
		Password:        "",
		PublicDashboard: false,
	}

	data, err := os.ReadFile("config.json")
	if err == nil {
		json.Unmarshal(data, &cfg)
	}
	return cfg
}
