package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/acme/autocert"

	pb "xray-panel/proto"
	"xray-panel/internal/app"
	"xray-panel/internal/handler"
	"xray-panel/internal/monitor"
	"xray-panel/internal/types"
	"xray-panel/internal/xray"
)

// Server HTTP 服务器
type Server struct {
	App        *app.App
	Handlers   *handler.Handlers
	FrontendFS embed.FS
}

// NewServer 创建服务器实例
func NewServer(a *app.App, frontendFS embed.FS) *Server {
	return &Server{
		App:        a,
		Handlers:   handler.NewHandlers(a, frontendFS),
		FrontendFS: frontendFS,
	}
}

// ===== 中间件 =====

func (s *Server) isAuthed(r *http.Request) bool {
	if s.App.Config.Password == "" {
		return true
	}
	cookie, err := r.Cookie("auth")
	return err == nil && cookie.Value == s.App.AuthToken
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.isAuthed(r) {
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
			w.Header().Set("CDN-Cache-Control", "no-store")
			http.Redirect(w, r, "/login?redirect="+r.URL.Path, http.StatusFound)
			return
		}
		next(w, r)
	}
}

func (s *Server) requireAuthAPI(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.isAuthed(r) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(401)
			json.NewEncoder(w).Encode(map[string]string{"error": "未登录"})
			return
		}
		next(w, r)
	}
}

func (s *Server) requirePublicOrAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.App.Config.PublicDashboard && !s.isAuthed(r) {
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
			w.Header().Set("CDN-Cache-Control", "no-store")
			http.Redirect(w, r, "/login?redirect="+r.URL.Path, http.StatusFound)
			return
		}
		next(w, r)
	}
}

func (s *Server) requirePublicOrAuthAPI(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.App.Config.PublicDashboard && !s.isAuthed(r) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(401)
			json.NewEncoder(w).Encode(map[string]string{"error": "未登录或面板未公开"})
			return
		}
		next(w, r)
	}
}

func requirePanelRequest(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Panel") != "1" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(403)
			json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
			return
		}
		next(w, r)
	}
}

func securityWrapper(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self' 'unsafe-inline' 'unsafe-eval' data: blob: ws: wss: https://cdn.jsdelivr.net https://cdn.staticfile.net https://fonts.googleapis.com https://fonts.gstatic.com;")
		next.ServeHTTP(w, r)
	})
}

// serveFrontendFile 从 embed.FS 读取前端文件
func (s *Server) serveFrontendFile(w http.ResponseWriter, path string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("CDN-Cache-Control", "no-store")
	content, err := s.FrontendFS.ReadFile(path)
	if err != nil {
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}
	w.Write(content)
}

// ===== 路由注册 =====

