package handler

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	pb "xray-panel/proto"
	"xray-panel/internal/app"
	"xray-panel/internal/monitor"
	"xray-panel/internal/telegram"
	"xray-panel/internal/types"
	"xray-panel/internal/ws"
	"xray-panel/internal/xray"
)

// Handlers 持有 App 引用，所有 handler 方法挂在此结构体上
type Handlers struct {
	App        *app.App
	Upgrader   websocket.Upgrader
	FrontendFS embed.FS
}

// NewHandlers 创建 handler 实例
func NewHandlers(a *app.App, frontendFS embed.FS) *Handlers {
	return &Handlers{
		App:        a,
		Upgrader:   ws.NewUpgrader(),
		FrontendFS: frontendFS,
	}
}

// ===== Dashboard API =====

func (h *Handlers) HandleAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("CDN-Cache-Control", "no-store")

	conn, err := h.App.XrayClient.GetStatsConn()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "无法连接 Xray API: " + err.Error()})
		return
	}
	defer conn.Close()

	client := pb.NewStatsServiceClient(conn)
	stats, err := h.App.XrayClient.QueryAllStats(client)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "查询失败: " + err.Error()})
		return
	}

	users, inbounds, outbounds := xray.ParseStats(stats)
	sysStats, _ := h.App.XrayClient.GetSysStats(client)

	h.App.RecordHistory(users)

	if h.App.TrafficStore != nil && sysStats != nil {
		h.App.TrafficStore.Update(users, sysStats.Uptime)
	}

	hist := h.App.GetHistoryCopy()

	data := types.DashboardData{
		Users:     users,
		Inbounds:  inbounds,
		Outbounds: outbounds,
		SysStats:  sysStats,
		History:   hist,
		UpdatedAt: time.Now().In(h.App.ChinaTZ).Format("2006-01-02 15:04:05"),
	}
	json.NewEncoder(w).Encode(data)
}

func (h *Handlers) HandleUserList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	conn, err := h.App.XrayClient.GetStatsConn()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "无法连接 Xray API: " + err.Error()})
		return
	}
	defer conn.Close()

	client := pb.NewStatsServiceClient(conn)
	stats, err := h.App.XrayClient.QueryAllStats(client)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "查询失败: " + err.Error()})
		return
	}

	userSet := make(map[string]bool)
	for _, s := range stats {
		parts := strings.Split(s.Name, ">>>")
		if len(parts) == 4 && parts[0] == "user" {
			userSet[parts[1]] = true
		}
	}

	users := make([]string, 0, len(userSet))
	for u := range userSet {
		users = append(users, u)
	}
	sort.Strings(users)
	json.NewEncoder(w).Encode(users)
}

