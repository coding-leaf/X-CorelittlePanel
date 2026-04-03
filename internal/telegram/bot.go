package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"xray-panel/internal/monitor"
	pb "xray-panel/proto"
	"xray-panel/internal/store"
	"xray-panel/internal/types"
	"xray-panel/internal/xray"
)

// Bot Telegram Bot 实例
type Bot struct {
	Token  string
	ChatID int64
	apiURL string
	mu     sync.Mutex
	offset int64

	// 依赖注入
	XrayClient     *xray.Client
	TrafficStore   *store.TrafficStore
	Config         *types.Config
	SpeedGetter    func() types.SpeedData
	VersionChecker *monitor.VersionChecker
	ChinaTZ        *time.Location
}

// New 创建 Bot 实例
func New(cfg *types.Config, xc *xray.Client, ts *store.TrafficStore, tz *time.Location, speedGetter func() types.SpeedData) *Bot {
	chatID, err := strconv.ParseInt(cfg.TelegramChatID, 10, 64)
	if err != nil {
		log.Printf("[TelegramBot] ChatID 格式错误: %v", err)
		return nil
	}
	return &Bot{
		Token:        cfg.TelegramToken,
		ChatID:       chatID,
		apiURL:       "https://api.telegram.org/bot" + cfg.TelegramToken,
		XrayClient:   xc,
		TrafficStore: ts,
		Config:       cfg,
		SpeedGetter:  speedGetter,
		ChinaTZ:      tz,
	}
}

// ===== Telegram API types =====

type TGUpdate struct {
	UpdateID      int64            `json:"update_id"`
	Message       *TGMessage       `json:"message"`
	CallbackQuery *TGCallbackQuery `json:"callback_query"`
}

type TGMessage struct {
	MessageID int64   `json:"message_id"`
	Chat      TGChat  `json:"chat"`
	Text      string  `json:"text"`
	From      *TGUser `json:"from"`
}

type TGChat struct {
	ID int64 `json:"id"`
}

type TGUser struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type TGCallbackQuery struct {
	ID      string     `json:"id"`
	Data    string     `json:"data"`
	Message *TGMessage `json:"message"`
	From    *TGUser    `json:"from"`
}

type TGInlineKeyboard struct {
	InlineKeyboard [][]TGInlineButton `json:"inline_keyboard"`
}

type TGInlineButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

// ===== API methods =====

func (b *Bot) apiCall(method string, payload interface{}) (json.RawMessage, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(b.apiURL+"/"+method, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var result struct {
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result"`
	}
	json.Unmarshal(data, &result)
	if !result.OK {
		return nil, fmt.Errorf("telegram API error: %s", string(data))
	}
	return result.Result, nil
}

func (b *Bot) SendMessage(chatID int64, text string, parseMode string) error {
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": parseMode,
	}
	_, err := b.apiCall("sendMessage", payload)
	return err
}

func (b *Bot) SendMessageWithKeyboard(chatID int64, text string, parseMode string, keyboard TGInlineKeyboard) error {
	payload := map[string]interface{}{
		"chat_id":      chatID,
		"text":         text,
		"parse_mode":   parseMode,
		"reply_markup": keyboard,
	}
	_, err := b.apiCall("sendMessage", payload)
	return err
}

func (b *Bot) AnswerCallback(callbackID string, text string) {
	b.apiCall("answerCallbackQuery", map[string]interface{}{
		"callback_query_id": callbackID,
		"text":              text,
	})
}

func (b *Bot) EditMessage(chatID int64, msgID int64, text string, parseMode string) {
	b.apiCall("editMessageText", map[string]interface{}{
		"chat_id":    chatID,
		"message_id": msgID,
		"text":       text,
		"parse_mode": parseMode,
	})
}

// SendNotification 发送管理员通知
func (b *Bot) SendNotification(text string) {
	b.SendMessage(b.ChatID, text, "Markdown")
}

func (b *Bot) RegisterCommands() {
	commands := []map[string]string{
		{"command": "status", "description": "系统概览"},
		{"command": "users", "description": "用户列表与流量"},
		{"command": "traffic", "description": "入站出站流量统计"},
		{"command": "sysinfo", "description": "主机信息"},
		{"command": "cert", "description": "证书状态"},
		{"command": "ping", "description": "网络连通性检测"},
		{"command": "version", "description": "Xray 版本检查"},
		{"command": "update", "description": "更新 Xray-core"},
		{"command": "restart", "description": "重启 Xray"},
		{"command": "reload", "description": "重载 Xray 配置"},
		{"command": "sub", "description": "生成订阅链接 (/sub email)"},
		{"command": "help", "description": "帮助"},
	}
	b.apiCall("setMyCommands", map[string]interface{}{"commands": commands})
}

