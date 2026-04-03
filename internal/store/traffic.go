package store

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"

	"xray-panel/internal/types"
)

// TrafficRecord stores per-user traffic for a period
type TrafficRecord struct {
	Uplink   int64 `json:"uplink"`
	Downlink int64 `json:"downlink"`
}

// TrafficStoreData 持久化数据结构
type TrafficStoreData struct {
	Monthly      map[string]map[string]*TrafficRecord `json:"monthly"`
	Daily        map[string]map[string]*TrafficRecord `json:"daily"`
	LastSnapshot map[string]*TrafficRecord            `json:"last_snapshot"`
	UserSettings map[string]*UserSetting              `json:"user_settings,omitempty"`
	LastUptime   uint32                               `json:"last_uptime"`
	SavedAt      string                               `json:"saved_at"`
}

// UserSetting 用户设置
type UserSetting struct {
	ResetDay       int    `json:"reset_day"`
	LastResetMonth string `json:"last_reset_month"`
}

// TrafficStore 流量持久化管理器
type TrafficStore struct {
	mu       sync.RWMutex
	data     TrafficStoreData
	filePath string
	dirty    bool
	chinaTZ  *time.Location
}

// NewTrafficStore 创建实例并从磁盘加载
func NewTrafficStore(filePath string, tz *time.Location) *TrafficStore {
	ts := &TrafficStore{
		filePath: filePath,
		chinaTZ:  tz,
		data: TrafficStoreData{
			Monthly:      make(map[string]map[string]*TrafficRecord),
			Daily:        make(map[string]map[string]*TrafficRecord),
			LastSnapshot: make(map[string]*TrafficRecord),
			UserSettings: make(map[string]*UserSetting),
		},
	}
	ts.load()
	return ts
}

func (ts *TrafficStore) load() {
	data, err := os.ReadFile(ts.filePath)
	if err != nil {
		log.Printf("[TrafficStore] 无历史数据文件，将创建新的")
		return
	}
	var stored TrafficStoreData
	if err := json.Unmarshal(data, &stored); err != nil {
		log.Printf("[TrafficStore] 解析历史数据失败: %v", err)
		return
	}
	if stored.Monthly == nil {
		stored.Monthly = make(map[string]map[string]*TrafficRecord)
	}
	if stored.Daily == nil {
		stored.Daily = make(map[string]map[string]*TrafficRecord)
	}
	if stored.LastSnapshot == nil {
		stored.LastSnapshot = make(map[string]*TrafficRecord)
	}
	if stored.UserSettings == nil {
		stored.UserSettings = make(map[string]*UserSetting)
	}
	ts.data = stored
	log.Printf("[TrafficStore] 已加载历史数据 (保存于 %s)", stored.SavedAt)
}

// Save 持久化到磁盘
func (ts *TrafficStore) Save() {
	ts.mu.Lock()
	ts.data.SavedAt = time.Now().In(ts.chinaTZ).Format("2006-01-02 15:04:05")
	data, err := json.MarshalIndent(ts.data, "", "  ")
	ts.mu.Unlock()
	if err != nil {
		log.Printf("[TrafficStore] 序列化数据失败: %v", err)
		return
	}
	if err := os.WriteFile(ts.filePath, data, 0644); err != nil {
		log.Printf("[TrafficStore] 保存数据失败: %v", err)
		return
	}
}

// Update 处理来自 Xray Stats API 的当前数据
func (ts *TrafficStore) Update(users []types.UserTraffic, uptime uint32) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	now := time.Now().In(ts.chinaTZ)
	month := now.Format("2006-01")
	day := now.Format("2006-01-02")

	restarted := uptime < ts.data.LastUptime && ts.data.LastUptime > 0

	if restarted {
		log.Printf("[TrafficStore] 检测到 Xray 重启 (uptime: %d -> %d)，保存上次快照", ts.data.LastUptime, uptime)
		if ts.data.LastSnapshot != nil {
			if ts.data.Monthly[month] == nil {
				ts.data.Monthly[month] = make(map[string]*TrafficRecord)
			}
			if ts.data.Daily[day] == nil {
				ts.data.Daily[day] = make(map[string]*TrafficRecord)
			}
			for email, snap := range ts.data.LastSnapshot {
				if _, ok := ts.data.Monthly[month][email]; !ok {
					ts.data.Monthly[month][email] = &TrafficRecord{}
				}
				ts.data.Monthly[month][email].Uplink += snap.Uplink
				ts.data.Monthly[month][email].Downlink += snap.Downlink

				if _, ok := ts.data.Daily[day][email]; !ok {
					ts.data.Daily[day][email] = &TrafficRecord{}
				}
				ts.data.Daily[day][email].Uplink += snap.Uplink
				ts.data.Daily[day][email].Downlink += snap.Downlink
			}
		}
		ts.data.LastSnapshot = make(map[string]*TrafficRecord)
	}

	for _, u := range users {
		ts.data.LastSnapshot[u.Email] = &TrafficRecord{
			Uplink:   u.Uplink,
			Downlink: u.Downlink,
		}
	}

	ts.data.LastUptime = uptime
	ts.dirty = true
}

