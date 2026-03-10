package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	pb "xray-panel/proto"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// 1. 如果完全没有设置密码，为了兼容性放行（风险自担）。
		if config.Password == "" {
			return true
		}

		origin := r.Header.Get("Origin")
		// 2. 允许非浏览器客户端（如脚本工具）直接发起的无 Origin 头请求
		if origin == "" {
			return true
		}

		// 3. 严格校验：Origin 的 Host 必须与请求目标的 Host（包含端口）完全一致。
		//    这能支持 CDN 域名访问（如 https://www.yezineko.top）
		//    也能支持直接 IP:端口 访问（如 http://1.2.3.4:8880）
		reqHost := r.Host
		return origin == "http://"+reqHost || origin == "https://"+reqHost
	},
}

// WSMessage represents the standard JSON envelope sent to connected WebSocket clients.
// It includes a Type string ("stats", "sysinfo", "speed") and a flexible Data payload.
type WSMessage struct {
	Type string      `json:"type"` // "stats", "sysinfo", "speed"
	Data interface{} `json:"data"`
}

// SpeedData holds real-time bandwidth metrics, including raw bytes/sec and formatted strings.
type SpeedData struct {
	UploadSpeed   float64 `json:"upload_speed"`   // bytes/sec
	DownloadSpeed float64 `json:"download_speed"` // bytes/sec
	UploadStr     string  `json:"upload_str"`     // formatted, e.g., "1.2 MB/s"
	DownloadStr   string  `json:"download_str"`   // formatted, e.g., "5.4 MB/s"
}

// Hub manages active WebSocket clients and broadcasts messages to them.
type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]bool
}

func newHub() *Hub {
	return &Hub{clients: make(map[*websocket.Conn]bool)}
}

func (h *Hub) add(c *websocket.Conn) {
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()
}

func (h *Hub) remove(c *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

func (h *Hub) broadcast(msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		err := c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			c.Close()
			go h.remove(c)
		}
	}
}

func (h *Hub) count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

var wsHub = newHub()

// handleWS API: GET /ws
// Upgrades an HTTP connection to a WebSocket connection, adds the client to the Hub,
// and sets up keep-alive ping mechanisms.
func handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	wsHub.add(conn)
	log.Printf("WebSocket client connected (%d total)", wsHub.count())

	// Set read deadline and pong handler for keepalive
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Ping every 30s to keep connection alive through Cloudflare
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
				return
			}
		}
	}()

	// Keep connection alive, read and discard client messages
	go func() {
		defer func() {
			wsHub.remove(conn)
			conn.Close()
			log.Printf("WebSocket client disconnected (%d remaining)", wsHub.count())
		}()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}()
}

// handleSpeed API: GET /api/speed
// Provides current speed data via a standard HTTP JSON response.
// This acts as a fallback for environments where WebSocket is blocked or unavailable.
func handleSpeed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	speedMu.Lock()
	sd := currentSpeed
	speedMu.Unlock()
	json.NewEncoder(w).Encode(sd)
}

// Speed tracking state
var (
	speedMu       sync.Mutex
	prevTotalUp   int64
	prevTotalDown int64
	prevTime      time.Time
	currentSpeed  SpeedData
)

func formatSpeed(bytesPerSec float64) string {
	if bytesPerSec < 1024 {
		return "0 B/s"
	} else if bytesPerSec < 1024*1024 {
		return formatFloat(bytesPerSec/1024) + " KB/s"
	} else if bytesPerSec < 1024*1024*1024 {
		return formatFloat(bytesPerSec/(1024*1024)) + " MB/s"
	}
	return formatFloat(bytesPerSec/(1024*1024*1024)) + " GB/s"
}

func formatFloat(f float64) string {
	if f < 10 {
		return strconvFormatFloat(f, 2)
	} else if f < 100 {
		return strconvFormatFloat(f, 1)
	}
	return strconvFormatFloat(f, 0)
}

func strconvFormatFloat(f float64, prec int) string {
	return strconv.FormatFloat(f, 'f', prec, 64)
}

// calcSpeed computes the network speed based on the difference in traffic totals over time.
func calcSpeed(totalUp, totalDown int64) SpeedData {
	speedMu.Lock()
	defer speedMu.Unlock()

	now := time.Now()
	sd := SpeedData{}

	if !prevTime.IsZero() {
		elapsed := now.Sub(prevTime).Seconds()
		if elapsed > 0 {
			sd.UploadSpeed = float64(totalUp-prevTotalUp) / elapsed
			sd.DownloadSpeed = float64(totalDown-prevTotalDown) / elapsed
			if sd.UploadSpeed < 0 {
				sd.UploadSpeed = 0
			}
			if sd.DownloadSpeed < 0 {
				sd.DownloadSpeed = 0
			}
			sd.UploadStr = formatSpeed(sd.UploadSpeed)
			sd.DownloadStr = formatSpeed(sd.DownloadSpeed)
		}
	}

	prevTotalUp = totalUp
	prevTotalDown = totalDown
	prevTime = now
	currentSpeed = sd
	return sd
}

// startWSBroadcast initiates two background goroutines that periodically fetch data
// (Xray stats, system information) and broadcast it to all connected WebSocket clients.
func startWSBroadcast() {
	// Stats + Speed every 10s
	go func() {
		for {
			time.Sleep(10 * time.Second)
			if wsHub.count() == 0 {
				continue
			}

			conn, err := getStatsConn()
			if err != nil {
				continue
			}
			client := pb.NewStatsServiceClient(conn)

			stats, err := queryAllStats(client)
			if err != nil {
				conn.Close()
				continue
			}

			users, inbounds, outbounds := parseStats(stats)
			sysStats, _ := getSysStats(client)
			conn.Close()

			recordHistory(users)

			// 更新流量持久化
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

			wsHub.broadcast(WSMessage{Type: "stats", Data: data})

			// Calculate and push speed
			var totalUp, totalDown int64
			for _, u := range users {
				totalUp += u.Uplink
				totalDown += u.Downlink
			}
			speed := calcSpeed(totalUp, totalDown)
			wsHub.broadcast(WSMessage{Type: "speed", Data: speed})
		}
	}()

	// SysInfo every 30s
	go func() {
		for {
			time.Sleep(30 * time.Second)
			if wsHub.count() == 0 {
				continue
			}
			info := collectSysInfo()
			wsHub.broadcast(WSMessage{Type: "sysinfo", Data: info})
		}
	}()
}
