package types

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

// UserTraffic 表示单个用户的流量统计数据
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

// SysStats 表示 Xray 进程的系统性能指标
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
	Users     []UserTraffic    `json:"users"`
	Inbounds  []InboundTraffic `json:"inbounds"`
	Outbounds []OutboundTraffic `json:"outbounds"`
	SysStats  *SysStats        `json:"sys_stats"`
	History   []TrafficHistory `json:"history"`
	UpdatedAt string           `json:"updated_at"`
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

// XrayClient 表示 Xray VLESS 入站中的单个用户信息
type XrayClient struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// CertInfo 包含从 X.509 证书中提取的关键信息
type CertInfo struct {
	Subject    string   `json:"subject"`
	Issuer     string   `json:"issuer"`
	NotBefore  string   `json:"not_before"`
	NotAfter   string   `json:"not_after"`
	DaysLeft   int      `json:"days_left"`
	DNSNames   []string `json:"dns_names"`
	IsExpired  bool     `json:"is_expired"`
	IsExpiring bool     `json:"is_expiring"`
}

// HostInfo 包含服务器的基本硬件和系统状态
type HostInfo struct {
	Hostname  string        `json:"hostname"`
	OS        string        `json:"os"`
	Arch      string        `json:"arch"`
	CPUs      int           `json:"cpus"`
	Memory    *MemInfo      `json:"memory"`
	Disk      []DiskInfo    `json:"disk"`
	Load      string        `json:"load"`
	Processes []ProcessInfo `json:"processes"`
	UptimeStr string        `json:"uptime"`
}

// MemInfo 表示内存使用情况
type MemInfo struct {
	Total     string `json:"total"`
	Used      string `json:"used"`
	Free      string `json:"free"`
	UsageRate string `json:"usage_rate"`
}

// DiskInfo 表示磁盘分区的使用情况
type DiskInfo struct {
	Mount   string `json:"mount"`
	Total   string `json:"total"`
	Used    string `json:"used"`
	Avail   string `json:"avail"`
	Percent string `json:"percent"`
}

// ProcessInfo 表示单个进程的资源占用
type ProcessInfo struct {
	PID  string `json:"pid"`
	Name string `json:"name"`
	CPU  string `json:"cpu"`
	Mem  string `json:"mem"`
}

// PingTarget 定义需要 TCP 连通性检测的目标
type PingTarget struct {
	Name string `json:"name"`
	Host string `json:"host"`
}

// PingResult 单台目标的检测结果
type PingResult struct {
	Name    string `json:"name"`
	Host    string `json:"host"`
	Latency int64  `json:"latency_ms"`
	Status  string `json:"status"`
}

// WSMessage WebSocket 推送消息体
type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// SpeedData 实时带宽数据
type SpeedData struct {
	UploadSpeed   float64 `json:"upload_speed"`
	DownloadSpeed float64 `json:"download_speed"`
	UploadStr     string  `json:"upload_str"`
	DownloadStr   string  `json:"download_str"`
}

// SubscribeLink 订阅链接
type SubscribeLink struct {
	Name   string `json:"name"`
	URI    string `json:"uri"`
	Base64 string `json:"base64"`
}

// DailyTrafficItem 每日流量记录
type DailyTrafficItem struct {
	Date     string `json:"date"`
	Uplink   int64  `json:"uplink"`
	Downlink int64  `json:"downlink"`
	Total    int64  `json:"total"`
}

// UserHistoryItem 用户历史流量汇总
type UserHistoryItem struct {
	Email        string `json:"email"`
	HistUplink   int64  `json:"hist_uplink"`
	HistDownlink int64  `json:"hist_downlink"`
	HistTotal    int64  `json:"hist_total"`
}
