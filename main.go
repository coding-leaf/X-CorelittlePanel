package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/acme/autocert"

	pb "xray-panel/proto"
)

var (
	config       Config
	trafficHist  []TrafficHistory
	histMutex    sync.RWMutex
	lastUplink   int64
	lastDownlink int64
	authToken    string
	chinaTZ      *time.Location
	trafficStore *TrafficStore

	// 登录防爆破机制
	loginRecords = make(map[string]*loginBan)
	loginMutex   sync.Mutex
)

type loginBan struct {
	fails       int
	blockedTill time.Time
	lastAttempt time.Time
}

func init() {
	chinaTZ, _ = time.LoadLocation("Asia/Shanghai")
	if chinaTZ == nil {
		chinaTZ = time.FixedZone("CST", 8*3600)
	}
}

// recordHistory 记录流量的历史快照，用于绘制图表
func recordHistory(users []UserTraffic) {
	var totalUp, totalDown int64
	for _, u := range users {
		totalUp += u.Uplink
		totalDown += u.Downlink
	}

	histMutex.Lock()
	defer histMutex.Unlock()

	trafficHist = append(trafficHist, TrafficHistory{
		Time:     time.Now().In(chinaTZ).Format("15:04"),
		Uplink:   totalUp,
		Downlink: totalDown,
	})

	if len(trafficHist) > 60 {
		trafficHist = trafficHist[len(trafficHist)-60:]
	}

	lastUplink = totalUp
	lastDownlink = totalDown
}

// handleAPI 向前端提供综合面板数据
func handleAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("CDN-Cache-Control", "no-store")

	conn, err := getStatsConn()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "无法连接 Xray API: " + err.Error()})
		return
	}
	defer conn.Close()

	client := pb.NewStatsServiceClient(conn)
	stats, err := queryAllStats(client)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "查询失败: " + err.Error()})
		return
	}

	users, inbounds, outbounds := parseStats(stats)
	sysStats, _ := getSysStats(client)

	recordHistory(users)

	if trafficStore != nil && sysStats != nil {
		trafficStore.Update(users, sysStats.Uptime)
	}

	histMutex.RLock()
	hist := make([]TrafficHistory, len(trafficHist))
	copy(hist, trafficHist)
	histMutex.RUnlock()

	data := DashboardData{
		Users:     users,
		Inbounds:  inbounds,
		Outbounds: outbounds,
		SysStats:  sysStats,
		History:   hist,
		UpdatedAt: time.Now().In(chinaTZ).Format("2006-01-02 15:04:05"),
	}
	json.NewEncoder(w).Encode(data)
}

// handleUserList 接口：获取当前 Xray API 中活跃的所有用户列表
func handleUserList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	conn, err := getStatsConn()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "无法连接 Xray API: " + err.Error()})
		return
	}
	defer conn.Close()

	client := pb.NewStatsServiceClient(conn)
	stats, err := queryAllStats(client)
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

