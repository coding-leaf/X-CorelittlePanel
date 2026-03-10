package main

import (
	"encoding/json"
	"os"
)

// Config 保存面板本身的运行参数配置
type Config struct {
	ListenAddr       string `json:"listen_addr"`
	XrayAPI          string `json:"xray_api"`
	Password         string `json:"password"`
	AccessLog        string `json:"access_log"`
	ErrorLog         string `json:"error_log"`
	XrayConfigPath   string `json:"xray_config_path"`
	XrayBinPath      string `json:"xray_bin_path"`
	CertPath         string `json:"cert_path"`
	TrafficDataPath  string `json:"traffic_data_path"`
	RealityAddr      string `json:"reality_addr"`
	RealityPublicKey string `json:"reality_public_key"`
	CDNDomain        string `json:"cdn_domain"`
	CDNEncryption    string `json:"cdn_encryption"`
	TelegramToken    string `json:"telegram_token"`
	TelegramChatID   string `json:"telegram_chat_id"`
	PublicDashboard  bool   `json:"public_dashboard"`
	TLSCertFile      string `json:"tls_cert_file"`
	TLSKeyFile       string `json:"tls_key_file"`
	Domain           string `json:"domain"`
	NodePort         string `json:"node_port"`
	NodeFP           string `json:"node_fp"`
	NodeALPN         string `json:"node_alpn"`
}

// UserTraffic 表示单个用户的流量统计数据流
type UserTraffic struct {
	Email    string `json:"email"`
	Uplink   int64  `json:"uplink"`
	Downlink int64  `json:"downlink"`
	Total    int64  `json:"total"`
}

// InboundTraffic 表示入站接口的流量统计
type InboundTraffic struct {
	Tag      string `json:"tag"`
	Uplink   int64  `json:"uplink"`
	Downlink int64  `json:"downlink"`
	Total    int64  `json:"total"`
}

// OutboundTraffic 表示出站接口的流量统计
type OutboundTraffic struct {
	Tag      string `json:"tag"`
	Uplink   int64  `json:"uplink"`
	Downlink int64  `json:"downlink"`
	Total    int64  `json:"total"`
}

// SysStats 表示系统运行时的基本性能指标
type SysStats struct {
	Uptime       uint32 `json:"uptime"`
	NumGoroutine uint32 `json:"num_goroutine"`
	Alloc        uint64 `json:"alloc"`
	TotalAlloc   uint64 `json:"total_alloc"`
	Sys          uint64 `json:"sys"`
	LiveObjects  uint64 `json:"live_objects"`
}

// TrafficHistory 表示某一时间点的流量历史快照
type TrafficHistory struct {
	Time     string `json:"time"`
	Uplink   int64  `json:"uplink"`
	Downlink int64  `json:"downlink"`
}

// DashboardData 是发送给前端面板的综合数据汇集
type DashboardData struct {
	Users     []UserTraffic     `json:"users"`
	Inbounds  []InboundTraffic  `json:"inbounds"`
	Outbounds []OutboundTraffic `json:"outbounds"`
	SysStats  *SysStats         `json:"sys_stats"`
	History   []TrafficHistory  `json:"history"`
	UpdatedAt string            `json:"updated_at"`
}

// AccessLogEntry 表示一条访问日志的解析结果
type AccessLogEntry struct {
	Time   string `json:"time"`
	FromIP string `json:"from_ip"`
	Target string `json:"target"`
	Route  string `json:"route"`
	Email  string `json:"email"`
}

// ErrorLogEntry 表示一条错误日志的解析结果
type ErrorLogEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Module  string `json:"module"`
	Message string `json:"message"`
	Domain  string `json:"domain"`
	Error   string `json:"error"`
}

// loadConfig 读取本地 config.json 并返回结构化的 Config 实例
func loadConfig() Config {
	cfg := Config{
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
