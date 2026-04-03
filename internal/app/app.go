package app

import (
	"sync"
	"time"

	"xray-panel/internal/monitor"
	"xray-panel/internal/store"
	"xray-panel/internal/telegram"
	"xray-panel/internal/types"
	"xray-panel/internal/ws"
	"xray-panel/internal/xray"
)

// App 是全局依赖注入容器，聚合所有核心组件
type App struct {
	Config       types.Config
	XrayClient   *xray.Client
	TrafficStore *store.TrafficStore
	WSHub        *ws.Hub
	SpeedTracker   *ws.SpeedTracker
	VersionChecker *monitor.VersionChecker
	TGBot        *telegram.Bot
	AuthToken    string
	ChinaTZ      *time.Location

	// 流量图表历史
	TrafficHist []types.TrafficHistory
	HistMutex   sync.RWMutex
	LastUplink  int64
	LastDownlink int64

	// 登录防爆破
	LoginRecords map[string]*LoginBan
	LoginMutex   sync.Mutex
}

// LoginBan 登录封锁记录
type LoginBan struct {
	Fails       int
	BlockedTill time.Time
	LastAttempt time.Time
}

// RecordHistory 记录流量历史快照
func (a *App) RecordHistory(users []types.UserTraffic) {
	var totalUp, totalDown int64
	for _, u := range users {
		totalUp += u.Uplink
		totalDown += u.Downlink
	}

	a.HistMutex.Lock()
	defer a.HistMutex.Unlock()

	a.TrafficHist = append(a.TrafficHist, types.TrafficHistory{
		Time:     time.Now().In(a.ChinaTZ).Format("15:04"),
		Uplink:   totalUp,
		Downlink: totalDown,
	})

	if len(a.TrafficHist) > 60 {
		a.TrafficHist = a.TrafficHist[len(a.TrafficHist)-60:]
	}

	a.LastUplink = totalUp
	a.LastDownlink = totalDown
}

// GetHistoryCopy 返回历史快照的拷贝
func (a *App) GetHistoryCopy() []types.TrafficHistory {
	a.HistMutex.RLock()
	defer a.HistMutex.RUnlock()
	hist := make([]types.TrafficHistory, len(a.TrafficHist))
	copy(hist, a.TrafficHist)
	return hist
}