func (h *Handlers) HandleResetUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	var req struct {
		Email string `json:"email"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	conn, err := h.App.XrayClient.GetStatsConn()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer conn.Close()

	client := pb.NewStatsServiceClient(conn)
	ctx := context.Background()
	client.GetStats(ctx, &pb.GetStatsRequest{
		Name: fmt.Sprintf("user>>>%s>>>traffic>>>uplink", req.Email), Reset_: true,
	})
	client.GetStats(ctx, &pb.GetStatsRequest{
		Name: fmt.Sprintf("user>>>%s>>>traffic>>>downlink", req.Email), Reset_: true,
	})

	if h.App.TrafficStore != nil {
		h.App.TrafficStore.ResetUser(req.Email)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handlers) HandleSysInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(monitor.CollectSysInfo())
}

func (h *Handlers) HandleSpeed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	sd := h.App.SpeedTracker.GetCurrent()
	json.NewEncoder(w).Encode(sd)
}

func (h *Handlers) HandlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(405)
		json.NewEncoder(w).Encode(map[string]string{"error": "方法不允许"})
		return
	}
	results := monitor.RunPingAll(monitor.DefaultTargets)
	json.NewEncoder(w).Encode(results)
}

func (h *Handlers) HandleWS(w http.ResponseWriter, r *http.Request) {
	ws.HandleWS(h.App.WSHub, h.Upgrader, w, r)
}

// ===== Version Check / Update =====

func (h *Handlers) HandleVersionCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.App.VersionChecker == nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "版本检查器未初始化"})
		return
	}
	// 如果带 ?refresh=1 则立即检查
	if r.URL.Query().Get("refresh") == "1" {
		info := h.App.VersionChecker.Check()
		json.NewEncoder(w).Encode(info)
		return
	}
	json.NewEncoder(w).Encode(h.App.VersionChecker.GetInfo())
}

func (h *Handlers) HandleXrayUpdate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(405)
		json.NewEncoder(w).Encode(map[string]string{"error": "方法不允许"})
		return
	}
	log.Printf("[XrayUpdate] 管理员触发 Xray 更新")
	output, err := monitor.RunXrayUpdate()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"output":  output,
			"error":   err.Error(),
		})
		return
	}
	// 更新后重新检查版本
	if h.App.VersionChecker != nil {
		h.App.VersionChecker.Check()
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"output":  output,
	})
}

// ===== Traffic Store Handlers =====

func (h *Handlers) HandleTrafficHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.App.TrafficStore == nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "store not initialized"})
		return
	}
	hist := h.App.TrafficStore.GetHistoricalTraffic()
	items := make([]types.UserHistoryItem, 0, len(hist))
	for email, rec := range hist {
		items = append(items, types.UserHistoryItem{
			Email: email, HistUplink: rec.Uplink, HistDownlink: rec.Downlink,
			HistTotal: rec.Uplink + rec.Downlink,
		})
	}
	json.NewEncoder(w).Encode(items)
}

func (h *Handlers) HandleDailyTraffic(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.App.TrafficStore == nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "store not initialized"})
		return
	}
	email := r.URL.Query().Get("email")
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if val, err := strconv.Atoi(d); err == nil && val > 0 {
			days = val
		}
	}
	if days > 90 {
		days = 90
	}
	items := h.App.TrafficStore.GetDailyTraffic(email, days)
	json.NewEncoder(w).Encode(items)
}

func (h *Handlers) HandleGetCycle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.App.TrafficStore == nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "store not initialized"})
		return
	}
	email := r.URL.Query().Get("email")
	if email == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "missing email"})
		return
	}
	set := h.App.TrafficStore.GetUserSetting(email)
	json.NewEncoder(w).Encode(set)
}

func (h *Handlers) HandleSetCycle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(405)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		Email string `json:"email"`
		Day   int    `json:"day"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid format"})
		return
	}
	if req.Email == "" || req.Day < 0 || req.Day > 31 {
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid parameters"})
		return
	}
	if h.App.TrafficStore != nil {
		h.App.TrafficStore.SetUserSetting(req.Email, req.Day)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

// ===== Auth Handlers =====

func (h *Handlers) GetClientIP(r *http.Request) string {
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteIP = r.RemoteAddr
	}
	isLocalOrigin := remoteIP == "127.0.0.1" || remoteIP == "::1"
	if isLocalOrigin {
		ip := r.Header.Get("CF-Connecting-IP")
		if ip == "" {
			ip = r.Header.Get("X-Forwarded-For")
		}
		if ip != "" {
			return strings.TrimSpace(strings.Split(ip, ",")[0])
		}
	}
	return remoteIP
}