// ===== Command handlers =====

func (b *Bot) handleCommand(msg *TGMessage) {
	if msg.Chat.ID != b.ChatID {
		b.SendMessage(msg.Chat.ID, "未授权访问", "")
		return
	}

	parts := strings.Fields(msg.Text)
	if len(parts) == 0 {
		return
	}
	cmd := strings.TrimPrefix(parts[0], "/")
	cmd = strings.Split(cmd, "@")[0]

	switch cmd {
	case "start", "help":
		b.cmdHelp(msg)
	case "status":
		b.cmdStatus(msg)
	case "users":
		b.cmdUsers(msg)
	case "traffic":
		b.cmdTraffic(msg)
	case "sysinfo":
		b.cmdSysInfo(msg)
	case "cert":
		b.cmdCert(msg)
	case "ping":
		b.cmdPing(msg)
	case "restart":
		b.cmdRestart(msg)
	case "reload":
		b.cmdReload(msg)
	case "version":
		b.cmdVersion(msg)
	case "update":
		b.cmdUpdate(msg)
	case "sub":
		if len(parts) > 1 {
			b.cmdSub(msg, parts[1])
		} else {
			b.SendMessage(msg.Chat.ID, "用法: `/sub 用户email`", "Markdown")
		}
	default:
		b.SendMessage(msg.Chat.ID, "未知命令，输入 /help 查看帮助", "")
	}
}

func (b *Bot) cmdHelp(msg *TGMessage) {
	text := `*Xray Panel Bot*

/status — 系统概览
/users — 用户流量
/traffic — 入站出站统计
/sysinfo — 主机信息
/cert — 证书状态
/ping — 网络检测
/version — 版本检查
/update — 更新 Xray-core
/restart — 重启 Xray
/reload — 重载配置
/sub email — 生成订阅链接`
	b.SendMessage(msg.Chat.ID, text, "Markdown")
}

func (b *Bot) cmdStatus(msg *TGMessage) {
	xrayStatus := XrayServiceStatus()
	statusEmoji := "OK"
	if xrayStatus != "active" {
		statusEmoji = "DOWN"
	}

	var statsText string
	conn, err := b.XrayClient.GetStatsConn()
	if err != nil {
		statsText = "Xray API 连接失败"
	} else {
		client := pb.NewStatsServiceClient(conn)
		stats, err := b.XrayClient.QueryAllStats(client)
		if err != nil {
			statsText = "查询失败"
		} else {
			users, _, _ := xray.ParseStats(stats)
			sysStats, _ := b.XrayClient.GetSysStats(client)

			var totalUp, totalDown int64
			for _, u := range users {
				totalUp += u.Uplink
				totalDown += u.Downlink
			}

			statsText = fmt.Sprintf("Up: `%s`\nDown: `%s`\nTotal: `%s`",
				monitor.FormatBytesGo(totalUp), monitor.FormatBytesGo(totalDown), monitor.FormatBytesGo(totalUp+totalDown))

			if sysStats != nil {
				statsText += fmt.Sprintf("\nUptime: `%s`", monitor.FormatUptimeGo(sysStats.Uptime))
			}
		}
		conn.Close()
	}

	sd := b.SpeedGetter()
	text := fmt.Sprintf("*System Overview*\n\nXray: %s `%s`\n\n%s\n\nSpeed: Up `%s` Down `%s`",
		statusEmoji, xrayStatus, statsText, sd.UploadStr, sd.DownloadStr)
	b.SendMessage(msg.Chat.ID, text, "Markdown")
}