// handleResetUser 重置单个用户的流量统计
func handleResetUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	conn, err := getStatsConn()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer conn.Close()

	client := pb.NewStatsServiceClient(conn)
	ctx := context.Background()

	client.GetStats(ctx, &pb.GetStatsRequest{
		Name:   fmt.Sprintf("user>>>%s>>>traffic>>>uplink", req.Email),
		Reset_: true,
	})
	client.GetStats(ctx, &pb.GetStatsRequest{
		Name:   fmt.Sprintf("user>>>%s>>>traffic>>>downlink", req.Email),
		Reset_: true,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// getClientIP 辅助函数：解析请求者的真实 IP，防止 IP 伪造
func getClientIP(r *http.Request) string {
	remoteIP := r.RemoteAddr
	if strings.Contains(remoteIP, ":") {
		remoteIP = strings.Split(remoteIP, ":")[0]
	}

	// 仅当请求直接来自本地代理 (如 Nginx/CF Tunnel 监听本地) 时，才信任请求头
	isLocalOrigin := remoteIP == "127.0.0.1" || remoteIP == "::1"

	if isLocalOrigin {
		ip := r.Header.Get("CF-Connecting-IP")
		if ip == "" {
			ip = r.Header.Get("X-Forwarded-For")
		}
		if ip != "" {
			// 如果有多个 IP (如 a.b.c.d, e.f.g.h)，取第一个
			return strings.TrimSpace(strings.Split(ip, ",")[0])
		}
	}

	// 如果是外部直接访问，严格使用真实的 tcp socket 源 IP
	return remoteIP
}

// handleLogin 登录逻辑与防刷拦截
func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		w.Header().Set("CDN-Cache-Control", "no-store")
		w.Write([]byte(loginHTML))
		return
	}

	ip := getClientIP(r)
	loginMutex.Lock()
	now := time.Now()
	rec, exists := loginRecords[ip]
	if !exists {
		rec = &loginBan{}
		loginRecords[ip] = rec
	}

	if now.Before(rec.blockedTill) {
		loginMutex.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(429)
		json.NewEncoder(w).Encode(map[string]string{"error": "尝试次数过多，请15分钟后再试"})
		return
	}
	loginMutex.Unlock()

	var req struct {
		Password string `json:"password"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Password != config.Password {
		loginMutex.Lock()
		rec.fails++
		rec.lastAttempt = now
		if rec.fails >= 5 {
			rec.blockedTill = now.Add(15 * time.Minute)
		}
		loginMutex.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]string{"error": "密码错误"})
		return
	}

	loginMutex.Lock()
	delete(loginRecords, ip)
	loginMutex.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     "auth",
		Value:    authToken,
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleLogout 退出登录并吊销当前及所有历史凭证
func handleLogout(w http.ResponseWriter, r *http.Request) {
	// 清空本地 Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// 重置全局凭证，导致所有未过期的旧 Cookie 在服务器端失效 (单用户面板专用方案)
	b := make([]byte, 32)
	rand.Read(b)
	authToken = hex.EncodeToString(b)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleSysInfo 接口：返回主机系统信息
func handleSysInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(collectSysInfo())
}

// initLoginRateLimitCleanup 启动定时清理机制，回收过期封锁记录的内存
func initLoginRateLimitCleanup() {
	go func() {
		for {
			time.Sleep(30 * time.Minute)
			loginMutex.Lock()
			now := time.Now()
			for ip, rec := range loginRecords {
				if now.After(rec.blockedTill) && now.Sub(rec.lastAttempt) > time.Hour {
					delete(loginRecords, ip)
				}
			}
			loginMutex.Unlock()
		}
	}()
}

func main() {
	config = loadConfig()
	b := make([]byte, 32)
	rand.Read(b)
	authToken = hex.EncodeToString(b)

	initLoginRateLimitCleanup()

	trafficDataPath := config.TrafficDataPath
	if trafficDataPath == "" {
		trafficDataPath = "traffic_data.json"
	}
	trafficStore = NewTrafficStore(trafficDataPath)
	trafficStore.StartAutoSave(2 * time.Minute)

	log.Printf("Xray Panel 启动中...")
	log.Printf("监听地址: %s", config.ListenAddr)
	log.Printf("Xray API: %s", config.XrayAPI)

	startWSBroadcast()
	initTelegramBot()
	setupRoutes()
	startHTTPServer()
}

// securityWrapper 添加全局安全响应头
func securityWrapper(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self' 'unsafe-inline' 'unsafe-eval' data: blob: ws: wss: https://cdn.jsdelivr.net https://cdn.staticfile.net https://fonts.googleapis.com https://fonts.gstatic.com;")
		next.ServeHTTP(w, r)
	})
}

// startHTTPServer 启动三合一 HTTPS 服务，保护面板和密码防止被嗅探。
// 该方案会自动根据配置动态切换 TLS 证书来源，确保通信安全。
func startHTTPServer() {
	addr := config.ListenAddr
	certFile := config.TLSCertFile
	keyFile := config.TLSKeyFile
	domain := config.Domain

	handler := securityWrapper(http.DefaultServeMux)

	// 方案一：使用用户配置的自备合法证书文件（如通过 acme.sh 或购买的证书）
	if certFile != "" && keyFile != "" {
		log.Printf("[HTTPS] 检测到证书文件，启动原生 TLS 服务: %s", addr)
		log.Fatal(http.ListenAndServeTLS(addr, certFile, keyFile, handler))
		return
	}

	// 方案二：利用 Let's Encrypt 自动申请与发证（需要宿主机具备公网连通性）
	if domain != "" {
		log.Printf("[HTTPS] 启用 Let's Encrypt 自动证书服务: %s, 域名: %s", addr, domain)
		m := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(domain),
			Cache:      autocert.DirCache("certs"),
		}
		server := &http.Server{
			Addr:      addr,
			Handler:   handler,
			TLSConfig: m.TLSConfig(),
		}
		log.Fatal(server.ListenAndServeTLS("", ""))
		return
	}

	// 智能判断：如果只是监听本地地址（如 127.0.0.1 或 localhost），默认降级为 HTTP，
	// 因为通常前面会有 Nginx 或 CF Tunnel 等安全代理负责加密。
	if strings.HasPrefix(addr, "127.0.0.1:") || strings.HasPrefix(addr, "localhost:") {
		log.Printf("[HTTP] 检测到仅监听本地环回地址，认为已存在前置安全代理，降级启动普通 HTTP: %s", addr)
		log.Fatal(http.ListenAndServe(addr, handler))
		return
	}

	// 方案三：退化为内存动态生成自签名 TLS 启动，兜底防止中间人明文嗅探。
	// 即使因为没有证书导致浏览器报不安全，数据传输也依然是加密的。
	log.Printf("[HTTPS] 检测到公网监听且未提供证书，为您动态生成自签名证书强制兜底: %s", addr)
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("无法生成私钥: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Xray Panel SelfSigned"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0), // 10年有效期
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		log.Fatalf("无法生成自签名证书: %v", err)
	}

	tlsCert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  priv,
	}

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{tlsCert},
		},
	}
	log.Fatal(server.ListenAndServeTLS("", ""))
}