func (h *Handlers) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		h.serveFrontendFile(w, "frontend/login.html")
		return
	}

	ip := h.GetClientIP(r)
	h.App.LoginMutex.Lock()
	now := time.Now()
	rec, exists := h.App.LoginRecords[ip]
	if !exists {
		rec = &app.LoginBan{}
		h.App.LoginRecords[ip] = rec
	}

	if now.Before(rec.BlockedTill) {
		h.App.LoginMutex.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(429)
		json.NewEncoder(w).Encode(map[string]string{"error": "尝试次数过多，请15分钟后再试"})
		return
	}
	h.App.LoginMutex.Unlock()

	var req struct {
		Password string `json:"password"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Password != h.App.Config.Password {
		h.App.LoginMutex.Lock()
		rec.Fails++
		rec.LastAttempt = now
		if rec.Fails >= 5 {
			rec.BlockedTill = now.Add(15 * time.Minute)
		}
		h.App.LoginMutex.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]string{"error": "密码错误"})
		return
	}

	h.App.LoginMutex.Lock()
	delete(h.App.LoginRecords, ip)
	h.App.LoginMutex.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name: "auth", Value: h.App.AuthToken, Path: "/",
		MaxAge: 86400 * 7, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handlers) HandleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name: "auth", Value: "", Path: "/",
		MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	// 重新生成 token 使所有旧 cookie 失效
	b := make([]byte, 32)
	rand.Read(b)
	h.App.AuthToken = hex.EncodeToString(b)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ===== Log Handlers =====

func (h *Handlers) HandleAccessLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.App.Config.AccessLog == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "未配置访问日志路径"})
		return
	}
	count := 100
	if c := r.URL.Query().Get("count"); c != "" {
		if val, err := strconv.Atoi(c); err == nil && val > 0 {
			count = val
		}
	}
	emailFilter := r.URL.Query().Get("email")
	lines, err := monitor.ReadLastNLines(h.App.Config.AccessLog, count)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	entries := make([]types.AccessLogEntry, 0)
	for _, line := range lines {
		entry := monitor.ParseAccessLogLine(line)
		if entry != nil {
			if emailFilter != "" && entry.Email != emailFilter {
				continue
			}
			entries = append(entries, *entry)
		}
	}
	json.NewEncoder(w).Encode(entries)
}

func (h *Handlers) HandleErrorLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.App.Config.ErrorLog == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "未配置错误日志路径"})
		return
	}
	count := 100
	if c := r.URL.Query().Get("count"); c != "" {
		if val, err := strconv.Atoi(c); err == nil && val > 0 {
			count = val
		}
	}
	lines, err := monitor.ReadLastNLines(h.App.Config.ErrorLog, count)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	entries := make([]types.ErrorLogEntry, 0)
	for _, line := range lines {
		entry := monitor.ParseErrorLogLine(line)
		if entry != nil {
			entries = append(entries, *entry)
		}
	}
	json.NewEncoder(w).Encode(entries)
}

func (h *Handlers) HandleClearLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(405)
		json.NewEncoder(w).Encode(map[string]string{"error": "方法不允许"})
		return
	}
	var req struct {
		Type string `json:"type"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	var path string
	switch req.Type {
	case "access":
		path = h.App.Config.AccessLog
	case "error":
		path = h.App.Config.ErrorLog
	default:
		json.NewEncoder(w).Encode(map[string]string{"error": "无效类型"})
		return
	}
	if path == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "未配置日志路径"})
		return
	}
	if err := os.Truncate(path, 0); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "清除失败: " + err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ===== Admin Handlers =====

func (h *Handlers) HandleAdminPage(w http.ResponseWriter, r *http.Request) {
	h.serveFrontendFile(w, "frontend/admin.html")
}

func (h *Handlers) HandleAdminUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.App.Config.XrayConfigPath == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "未配置 xray_config_path"})
		return
	}
	cfg, err := xray.ReadXrayConfig(h.App.Config.XrayConfigPath)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	clients := xray.ExtractClients(cfg)
	if clients == nil {
		clients = []types.XrayClient{}
	}
	json.NewEncoder(w).Encode(clients)
}