func (b *Bot) cmdUsers(msg *TGMessage) {
	conn, err := b.XrayClient.GetStatsConn()
	if err != nil {
		b.SendMessage(msg.Chat.ID, "Xray API 连接失败", "")
		return
	}
	defer conn.Close()

	client := pb.NewStatsServiceClient(conn)
	stats, err := b.XrayClient.QueryAllStats(client)
	if err != nil {
		b.SendMessage(msg.Chat.ID, "查询失败", "")
		return
	}

	users, _, _ := xray.ParseStats(stats)
	if len(users) == 0 {
		b.SendMessage(msg.Chat.ID, "暂无用户数据", "")
		return
	}

	var sb strings.Builder
	sb.WriteString("*用户流量*\n\n")
	for i, u := range users {
		sb.WriteString(fmt.Sprintf("%d. `%s`\n   Up %s Down %s Total %s\n",
			i+1, u.Email,
			monitor.FormatBytesGo(u.Uplink), monitor.FormatBytesGo(u.Downlink), monitor.FormatBytesGo(u.Total)))
	}

	if b.TrafficStore != nil {
		hist := b.TrafficStore.GetHistoricalTraffic()
		if len(hist) > 0 {
			sb.WriteString("\n*历史总量*\n")
			for _, u := range users {
				if h, ok := hist[u.Email]; ok {
					sb.WriteString(fmt.Sprintf("- `%s`: %s\n", u.Email, monitor.FormatBytesGo(h.Uplink+h.Downlink)))
				}
			}
		}
	}

	var buttons [][]TGInlineButton
	for _, u := range users {
		buttons = append(buttons, []TGInlineButton{
			{Text: "订阅 " + u.Email, CallbackData: "sub:" + u.Email},
		})
	}
	kb := TGInlineKeyboard{InlineKeyboard: buttons}
	b.SendMessageWithKeyboard(msg.Chat.ID, sb.String(), "Markdown", kb)
}

func (b *Bot) cmdTraffic(msg *TGMessage) {
	conn, err := b.XrayClient.GetStatsConn()
	if err != nil {
		b.SendMessage(msg.Chat.ID, "Xray API 连接失败", "")
		return
	}
	defer conn.Close()

	client := pb.NewStatsServiceClient(conn)
	stats, err := b.XrayClient.QueryAllStats(client)
	if err != nil {
		b.SendMessage(msg.Chat.ID, "查询失败", "")
		return
	}

	_, inbounds, outbounds := xray.ParseStats(stats)

	var sb strings.Builder
	sb.WriteString("*流量统计*\n\n*入站:*\n")
	for _, ib := range inbounds {
		sb.WriteString(fmt.Sprintf("- `%s` Up %s Down %s\n", ib.Tag, monitor.FormatBytesGo(ib.Uplink), monitor.FormatBytesGo(ib.Downlink)))
	}
	sb.WriteString("\n*出站:*\n")
	for _, ob := range outbounds {
		sb.WriteString(fmt.Sprintf("- `%s` Up %s Down %s\n", ob.Tag, monitor.FormatBytesGo(ob.Uplink), monitor.FormatBytesGo(ob.Downlink)))
	}
	b.SendMessage(msg.Chat.ID, sb.String(), "Markdown")
}

func (b *Bot) cmdSysInfo(msg *TGMessage) {
	info := monitor.CollectSysInfo()
	var sb strings.Builder
	sb.WriteString("*主机信息*\n\n")
	sb.WriteString(fmt.Sprintf("主机: `%s`\n", info.Hostname))
	sb.WriteString(fmt.Sprintf("系统: `%s/%s`\n", info.OS, info.Arch))
	sb.WriteString(fmt.Sprintf("CPU: `%d` 核\n", info.CPUs))
	if info.Memory != nil {
		sb.WriteString(fmt.Sprintf("内存: `%s` / `%s` (%s)\n", info.Memory.Used, info.Memory.Total, info.Memory.UsageRate))
	}
	if info.Load != "" {
		parts := strings.Fields(info.Load)
		if len(parts) >= 3 {
			sb.WriteString(fmt.Sprintf("负载: `%s %s %s`\n", parts[0], parts[1], parts[2]))
		}
	}
	if info.UptimeStr != "" {
		sb.WriteString(fmt.Sprintf("运行: `%s`\n", strings.Replace(info.UptimeStr, "up ", "", 1)))
	}
	if len(info.Disk) > 0 {
		sb.WriteString("\n*磁盘:*\n")
		for _, d := range info.Disk {
			sb.WriteString(fmt.Sprintf("- `%s` %s/%s (%s)\n", d.Mount, d.Used, d.Total, d.Percent))
		}
	}
	b.SendMessage(msg.Chat.ID, sb.String(), "Markdown")
}