func (s *Server) SetupRoutes() {
	h := s.Handlers

	// 静态资源
	frontendSub, err := fs.Sub(s.FrontendFS, "frontend")
	if err == nil {
		http.Handle("/css/", http.FileServer(http.FS(frontendSub)))
		http.Handle("/js/", http.FileServer(http.FS(frontendSub)))
	}

	// 公共页面
	http.HandleFunc("/login", h.HandleLogin)
	http.HandleFunc("/api/logout", s.requireAuthAPI(h.HandleLogout))

	// 首页
	http.HandleFunc("/", s.requirePublicOrAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		s.serveFrontendFile(w, "frontend/index.html")
	}))

	// 日志页面
	http.HandleFunc("/admin/logs", s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		s.serveFrontendFile(w, "frontend/logs.html")
	}))

	// WebSocket
	http.HandleFunc("/ws", s.requirePublicOrAuthAPI(h.HandleWS))

	// 前台 API
	http.HandleFunc("/api/stats", requirePanelRequest(s.requirePublicOrAuthAPI(h.HandleAPI)))
	http.HandleFunc("/api/sysinfo", requirePanelRequest(s.requirePublicOrAuthAPI(h.HandleSysInfo)))
	http.HandleFunc("/api/traffic-history", requirePanelRequest(s.requirePublicOrAuthAPI(h.HandleTrafficHistory)))
	http.HandleFunc("/api/traffic-daily", requirePanelRequest(s.requirePublicOrAuthAPI(h.HandleDailyTraffic)))

	http.HandleFunc("/api/reset", requirePanelRequest(s.requireAuthAPI(h.HandleResetUser)))
	http.HandleFunc("/api/speed", requirePanelRequest(s.requireAuthAPI(h.HandleSpeed)))
	http.HandleFunc("/api/ping", requirePanelRequest(s.requireAuthAPI(h.HandlePing)))
	http.HandleFunc("/api/logs", requirePanelRequest(s.requireAuthAPI(h.HandleAccessLogs)))
	http.HandleFunc("/api/errors", requirePanelRequest(s.requireAuthAPI(h.HandleErrorLogs)))
	http.HandleFunc("/api/users", requirePanelRequest(s.requireAuthAPI(h.HandleUserList)))
	http.HandleFunc("/api/clear-logs", requirePanelRequest(s.requireAuthAPI(h.HandleClearLogs)))

	http.HandleFunc("/api/cycle", requirePanelRequest(s.requireAuthAPI(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			h.HandleSetCycle(w, r)
		} else {
			h.HandleGetCycle(w, r)
		}
	})))

	// Admin 页面与 API
	http.HandleFunc("/admin/panel", s.requireAuth(h.HandleAdminPage))
	http.HandleFunc("/admin/api/users", requirePanelRequest(s.requireAuthAPI(h.HandleAdminUsers)))
	http.HandleFunc("/admin/api/users/add", requirePanelRequest(s.requireAuthAPI(h.HandleAdminAddUser)))
	http.HandleFunc("/admin/api/users/delete", requirePanelRequest(s.requireAuthAPI(h.HandleAdminDeleteUser)))
	http.HandleFunc("/admin/api/xray/restart", requirePanelRequest(s.requireAuthAPI(h.HandleXrayRestart)))
	http.HandleFunc("/admin/api/xray/reload", requirePanelRequest(s.requireAuthAPI(h.HandleXrayReload)))
	http.HandleFunc("/admin/api/xray/status", requirePanelRequest(s.requireAuthAPI(h.HandleXrayStatus)))
	http.HandleFunc("/admin/api/xconf", requirePanelRequest(s.requireAuthAPI(h.HandleConfigGet)))
	http.HandleFunc("/admin/api/xconf/save", requirePanelRequest(s.requireAuthAPI(h.HandleConfigSave)))
	http.HandleFunc("/admin/api/xconf/validate", requirePanelRequest(s.requireAuthAPI(h.HandleConfigValidate)))
	http.HandleFunc("/admin/api/xconf/restore", requirePanelRequest(s.requireAuthAPI(h.HandleConfigRestore)))
	http.HandleFunc("/admin/api/cert", requirePanelRequest(s.requireAuthAPI(h.HandleCertStatus)))
	http.HandleFunc("/admin/api/telegram/test", requirePanelRequest(s.requireAuthAPI(h.HandleTelegramTest)))
	http.HandleFunc("/admin/api/telegram/status", requirePanelRequest(s.requireAuthAPI(h.HandleTelegramStatus)))
	http.HandleFunc("/admin/api/subscribe", requirePanelRequest(s.requireAuthAPI(h.HandleSubscribe)))
	http.HandleFunc("/admin/api/version", requirePanelRequest(s.requireAuthAPI(h.HandleVersionCheck)))
	http.HandleFunc("/admin/api/xray-update", requirePanelRequest(s.requireAuthAPI(h.HandleXrayUpdate)))
}

// ===== WS 广播 =====