func (h *Handlers) HandleAdminAddUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(405)
		json.NewEncoder(w).Encode(map[string]string{"error": "方法不允许"})
		return
	}
	var req struct {
		Email string `json:"email"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Email == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "email 不能为空"})
		return
	}
	if strings.ContainsAny(req.Email, "><:\\/\"'") {
		json.NewEncoder(w).Encode(map[string]string{"error": "email 包含非法特殊字符"})
		return
	}

	cfg, err := xray.ReadXrayConfig(h.App.Config.XrayConfigPath)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	newUUID := uuid.New().String()
	client := types.XrayClient{ID: newUUID, Email: req.Email}
	if err := xray.AddClientToConfig(cfg, client); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if err := xray.WriteXrayConfig(h.App.Config.XrayConfigPath, cfg); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	output, err := telegram.XrayServiceCmd("reload")
	reloadStatus := "ok"
	if err != nil {
		reloadStatus = "reload失败: " + output
		log.Printf("[Admin] 添加用户后重启 Xray 失败: %v, 输出: %s", err, output)
	}

	if h.App.TGBot != nil {
		h.App.TGBot.SendNotification(fmt.Sprintf("*新用户添加*\nEmail: `%s`\nUUID: `%s`", req.Email, newUUID))
	}

	log.Printf("[Admin] 添加用户: %s (UUID: %s)", req.Email, newUUID)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok", "uuid": newUUID, "reload": reloadStatus,
	})
}

func (h *Handlers) HandleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(405)
		json.NewEncoder(w).Encode(map[string]string{"error": "方法不允许"})
		return
	}
	var req struct {
		Email string `json:"email"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Email == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "email 不能为空"})
		return
	}

	cfg, err := xray.ReadXrayConfig(h.App.Config.XrayConfigPath)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if err := xray.RemoveClientFromConfig(cfg, req.Email); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if err := xray.WriteXrayConfig(h.App.Config.XrayConfigPath, cfg); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	output, err := telegram.XrayServiceCmd("reload")
	reloadStatus := "ok"
	if err != nil {
		reloadStatus = "reload失败: " + output
	}

	if h.App.TrafficStore != nil {
		h.App.TrafficStore.ResetUser(req.Email)
	}
	if h.App.TGBot != nil {
		h.App.TGBot.SendNotification(fmt.Sprintf("*用户已删除*\nEmail: `%s`", req.Email))
	}

	log.Printf("[Admin] 删除用户: %s", req.Email)
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "reload": reloadStatus})
}

func (h *Handlers) HandleXrayRestart(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(405)
		json.NewEncoder(w).Encode(map[string]string{"error": "方法不允许"})
		return
	}
	output, err := telegram.XrayServiceCmd("restart")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "重启失败: " + output})
		return
	}
	if h.App.TGBot != nil {
		h.App.TGBot.SendNotification("*Xray 已重启*")
	}
	log.Printf("[Admin] Xray 重启")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handlers) HandleXrayReload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(405)
		json.NewEncoder(w).Encode(map[string]string{"error": "方法不允许"})
		return
	}
	output, err := telegram.XrayServiceCmd("reload")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "重载失败: " + output})
		return
	}
	if h.App.TGBot != nil {
		h.App.TGBot.SendNotification("*Xray 配置已重载*")
	}
	log.Printf("[Admin] Xray 重载配置")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "message": strings.TrimSpace(output)})
}

func (h *Handlers) HandleXrayStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status := telegram.XrayServiceStatus()
	json.NewEncoder(w).Encode(map[string]string{"status": status})
}

func (h *Handlers) HandleConfigGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.App.Config.XrayConfigPath == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "未配置 xray_config_path"})
		return
	}
	data, err := os.ReadFile(h.App.Config.XrayConfigPath)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "读取失败: " + err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"content": string(data)})
}