func (b *Bot) cmdCert(msg *TGMessage) {
	if b.Config.CertPath == "" {
		b.SendMessage(msg.Chat.ID, "未配置 cert\\_path", "Markdown")
		return
	}
	info, err := ParseCertificate(b.Config.CertPath)
	if err != nil {
		b.SendMessage(msg.Chat.ID, err.Error(), "")
		return
	}

	emoji := "OK"
	if info.IsExpired {
		emoji = "EXPIRED"
	} else if info.IsExpiring {
		emoji = "EXPIRING"
	}

	text := fmt.Sprintf("*证书状态* %s\n\n域名: `%s`\n颁发: `%s`\n有效期: `%s`\n到期: `%s`\n剩余: `%d 天`\nSAN: `%s`",
		emoji, info.Subject, info.Issuer, info.NotBefore, info.NotAfter, info.DaysLeft,
		strings.Join(info.DNSNames, ", "))
	b.SendMessage(msg.Chat.ID, text, "Markdown")
}

func (b *Bot) cmdPing(msg *TGMessage) {
	b.SendMessage(msg.Chat.ID, "正在检测网络...", "")
	results := monitor.RunPingAll(monitor.DefaultTargets)

	var sb strings.Builder
	sb.WriteString("*网络检测结果*\n\n")
	for _, r := range results {
		if r.Status == "ok" {
			sb.WriteString(fmt.Sprintf("OK %s: `%dms`\n", r.Name, r.Latency))
		} else {
			sb.WriteString(fmt.Sprintf("FAIL %s: 超时\n", r.Name))
		}
	}
	b.SendMessage(msg.Chat.ID, sb.String(), "Markdown")
}

func (b *Bot) cmdRestart(msg *TGMessage) {
	kb := TGInlineKeyboard{
		InlineKeyboard: [][]TGInlineButton{
			{
				{Text: "确认重启", CallbackData: "confirm_restart"},
				{Text: "取消", CallbackData: "cancel"},
			},
		},
	}
	b.SendMessageWithKeyboard(msg.Chat.ID, "确认重启 Xray？", "", kb)
}

func (b *Bot) cmdReload(msg *TGMessage) {
	kb := TGInlineKeyboard{
		InlineKeyboard: [][]TGInlineButton{
			{
				{Text: "确认重载", CallbackData: "confirm_reload"},
				{Text: "取消", CallbackData: "cancel"},
			},
		},
	}
	b.SendMessageWithKeyboard(msg.Chat.ID, "确认重载 Xray 配置？", "", kb)
}