func (s *Server) StartWSBroadcast() {
	// 速率计算独立 goroutine —— 不受 WS 连接状态影响
	go func() {
		for {
			time.Sleep(10 * time.Second)
			conn, err := s.App.XrayClient.GetStatsConn()
			if err != nil {
				continue
			}
			client := pb.NewStatsServiceClient(conn)
			stats, err := s.App.XrayClient.QueryAllStats(client)
			conn.Close()
			if err != nil {
				continue
			}
			users, inbounds, outbounds := xray.ParseStats(stats)

			// 单独连接获取 SysStats
			var sysStats *types.SysStats
			conn2, err := s.App.XrayClient.GetStatsConn()
			if err == nil {
				client2 := pb.NewStatsServiceClient(conn2)
				sysStats, _ = s.App.XrayClient.GetSysStats(client2)
				conn2.Close()
			}

			s.App.RecordHistory(users)
			if s.App.TrafficStore != nil && sysStats != nil {
				s.App.TrafficStore.Update(users, sysStats.Uptime)
			}

			// 计算速率（始终运行）
			var totalUp, totalDown int64
			for _, u := range users {
				totalUp += u.Uplink
				totalDown += u.Downlink
			}
			speed := s.App.SpeedTracker.CalcSpeed(totalUp, totalDown)

			// 有 WS 客户端时才广播
			if s.App.WSHub.Count() > 0 {
				hist := s.App.GetHistoryCopy()
				data := types.DashboardData{
					Users: users, Inbounds: inbounds, Outbounds: outbounds,
					SysStats: sysStats, History: hist,
					UpdatedAt: time.Now().In(s.App.ChinaTZ).Format("2006-01-02 15:04:05"),
				}
				s.App.WSHub.Broadcast(types.WSMessage{Type: "stats", Data: data})
				s.App.WSHub.Broadcast(types.WSMessage{Type: "speed", Data: speed})
			}
		}
	}()

	// SysInfo every 30s
	go func() {
		for {
			time.Sleep(30 * time.Second)
			if s.App.WSHub.Count() == 0 {
				continue
			}
			info := monitor.CollectSysInfo()
			s.App.WSHub.Broadcast(types.WSMessage{Type: "sysinfo", Data: info})
		}
	}()
}

// ===== 登录防爆破清理 =====

func (s *Server) InitLoginRateLimitCleanup() {
	go func() {
		for {
			time.Sleep(30 * time.Minute)
			s.App.LoginMutex.Lock()
			now := time.Now()
			for ip, rec := range s.App.LoginRecords {
				if now.After(rec.BlockedTill) && now.Sub(rec.LastAttempt) > time.Hour {
					delete(s.App.LoginRecords, ip)
				}
			}
			s.App.LoginMutex.Unlock()
		}
	}()
}

// ===== HTTP 服务器启动 =====

func (s *Server) Start() {
	addr := s.App.Config.ListenAddr
	certFile := s.App.Config.TLSCertFile
	keyFile := s.App.Config.TLSKeyFile
	domain := s.App.Config.Domain

	httpHandler := securityWrapper(http.DefaultServeMux)

	if certFile != "" && keyFile != "" {
		log.Printf("[HTTPS] 检测到证书文件，启动原生 TLS 服务: %s", addr)
		log.Fatal(http.ListenAndServeTLS(addr, certFile, keyFile, httpHandler))
		return
	}

	if domain != "" {
		log.Printf("[HTTPS] 启用 Let's Encrypt 自动证书服务: %s, 域名: %s", addr, domain)
		m := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(domain),
			Cache:      autocert.DirCache("certs"),
		}
		srv := &http.Server{
			Addr: addr, Handler: httpHandler, TLSConfig: m.TLSConfig(),
		}
		log.Fatal(srv.ListenAndServeTLS("", ""))
		return
	}

	if strings.HasPrefix(addr, "127.0.0.1:") || strings.HasPrefix(addr, "localhost:") {
		log.Printf("[HTTP] 检测到仅监听本地环回地址，降级启动普通 HTTP: %s", addr)
		log.Fatal(http.ListenAndServe(addr, httpHandler))
		return
	}

	log.Printf("[HTTPS] 生成自签名证书兜底: %s", addr)
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("无法生成私钥: %v", err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"Xray Panel SelfSigned"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		log.Fatalf("无法生成自签名证书: %v", err)
	}
	tlsCert := tls.Certificate{Certificate: [][]byte{certDER}, PrivateKey: priv}
	srv := &http.Server{
		Addr: addr, Handler: httpHandler,
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{tlsCert}},
	}
	log.Fatal(srv.ListenAndServeTLS("", ""))
}
