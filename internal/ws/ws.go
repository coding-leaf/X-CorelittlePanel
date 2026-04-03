package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"xray-panel/internal/types"
)

// Hub 管理活跃的 WebSocket 客户端并广播消息
type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]bool
}

// NewHub 创建新的 Hub 实例
func NewHub() *Hub {
	return &Hub{clients: make(map[*websocket.Conn]bool)}
}

func (h *Hub) Add(c *websocket.Conn) {
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()
}

func (h *Hub) Remove(c *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

func (h *Hub) Broadcast(msg types.WSMessage) {
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
			go h.Remove(c)
		}
	}
}

func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// NewUpgrader 创建 WebSocket Upgrader
// WS 连接已经通过 cookie 认证中间件保护，这里只需要宽松的 Origin 检查
// 避免因反向代理(Cloudflare/Nginx)改写 Origin 导致连接被拒绝
func NewUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // 认证由 requirePublicOrAuthAPI 中间件保证
		},
		HandshakeTimeout: 10 * time.Second,
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
	}
}

// SpeedTracker 实时速率追踪器
type SpeedTracker struct {
	mu            sync.Mutex
	prevTotalUp   int64
	prevTotalDown int64
	prevTime      time.Time
	Current       types.SpeedData
}

// NewSpeedTracker 创建速率追踪器
func NewSpeedTracker() *SpeedTracker {
	return &SpeedTracker{}
}

// CalcSpeed 根据流量差值计算实时速率
func (st *SpeedTracker) CalcSpeed(totalUp, totalDown int64) types.SpeedData {
	st.mu.Lock()
	defer st.mu.Unlock()

	now := time.Now()
	sd := types.SpeedData{}

	if !st.prevTime.IsZero() {
		elapsed := now.Sub(st.prevTime).Seconds()
		if elapsed > 0 {
			sd.UploadSpeed = float64(totalUp-st.prevTotalUp) / elapsed
			sd.DownloadSpeed = float64(totalDown-st.prevTotalDown) / elapsed
			if sd.UploadSpeed < 0 {
				sd.UploadSpeed = 0
			}
			if sd.DownloadSpeed < 0 {
				sd.DownloadSpeed = 0
			}
			sd.UploadStr = FormatSpeed(sd.UploadSpeed)
			sd.DownloadStr = FormatSpeed(sd.DownloadSpeed)
		}
	}

	st.prevTotalUp = totalUp
	st.prevTotalDown = totalDown
	st.prevTime = now
	st.Current = sd
	return sd
}

// GetCurrent 获取当前速率
func (st *SpeedTracker) GetCurrent() types.SpeedData {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.Current
}

// FormatSpeed 格式化速率
func FormatSpeed(bytesPerSec float64) string {
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
		return strconv.FormatFloat(f, 'f', 2, 64)
	} else if f < 100 {
		return strconv.FormatFloat(f, 'f', 1, 64)
	}
	return strconv.FormatFloat(f, 'f', 0, 64)
}

// HandleWS 升级 HTTP 连接为 WebSocket
func HandleWS(hub *Hub, upgrader websocket.Upgrader, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	hub.Add(conn)
	log.Printf("WebSocket client connected (%d total)", hub.Count())

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
				return
			}
		}
	}()

	go func() {
		defer func() {
			hub.Remove(conn)
			conn.Close()
			log.Printf("WebSocket client disconnected (%d remaining)", hub.Count())
		}()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}()
}
