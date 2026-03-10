package main

import "net/http"

// setupRoutes 配置并注册所有 HTTP 路由
func setupRoutes() {
	// ==========================================
	// 1. 公共页面与基础接口
	// ==========================================
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/api/logout", requireAuthAPI(handleLogout))

	// 首页 Dashboard (监控面板) - 受 PublicDashboard 控制
	http.HandleFunc("/", requirePublicOrAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(indexHTML))
	}))

	// 基础日志展示页面 (需登录)
	http.HandleFunc("/admin/logs", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(logsHTML))
	}))

	// WebSocket 端点 (实时推送数据) - 受 PublicDashboard 控制
	http.HandleFunc("/ws", requirePublicOrAuthAPI(handleWS))

	// ==========================================
	// 2. 前台数据 API (由首页 JS 调用，需带有 X-Panel头)
	// ==========================================
	// 以下接口受 PublicDashboard 控制
	http.HandleFunc("/api/stats", requirePanelRequest(requirePublicOrAuthAPI(handleAPI)))
	http.HandleFunc("/api/sysinfo", requirePanelRequest(requirePublicOrAuthAPI(handleSysInfo)))
	http.HandleFunc("/api/traffic-history", requirePanelRequest(requirePublicOrAuthAPI(handleTrafficHistory)))

	// 以下接口需进一步携带 auth Cookie
	http.HandleFunc("/api/reset", requirePanelRequest(requireAuthAPI(handleResetUser)))
	http.HandleFunc("/api/speed", requirePanelRequest(requireAuthAPI(handleSpeed)))
	http.HandleFunc("/api/ping", requirePanelRequest(requireAuthAPI(handlePing)))
	http.HandleFunc("/api/logs", requirePanelRequest(requireAuthAPI(handleAccessLogs)))
	http.HandleFunc("/api/errors", requirePanelRequest(requireAuthAPI(handleErrorLogs)))
	http.HandleFunc("/api/users", requirePanelRequest(requireAuthAPI(handleUserList)))
	http.HandleFunc("/api/clear-logs", requirePanelRequest(requireAuthAPI(handleClearLogs)))

	// ==========================================
	// 3. 高级管理端点 /admin/* (由 Admin 面板 JS 调用，受 Cloudflare Access 保护)
	// ==========================================

	// Admin 面板页面入口 (需登录，未登录会跳转到 /login)
	http.HandleFunc("/admin/panel", requireAuth(handleAdminPage))

	// 3.1 用户管理
	http.HandleFunc("/admin/api/users", requirePanelRequest(requireAuthAPI(handleAdminUsers)))
	http.HandleFunc("/admin/api/users/add", requirePanelRequest(requireAuthAPI(handleAdminAddUser)))
	http.HandleFunc("/admin/api/users/delete", requirePanelRequest(requireAuthAPI(handleAdminDeleteUser)))

	// 3.2 Xray 进程控制
	http.HandleFunc("/admin/api/xray/restart", requirePanelRequest(requireAuthAPI(handleXrayRestart)))
	http.HandleFunc("/admin/api/xray/reload", requirePanelRequest(requireAuthAPI(handleXrayReload)))
	http.HandleFunc("/admin/api/xray/status", requirePanelRequest(requireAuthAPI(handleXrayStatus)))

	// 3.3 配置文件编辑
	http.HandleFunc("/admin/api/xconf", requirePanelRequest(requireAuthAPI(handleConfigGet)))
	http.HandleFunc("/admin/api/xconf/save", requirePanelRequest(requireAuthAPI(handleConfigSave)))
	http.HandleFunc("/admin/api/xconf/validate", requirePanelRequest(requireAuthAPI(handleConfigValidate)))
	http.HandleFunc("/admin/api/xconf/restore", requirePanelRequest(requireAuthAPI(handleConfigRestore)))

	// 3.4 服务器与证书状态
	http.HandleFunc("/admin/api/cert", requirePanelRequest(requireAuthAPI(handleCertStatus)))

	// 3.5 Telegram 控制
	http.HandleFunc("/admin/api/telegram/test", requirePanelRequest(requireAuthAPI(handleTelegramTest)))
	http.HandleFunc("/admin/api/telegram/status", requirePanelRequest(requireAuthAPI(handleTelegramStatus)))

	// 3.6 订阅链接生成
	http.HandleFunc("/admin/api/subscribe", requirePanelRequest(requireAuthAPI(handleSubscribe)))
}