// GetHistoricalTraffic 返回所有用户的累计历史流量
func (ts *TrafficStore) GetHistoricalTraffic() map[string]*TrafficRecord {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	result := make(map[string]*TrafficRecord)
	for _, monthData := range ts.data.Monthly {
		for email, rec := range monthData {
			if result[email] == nil {
				result[email] = &TrafficRecord{}
			}
			result[email].Uplink += rec.Uplink
			result[email].Downlink += rec.Downlink
		}
	}
	for email, snap := range ts.data.LastSnapshot {
		if result[email] == nil {
			result[email] = &TrafficRecord{}
		}
		result[email].Uplink += snap.Uplink
		result[email].Downlink += snap.Downlink
	}
	return result
}

// StartAutoSave 定时保存
func (ts *TrafficStore) StartAutoSave(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			ts.mu.RLock()
			needSave := ts.dirty
			ts.mu.RUnlock()
			if needSave {
				ts.Save()
				ts.mu.Lock()
				ts.dirty = false
				ts.mu.Unlock()
			}
		}
	}()
}

// ResetUser 清除指定用户的全部流量记录
func (ts *TrafficStore) ResetUser(email string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	delete(ts.data.LastSnapshot, email)
	for _, monthData := range ts.data.Monthly {
		delete(monthData, email)
	}
	for _, dayData := range ts.data.Daily {
		delete(dayData, email)
	}
	ts.dirty = true
}

// GetUserSetting 获取用户设置
func (ts *TrafficStore) GetUserSetting(email string) *UserSetting {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	if set, ok := ts.data.UserSettings[email]; ok {
		return &UserSetting{ResetDay: set.ResetDay, LastResetMonth: set.LastResetMonth}
	}
	return &UserSetting{ResetDay: 0, LastResetMonth: ""}
}

// SetUserSetting 设置用户周期
func (ts *TrafficStore) SetUserSetting(email string, day int) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if ts.data.UserSettings == nil {
		ts.data.UserSettings = make(map[string]*UserSetting)
	}
	if set, ok := ts.data.UserSettings[email]; ok {
		set.ResetDay = day
	} else {
		ts.data.UserSettings[email] = &UserSetting{ResetDay: day}
	}
	ts.dirty = true
}

// CheckAndResetCycles 检查并执行月度周期重置
func (ts *TrafficStore) CheckAndResetCycles() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if ts.data.UserSettings == nil || len(ts.data.UserSettings) == 0 {
		return
	}

	now := time.Now().In(ts.chinaTZ)
	currentDay := now.Day()
	currentMonth := now.Format("2006-01")
	lastDayOfMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, ts.chinaTZ).Day()

	needSave := false
	for email, set := range ts.data.UserSettings {
		if set.ResetDay <= 0 {
			continue
		}
		if set.LastResetMonth == currentMonth {
			continue
		}
		shouldReset := false
		if currentDay == set.ResetDay {
			shouldReset = true
		} else if currentDay == lastDayOfMonth && set.ResetDay > lastDayOfMonth {
			shouldReset = true
		}
		if shouldReset {
			delete(ts.data.LastSnapshot, email)
			for _, monthData := range ts.data.Monthly {
				delete(monthData, email)
			}
			for _, dayData := range ts.data.Daily {
				delete(dayData, email)
			}
			set.LastResetMonth = currentMonth
			log.Printf("[Cycle] 触发用户 %s 的周期流量重置 (设定日: %d，今日: %d)", email, set.ResetDay, currentDay)
			needSave = true
		}
	}
	if needSave {
		ts.dirty = true
	}
}

// GetDailyTraffic 返回每日流量
func (ts *TrafficStore) GetDailyTraffic(email string, days int) []types.DailyTrafficItem {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	now := time.Now().In(ts.chinaTZ)
	var items []types.DailyTrafficItem

	for i := days - 1; i >= 0; i-- {
		day := now.AddDate(0, 0, -i).Format("2006-01-02")
		dayData := ts.data.Daily[day]

		if email != "" {
			var up, down int64
			if dayData != nil {
				if rec, ok := dayData[email]; ok {
					up = rec.Uplink
					down = rec.Downlink
				}
			}
			if i == 0 {
				if snap, ok := ts.data.LastSnapshot[email]; ok {
					up += snap.Uplink
					down += snap.Downlink
				}
			}
			items = append(items, types.DailyTrafficItem{Date: day, Uplink: up, Downlink: down, Total: up + down})
		} else {
			var up, down int64
			if dayData != nil {
				for _, rec := range dayData {
					up += rec.Uplink
					down += rec.Downlink
				}
			}
			if i == 0 {
				for _, snap := range ts.data.LastSnapshot {
					up += snap.Uplink
					down += snap.Downlink
				}
			}
			items = append(items, types.DailyTrafficItem{Date: day, Uplink: up, Downlink: down, Total: up + down})
		}
	}
	return items
}

// CleanOldDaily 清理超过 maxDays 的每日记录
func (ts *TrafficStore) CleanOldDaily(maxDays int) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	cutoff := time.Now().In(ts.chinaTZ).AddDate(0, 0, -maxDays).Format("2006-01-02")
	for day := range ts.data.Daily {
		if day < cutoff {
			delete(ts.data.Daily, day)
			ts.dirty = true
		}
	}
}
