package main

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
	pb "xray-panel/proto"
)

// ============================================================
// Telegram Bot — Interactive management + notifications
// ============================================================

var tgBot *TelegramBot

// TelegramBot represents a Telegram Bot instance used for notifications and remote control.
type TelegramBot struct {
	Token  string // Telegram Bot API Token
	ChatID int64  // The authorized Chat ID (user or group) that can interact with the bot
	apiURL string // Base URL for Telegram API requests
	mu     sync.Mutex
	offset int64 // Used for long polling to track the last processed update
}

// NewTelegramBot initializes a new TelegramBot with the given token and chat ID.
func NewTelegramBot(token string, chatID int64) *TelegramBot {
	return &TelegramBot{
		Token:  token,
		ChatID: chatID,
		apiURL: "https://api.telegram.org/bot" + token,
	}
}

// ============================================================
// Telegram API types
// ============================================================

// TGUpdate represents a single update received from Telegram API.
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

// ============================================================
// API methods
// ============================================================

// apiCall executes an HTTP request to the Telegram Bot API.
func (b *TelegramBot) apiCall(method string, payload interface{}) (json.RawMessage, error) {
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

// SendMessage sends a simple text message to the specified chat.
// Uses parseMode (e.g. "Markdown" or "HTML") for text formatting.
func (b *TelegramBot) SendMessage(chatID int64, text string, parseMode string) error {
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": parseMode,
	}
	_, err := b.apiCall("sendMessage", payload)
	return err
}

func (b *TelegramBot) SendMessageWithKeyboard(chatID int64, text string, parseMode string, keyboard TGInlineKeyboard) error {
	payload := map[string]interface{}{
		"chat_id":      chatID,
		"text":         text,
		"parse_mode":   parseMode,
		"reply_markup": keyboard,
	}
	_, err := b.apiCall("sendMessage", payload)
	return err
}

func (b *TelegramBot) AnswerCallback(callbackID string, text string) {
	b.apiCall("answerCallbackQuery", map[string]interface{}{
		"callback_query_id": callbackID,
		"text":              text,
	})
}

func (b *TelegramBot) EditMessage(chatID int64, msgID int64, text string, parseMode string) {
	b.apiCall("editMessageText", map[string]interface{}{
		"chat_id":    chatID,
		"message_id": msgID,
		"text":       text,
		"parse_mode": parseMode,
	})
}

// SendNotification is a helper to push a Markdown-formatted message to the authorized admin chat.
func (b *TelegramBot) SendNotification(text string) {
	b.SendMessage(b.ChatID, text, "Markdown")
}

// ============================================================
// Register bot commands
// ============================================================

// RegisterCommands informs Telegram of the bot's available slash commands,
// so they appear as suggestions in the Telegram client UI.
func (b *TelegramBot) RegisterCommands() {
	commands := []map[string]string{
		{"command": "status", "description": "📊 系统概览"},
		{"command": "users", "description": "👤 用户列表与流量"},
		{"command": "traffic", "description": "📈 入站出站流量统计"},
		{"command": "sysinfo", "description": "🖥 主机信息"},
		{"command": "cert", "description": "🔐 证书状态"},
		{"command": "ping", "description": "🌐 网络连通性检测"},
		{"command": "restart", "description": "🔄 重启 Xray"},
		{"command": "reload", "description": "♻️ 重载 Xray 配置"},
		{"command": "sub", "description": "🔗 生成订阅链接 (/sub email)"},
		{"command": "help", "description": "📖 帮助"},
	}
	b.apiCall("setMyCommands", map[string]interface{}{"commands": commands})
}

// ============================================================
// Command handlers
// ============================================================