func (b *Bot) cmdSub(msg *TGMessage, email string) {
	links, err := GenerateSubscribeLinks(b.Config, email)
	if err != nil {
		b.SendMessage(msg.Chat.ID, err.Error(), "")
		return
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*订阅链接 — %s*\n\n", email))
	for _, l := range links {
		sb.WriteString(fmt.Sprintf("*%s:*\n`%s`\n\n", l.Name, l.URI))
	}
	b.SendMessage(msg.Chat.ID, sb.String(), "Markdown")
}

func (b *Bot) cmdVersion(msg *TGMessage) {
	if b.VersionChecker == nil {
		b.SendMessage(msg.Chat.ID, "版本检查器未初始化", "")
		return
	}
	info := b.VersionChecker.Check()
	status := "✅ 已是最新"
	if info.HasUpdate {
		status = "⬆️ 有新版本可用"
	}
	text := fmt.Sprintf("*Xray 版本检查* %s\n\n当前: `%s`\n最新: `%s`\n检查: `%s`",
		status, info.Current, info.Latest, info.CheckedAt)
	if info.Error != "" {
		text += fmt.Sprintf("\n错误: `%s`", info.Error)
	}
	b.SendMessage(msg.Chat.ID, text, "Markdown")
}

func (b *Bot) cmdUpdate(msg *TGMessage) {
	kb := TGInlineKeyboard{
		InlineKeyboard: [][]TGInlineButton{
			{
				{Text: "确认更新 Xray", CallbackData: "confirm_update"},
				{Text: "取消", CallbackData: "cancel"},
			},
		},
	}
	b.SendMessageWithKeyboard(msg.Chat.ID, "⚠️ 确认更新 Xray-core 到最新版？\n更新将执行官方安装脚本并重启 Xray 服务。", "", kb)
}

// handleCallback 处理回调按钮
func (b *Bot) handleCallback(cb *TGCallbackQuery) {
	if cb.Message == nil || cb.Message.Chat.ID != b.ChatID {
		b.AnswerCallback(cb.ID, "未授权")
		return
	}

	switch {
	case cb.Data == "confirm_restart":
		b.AnswerCallback(cb.ID, "正在重启...")
		b.EditMessage(cb.Message.Chat.ID, cb.Message.MessageID, "正在重启 Xray...", "")
		output, err := XrayServiceCmd("restart")
		if err != nil {
			b.EditMessage(cb.Message.Chat.ID, cb.Message.MessageID, "重启失败: "+output, "")
		} else {
			b.EditMessage(cb.Message.Chat.ID, cb.Message.MessageID, "Xray 已重启", "")
		}

	case cb.Data == "confirm_reload":
		b.AnswerCallback(cb.ID, "正在重载...")
		b.EditMessage(cb.Message.Chat.ID, cb.Message.MessageID, "正在重载配置...", "")
		output, err := XrayServiceCmd("reload")
		if err != nil {
			b.EditMessage(cb.Message.Chat.ID, cb.Message.MessageID, "重载失败: "+output, "")
		} else {
			b.EditMessage(cb.Message.Chat.ID, cb.Message.MessageID, "Xray 配置已重载", "")
		}

	case cb.Data == "confirm_update":
		b.AnswerCallback(cb.ID, "正在更新...")
		b.EditMessage(cb.Message.Chat.ID, cb.Message.MessageID, "⏳ 正在更新 Xray-core，请稍候...", "")
		output, err := monitor.RunXrayUpdate()
		if err != nil {
			b.EditMessage(cb.Message.Chat.ID, cb.Message.MessageID, "❌ 更新失败:\n"+truncateOutput(output, 500), "")
		} else {
			// 更新后重新检查版本
			ver := ""
			if b.VersionChecker != nil {
				info := b.VersionChecker.Check()
				ver = info.Current
			}
			b.EditMessage(cb.Message.Chat.ID, cb.Message.MessageID, "✅ Xray 更新完成\n当前版本: "+ver, "")
		}

	case cb.Data == "cancel":
		b.AnswerCallback(cb.ID, "已取消")
		b.EditMessage(cb.Message.Chat.ID, cb.Message.MessageID, "操作已取消", "")

	case strings.HasPrefix(cb.Data, "sub:"):
		email := strings.TrimPrefix(cb.Data, "sub:")
		b.AnswerCallback(cb.ID, "生成中...")
		b.cmdSub(cb.Message, email)
	}
}

// StartPolling 启动消息轮询
func (b *Bot) StartPolling() {
	b.RegisterCommands()
	log.Printf("[TelegramBot] 开始轮询消息 (ChatID: %d)", b.ChatID)

	go func() {
		for {
			updates, err := b.getUpdates()
			if err != nil {
				log.Printf("[TelegramBot] 轮询错误: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}
			for _, u := range updates {
				if u.UpdateID >= b.offset {
					b.offset = u.UpdateID + 1
				}
				if u.Message != nil && strings.HasPrefix(u.Message.Text, "/") {
					b.handleCommand(u.Message)
				}
				if u.CallbackQuery != nil {
					b.handleCallback(u.CallbackQuery)
				}
			}
		}
	}()
}

func (b *Bot) getUpdates() ([]TGUpdate, error) {
	payload := map[string]interface{}{
		"offset":  b.offset,
		"timeout": 30,
	}
	result, err := b.apiCall("getUpdates", payload)
	if err != nil {
		return nil, err
	}
	var updates []TGUpdate
	json.Unmarshal(result, &updates)
	return updates, nil
}

// StartCertChecker 每日检查证书过期
func (b *Bot) StartCertChecker() {
	if b.Config.CertPath == "" {
		return
	}
	go func() {
		for {
			time.Sleep(24 * time.Hour)
			info, err := ParseCertificate(b.Config.CertPath)
			if err != nil {
				continue
			}
			if info.DaysLeft <= 7 && info.DaysLeft >= 0 {
				b.SendNotification(fmt.Sprintf("*证书即将过期*\n\n域名: `%s`\n剩余: `%d 天`\n到期: `%s`",
					info.Subject, info.DaysLeft, info.NotAfter))
			}
		}
	}()
}

func truncateOutput(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... (截断)"
}