func (h *Handlers) HandleConfigSave(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(405)
		json.NewEncoder(w).Encode(map[string]string{"error": "方法不允许"})
		return
	}
	var req struct {
		Content string `json:"content"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Content == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "内容不能为空"})
		return
	}
	backupPath := h.App.Config.XrayConfigPath + ".bak"
	if orig, err := os.ReadFile(h.App.Config.XrayConfigPath); err == nil {
		os.WriteFile(backupPath, orig, 0644)
	}
	if err := os.WriteFile(h.App.Config.XrayConfigPath, []byte(req.Content), 0644); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "写入失败: " + err.Error()})
		return
	}
	log.Printf("[Admin] 配置文件已保存")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handlers) HandleConfigValidate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(405)
		json.NewEncoder(w).Encode(map[string]string{"error": "方法不允许"})
		return
	}
	var req struct {
		Content string `json:"content"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Content == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"valid": false, "message": "配置内容为空"})
		return
	}

	xrayBin := h.App.Config.XrayBinPath
	if xrayBin == "" {
		xrayBin = "xray"
		if _, err := exec.LookPath(xrayBin); err != nil {
			for _, p := range []string{"/usr/local/bin/xray", "/usr/bin/xray"} {
				if _, err := os.Stat(p); err == nil {
					xrayBin = p
					break
				}
			}
		}
	}
	if !strings.HasSuffix(xrayBin, "/xray") && xrayBin != "xray" && !strings.HasSuffix(xrayBin, "\\xray.exe") && xrayBin != "xray.exe" {
		json.NewEncoder(w).Encode(map[string]interface{}{"valid": false, "message": "安全错误: 无效的 xray 执行路径 (" + xrayBin + ")"})
		return
	}

	tmpFile, err := os.CreateTemp("", "xray-test-*.json")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"valid": false, "message": "无法创建临时测试文件: " + err.Error()})
		return
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte(req.Content))
	tmpFile.Close()

	out, err := exec.Command(xrayBin, "-test", "-config", tmpFile.Name()).CombinedOutput()
	result := strings.TrimSpace(string(out))
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"valid": false, "message": result})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"valid": true, "message": result})
}

func (h *Handlers) HandleConfigRestore(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(405)
		json.NewEncoder(w).Encode(map[string]string{"error": "方法不允许"})
		return
	}
	backupPath := h.App.Config.XrayConfigPath + ".bak"
	data, err := os.ReadFile(backupPath)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "备份文件不存在: " + err.Error()})
		return
	}
	if err := os.WriteFile(h.App.Config.XrayConfigPath, data, 0644); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "还原失败: " + err.Error()})
		return
	}
	log.Printf("[Admin] 配置文件已从备份还原")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handlers) HandleCertStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.App.Config.CertPath == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "未配置 cert_path"})
		return
	}
	info, err := telegram.ParseCertificate(h.App.Config.CertPath)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(info)
}

func (h *Handlers) HandleTelegramTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(405)
		json.NewEncoder(w).Encode(map[string]string{"error": "方法不允许"})
		return
	}
	if h.App.TGBot == nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Telegram Bot 未配置"})
		return
	}
	err := h.App.TGBot.SendMessage(h.App.TGBot.ChatID, "Xray Panel 测试消息\n\n"+time.Now().In(h.App.ChinaTZ).Format("2006-01-02 15:04:05"), "")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "发送失败: " + err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handlers) HandleTelegramStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.App.TGBot == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"configured": false})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"configured": true, "chat_id": h.App.TGBot.ChatID})
}

func (h *Handlers) HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	email := r.URL.Query().Get("email")
	if email == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "email 参数必须"})
		return
	}
	links, err := telegram.GenerateSubscribeLinks(&h.App.Config, email)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	var allURIs []string
	for _, l := range links {
		allURIs = append(allURIs, l.URI)
	}
	combinedBase64 := base64.StdEncoding.EncodeToString([]byte(strings.Join(allURIs, "\n")))
	json.NewEncoder(w).Encode(map[string]interface{}{"links": links, "combined_base64": combinedBase64})
}

// serveFrontendFile 从 embed.FS 读取前端文件（需要由 server 层注入）
// 这里用 placeholder，实际由 server 层处理
func (h *Handlers) serveFrontendFile(w http.ResponseWriter, path string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("CDN-Cache-Control", "no-store")
	content, err := h.FrontendFS.ReadFile(path)
	if err != nil {
		http.Error(w, "Error reading file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(content)
}