// handleCommand routes recognized Telegram slash commands to their respective Handler functions.
func (b *TelegramBot) handleCommand(msg *TGMessage) {
	// Only respond to configured chat
	if msg.Chat.ID != b.ChatID {
		b.SendMessage(msg.Chat.ID, "⛔ 未授权访问", "")
		return
	}

	parts := strings.Fields(msg.Text)
	if len(parts) == 0 {
		return
	}
	cmd := strings.TrimPrefix(parts[0], "/")
	cmd = strings.Split(cmd, "@")[0] // Remove @botname suffix

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

func (b *TelegramBot) cmdHelp(msg *TGMessage) {
	text := `📖 *Xray Panel Bot*

/status — 📊 系统概览
/users — 👤 用户流量
/traffic — 📈 入站出站统计
/sysinfo — 🖥 主机信息
/cert — 🔐 证书状态
/ping — 🌐 网络检测
/restart — 🔄 重启 Xray
/reload — ♻️ 重载配置
/sub email — 🔗 生成订阅链接`
	b.SendMessage(msg.Chat.ID, text, "Markdown")
}

func (b *TelegramBot) cmdStatus(msg *TGMessage) {
	// Xray status
	xrayStatus := getXrayStatus()
	statusEmoji := "🟢"
	if xrayStatus != "active" {
		statusEmoji = "🔴"
	}

	// Get stats from Xray API
	var statsText string
	conn, err := getStatsConn()
	if err != nil {
		statsText = "⚠️ Xray API 连接失败"
	} else {
		client := pb.NewStatsServiceClient(conn)
		stats, err := queryAllStats(client)
		if err != nil {
			statsText = "⚠️ 查询失败"
		} else {
			users, _, _ := parseStats(stats)
			sysStats, _ := getSysStats(client)

			var totalUp, totalDown int64
			for _, u := range users {
				totalUp += u.Uplink
				totalDown += u.Downlink
			}

			statsText = fmt.Sprintf("↑ 上行: `%s`\n↓ 下行: `%s`\n📦 总量: `%s`",
				formatBytesGo(totalUp), formatBytesGo(totalDown), formatBytesGo(totalUp+totalDown))

			if sysStats != nil {
				statsText += fmt.Sprintf("\n⏱ 运行: `%s`", formatUptimeGo(sysStats.Uptime))
			}
		}
		conn.Close()
	}

	// Speed
	speedMu.Lock()
	sd := currentSpeed
	speedMu.Unlock()

	text := fmt.Sprintf("📊 *系统概览*\n\nXray: %s `%s`\n\n%s\n\n🚀 速率: ↑`%s` ↓`%s`",
		statusEmoji, xrayStatus, statsText, sd.UploadStr, sd.DownloadStr)
	b.SendMessage(msg.Chat.ID, text, "Markdown")
}

func (b *TelegramBot) cmdUsers(msg *TGMessage) {
	conn, err := getStatsConn()
	if err != nil {
		b.SendMessage(msg.Chat.ID, "⚠️ Xray API 连接失败", "")
		return
	}
	defer conn.Close()

	client := pb.NewStatsServiceClient(conn)
	stats, err := queryAllStats(client)
	if err != nil {
		b.SendMessage(msg.Chat.ID, "⚠️ 查询失败", "")
		return
	}

	users, _, _ := parseStats(stats)
	if len(users) == 0 {
		b.SendMessage(msg.Chat.ID, "暂无用户数据", "")
		return
	}

	var sb strings.Builder
	sb.WriteString("👤 *用户流量*\n\n")
	for i, u := range users {
		sb.WriteString(fmt.Sprintf("%d. `%s`\n   ↑%s ↓%s 📦%s\n",
			i+1, u.Email,
			formatBytesGo(u.Uplink), formatBytesGo(u.Downlink), formatBytesGo(u.Total)))
	}

	// Add historical data if available
	if trafficStore != nil {
		hist := trafficStore.GetHistoricalTraffic()
		if len(hist) > 0 {
			sb.WriteString("\n📚 *历史总量*\n")
			for _, u := range users {
				if h, ok := hist[u.Email]; ok {
					sb.WriteString(fmt.Sprintf("• `%s`: %s\n", u.Email, formatBytesGo(h.Uplink+h.Downlink)))
				}
			}
		}
	}

	// Inline buttons for each user
	var buttons [][]TGInlineButton
	for _, u := range users {
		buttons = append(buttons, []TGInlineButton{
			{Text: "🔗 订阅 " + u.Email, CallbackData: "sub:" + u.Email},
		})
	}
	kb := TGInlineKeyboard{InlineKeyboard: buttons}
	b.SendMessageWithKeyboard(msg.Chat.ID, sb.String(), "Markdown", kb)
}

func (b *TelegramBot) cmdTraffic(msg *TGMessage) {
	conn, err := getStatsConn()
	if err != nil {
		b.SendMessage(msg.Chat.ID, "⚠️ Xray API 连接失败", "")
		return
	}
	defer conn.Close()

	client := pb.NewStatsServiceClient(conn)
	stats, err := queryAllStats(client)
	if err != nil {
		b.SendMessage(msg.Chat.ID, "⚠️ 查询失败", "")
		return
	}

	_, inbounds, outbounds := parseStats(stats)

	var sb strings.Builder
	sb.WriteString("📈 *流量统计*\n\n")
	sb.WriteString("*入站:*\n")
	for _, ib := range inbounds {
		sb.WriteString(fmt.Sprintf("• `%s` ↑%s ↓%s\n", ib.Tag, formatBytesGo(ib.Uplink), formatBytesGo(ib.Downlink)))
	}
	sb.WriteString("\n*出站:*\n")
	for _, ob := range outbounds {
		sb.WriteString(fmt.Sprintf("• `%s` ↑%s ↓%s\n", ob.Tag, formatBytesGo(ob.Uplink), formatBytesGo(ob.Downlink)))
	}

	b.SendMessage(msg.Chat.ID, sb.String(), "Markdown")
}

func (b *TelegramBot) cmdSysInfo(msg *TGMessage) {
	info := collectSysInfo()
	var sb strings.Builder
	sb.WriteString("🖥 *主机信息*\n\n")
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
			sb.WriteString(fmt.Sprintf("• `%s` %s/%s (%s)\n", d.Mount, d.Used, d.Total, d.Percent))
		}
	}

	b.SendMessage(msg.Chat.ID, sb.String(), "Markdown")
}

