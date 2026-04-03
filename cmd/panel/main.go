package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	xraypanel "xray-panel"
	"xray-panel/internal/app"
	"xray-panel/internal/config"
	"xray-panel/internal/monitor"
	"xray-panel/internal/server"
	"xray-panel/internal/store"
	"xray-panel/internal/telegram"
	"xray-panel/internal/ws"
	"xray-panel/internal/xray"
)

func main() {
	// 1. 加载配置
	cfg := config.Load()

	// 2. 初始化时区
	chinaTZ, _ := time.LoadLocation("Asia/Shanghai")
	if chinaTZ == nil {
		chinaTZ = time.FixedZone("CST", 8*3600)
	}

	// 3. 生成随机 auth token
	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)

	// 4. 初始化流量持久化
	trafficDataPath := cfg.TrafficDataPath
	if trafficDataPath == "" {
		trafficDataPath = "traffic_data.json"
	}
	trafficStore := store.NewTrafficStore(trafficDataPath, chinaTZ)
	trafficStore.StartAutoSave(2 * time.Minute)

	// 5. 定期清理与周期检查
	go func() {
		for {
			time.Sleep(1 * time.Hour)
			trafficStore.CleanOldDaily(90)
			trafficStore.CheckAndResetCycles()
		}
	}()

	// 6. 构建 App 依赖容器
	xrayClient := xray.NewClient(cfg.XrayAPI)
	speedTracker := ws.NewSpeedTracker()
	versionChecker := monitor.NewVersionChecker(chinaTZ)

	a := &app.App{
		Config:         cfg,
		XrayClient:     xrayClient,
		TrafficStore:   trafficStore,
		WSHub:          ws.NewHub(),
		SpeedTracker:   speedTracker,
		VersionChecker: versionChecker,
		AuthToken:      hex.EncodeToString(tokenBytes),
		ChinaTZ:        chinaTZ,
		LoginRecords:   make(map[string]*app.LoginBan),
	}

	// 7. 初始化 Telegram Bot
	var tgBot *telegram.Bot
	if cfg.TelegramToken != "" && cfg.TelegramChatID != "" {
		tgBot = telegram.New(&cfg, xrayClient, trafficStore, chinaTZ, speedTracker.GetCurrent)
		if tgBot != nil {
			tgBot.VersionChecker = versionChecker
			a.TGBot = tgBot
			tgBot.StartPolling()
			tgBot.StartCertChecker()
			log.Printf("[TelegramBot] Bot 已启动")
		}
	} else {
		log.Printf("[TelegramBot] 未配置 Token 或 ChatID，Bot 已禁用")
	}

	// 8. 启动版本定期检查（每6小时），发现新版本通过 Telegram 通知
	versionChecker.StartPeriodicCheck(func(info monitor.VersionInfo) {
		if tgBot != nil {
			tgBot.SendNotification(fmt.Sprintf("⬆️ *Xray 新版本可用*\n\n当前: `%s`\n最新: `%s`\n\n使用 /update 命令更新",
				info.Current, info.Latest))
		}
	})

	// 9. 启动 HTTP 服务器
	log.Printf("Xray Panel 启动中...")
	log.Printf("监听地址: %s", cfg.ListenAddr)
	log.Printf("Xray API: %s", cfg.XrayAPI)

	srv := server.NewServer(a, xraypanel.FrontendFS)
	srv.StartWSBroadcast()
	srv.InitLoginRateLimitCleanup()
	srv.SetupRoutes()
	srv.Start()
}