func (b *TelegramBot) cmdCert(msg *TGMessage) {
	if config.CertPath == "" {
		b.SendMessage(msg.Chat.ID, "⚠️ 未配置 cert\\_path", "Markdown")
		return
	}
	info, err := parseCertificate(config.CertPath)
	if err != nil {
		b.SendMessage(msg.Chat.ID, "⚠️ "+err.Error(), "")
		return
	}

	emoji := "🟢"
	if info.IsExpired {
		emoji = "🔴"
	} else if info.IsExpiring {
		emoji = "🟡"
	}

	text := fmt.Sprintf("🔐 *证书状态* %s\n\n域名: `%s`\n颁发: `%s`\n有效期: `%s`\n到期: `%s`\n剩余: `%d 天`\nSAN: `%s`",
		emoji, info.Subject, info.Issuer, info.NotBefore, info.NotAfter, info.DaysLeft,
		strings.Join(info.DNSNames, ", "))
	b.SendMessage(msg.Chat.ID, text, "Markdown")
}

func (b *TelegramBot) cmdPing(msg *TGMessage) {
	b.SendMessage(msg.Chat.ID, "🌐 正在检测网络...", "")

	results := make([]PingResult, len(defaultTargets))
	var wg sync.WaitGroup
	for i, t := range defaultTargets {
		wg.Add(1)
		go func(idx int, target PingTarget) {
			defer wg.Done()
			latency, err := tcpPing(target.Host, 5*time.Second)
			if err != nil {
				results[idx] = PingResult{Name: target.Name, Host: target.Host, Latency: -1, Status: "fail"}
			} else {
				results[idx] = PingResult{Name: target.Name, Host: target.Host, Latency: latency.Milliseconds(), Status: "ok"}
			}
		}(i, t)
	}
	wg.Wait()

	var sb strings.Builder
	sb.WriteString("🌐 *网络检测结果*\n\n")
	for _, r := range results {
		if r.Status == "ok" {
			sb.WriteString(fmt.Sprintf("🟢 %s: `%dms`\n", r.Name, r.Latency))
		} else {
			sb.WriteString(fmt.Sprintf("🔴 %s: 超时\n", r.Name))
		}
	}
	b.SendMessage(msg.Chat.ID, sb.String(), "Markdown")
}

func (b *TelegramBot) cmdRestart(msg *TGMessage) {
	kb := TGInlineKeyboard{
		InlineKeyboard: [][]TGInlineButton{
			{
				{Text: "✅ 确认重启", CallbackData: "confirm_restart"},
				{Text: "❌ 取消", CallbackData: "cancel"},
			},
		},
	}
	b.SendMessageWithKeyboard(msg.Chat.ID, "🔄 确认重启 Xray？", "", kb)
}

func (b *TelegramBot) cmdReload(msg *TGMessage) {
	kb := TGInlineKeyboard{
		InlineKeyboard: [][]TGInlineButton{
			{
				{Text: "✅ 确认重载", CallbackData: "confirm_reload"},
				{Text: "❌ 取消", CallbackData: "cancel"},
			},
		},
	}
	b.SendMessageWithKeyboard(msg.Chat.ID, "♻️ 确认重载 Xray 配置？", "", kb)
}

func (b *TelegramBot) cmdSub(msg *TGMessage, email string) {
	links, err := generateSubscribeLinks(email)
	if err != nil {
		b.SendMessage(msg.Chat.ID, "⚠️ "+err.Error(), "")
		return
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔗 *订阅链接 — %s*\n\n", email))
	for _, l := range links {
		sb.WriteString(fmt.Sprintf("*%s:*\n`%s`\n\n", l.Name, l.URI))
	}
	b.SendMessage(msg.Chat.ID, sb.String(), "Markdown")
}

// ============================================================
// Callback handler
// ============================================================

// handleCallback processes button clicks from inline keyboards sent by the bot.
func (b *TelegramBot) handleCallback(cb *TGCallbackQuery) {
	if cb.Message == nil || cb.Message.Chat.ID != b.ChatID {
		b.AnswerCallback(cb.ID, "未授权")
		return
	}

	switch {
	case cb.Data == "confirm_restart":
		b.AnswerCallback(cb.ID, "正在重启...")
		b.EditMessage(cb.Message.Chat.ID, cb.Message.MessageID, "🔄 正在重启 Xray...", "")
		output, err := xrayServiceCmd("restart")
		if err != nil {
			b.EditMessage(cb.Message.Chat.ID, cb.Message.MessageID, "❌ 重启失败: "+output, "")
		} else {
			b.EditMessage(cb.Message.Chat.ID, cb.Message.MessageID, "✅ Xray 已重启", "")
		}

	case cb.Data == "confirm_reload":
		b.AnswerCallback(cb.ID, "正在重载...")
		b.EditMessage(cb.Message.Chat.ID, cb.Message.MessageID, "♻️ 正在重载配置...", "")
		output, err := xrayServiceCmd("reload")
		if err != nil {
			b.EditMessage(cb.Message.Chat.ID, cb.Message.MessageID, "❌ 重载失败: "+output, "")
		} else {
			b.EditMessage(cb.Message.Chat.ID, cb.Message.MessageID, "✅ Xray 配置已重载", "")
		}

	case cb.Data == "cancel":
		b.AnswerCallback(cb.ID, "已取消")
		b.EditMessage(cb.Message.Chat.ID, cb.Message.MessageID, "❌ 操作已取消", "")

	case strings.HasPrefix(cb.Data, "sub:"):
		email := strings.TrimPrefix(cb.Data, "sub:")
		b.AnswerCallback(cb.ID, "生成中...")
		b.cmdSub(cb.Message, email)
	}
}

// ============================================================
// Polling loop
// ============================================================

// StartPolling initiates a background loop using Telegram long polling to fetch new messages and callbacks.
func (b *TelegramBot) StartPolling() {
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

func (b *TelegramBot) getUpdates() ([]TGUpdate, error) {
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

// ============================================================
// Certificate expiry checker (daily)
// ============================================================

// StartCertChecker runs a background goroutine that checks the TLS certificate expiry once a day.
// If the certificate is expiring within 7 days, it sends an alert via Telegram.
func (b *TelegramBot) StartCertChecker() {
	if config.CertPath == "" {
		return
	}
	go func() {
		for {
			time.Sleep(24 * time.Hour)
			info, err := parseCertificate(config.CertPath)
			if err != nil {
				continue
			}
			if info.DaysLeft <= 7 && info.DaysLeft >= 0 {
				emoji := "⚠️"
				if info.DaysLeft <= 3 {
					emoji = "🚨"
				}
				b.SendNotification(fmt.Sprintf("%s *证书即将过期*\n\n域名: `%s`\n剩余: `%d 天`\n到期: `%s`",
					emoji, info.Subject, info.DaysLeft, info.NotAfter))
			}
		}
	}()
}

// ============================================================
// HTTP handlers for admin panel
// ============================================================

// handleTelegramTest API: POST /admin/api/telegram/test
// Sends a manual test message to the configured Telegram chat to verify connectivity.
func handleTelegramTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(405)
		json.NewEncoder(w).Encode(map[string]string{"error": "方法不允许"})
		return
	}
	if tgBot == nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Telegram Bot 未配置"})
		return
	}
	err := tgBot.SendMessage(tgBot.ChatID, "✅ Xray Panel 测试消息\n\n"+time.Now().In(chinaTZ).Format("2006-01-02 15:04:05"), "")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "发送失败: " + err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleTelegramStatus API: GET /admin/api/telegram/status
// Returns whether the Telegram bot is configured and running.
func handleTelegramStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if tgBot == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"configured": false,
		})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"configured": true,
		"chat_id":    tgBot.ChatID,
	})
}

// ============================================================
// Helpers — format functions for bot messages
// ============================================================

func formatBytesGo(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	const k = 1024
	sizes := []string{"B", "KB", "MB", "GB", "TB"}
	i := 0
	b := float64(bytes)
	for b >= k && i < len(sizes)-1 {
		b /= k
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%.0f %s", b, sizes[i])
	}
	return fmt.Sprintf("%.2f %s", b, sizes[i])
}

func formatUptimeGo(seconds uint32) string {
	d := seconds / 86400
	h := (seconds % 86400) / 3600
	m := (seconds % 3600) / 60
	if d > 0 {
		return fmt.Sprintf("%d天%d时", d, h)
	}
	if h > 0 {
		return fmt.Sprintf("%d时%d分", h, m)
	}
	return fmt.Sprintf("%d分", m)
}

// initTelegramBot initializes the TelegramBot globally based on configurations and starts its long polling loop.
func initTelegramBot() {
	if config.TelegramToken == "" || config.TelegramChatID == "" {
		log.Printf("[TelegramBot] 未配置 Token 或 ChatID，Bot 已禁用")
		return
	}
	chatID, err := strconv.ParseInt(config.TelegramChatID, 10, 64)
	if err != nil {
		log.Printf("[TelegramBot] ChatID 格式错误: %v", err)
		return
	}
	tgBot = NewTelegramBot(config.TelegramToken, chatID)
	tgBot.StartPolling()
	tgBot.StartCertChecker()
	log.Printf("[TelegramBot] Bot 已启动")
}
