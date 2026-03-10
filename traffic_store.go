package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// TrafficRecord stores per-user traffic for a period (uplink and downlink bytes)
type TrafficRecord struct {
	Uplink   int64 `json:"uplink"`
	Downlink int64 `json:"downlink"`
}

// TrafficStoreData is the persisted data structure serialized to traffic_data.json
type TrafficStoreData struct {
	// Monthly: map[YYYY-MM]map[email]TrafficRecord (archives past traffic by month)
	Monthly map[string]map[string]*TrafficRecord `json:"monthly"`
	// LastSnapshot: the last seen Xray traffic values (current session)
	LastSnapshot map[string]*TrafficRecord `json:"last_snapshot"`
	// LastUptime: last known Xray uptime to detect service restarts
	LastUptime uint32 `json:"last_uptime"`
	// SavedAt: timestamp of the last successful save
	SavedAt string `json:"saved_at"`
}

// TrafficStore handles traffic data persistence in memory and to disk,
// ensuring thread-safe operations during concurrent API requests.
type TrafficStore struct {
	mu       sync.RWMutex
	data     TrafficStoreData
	filePath string
	dirty    bool
}

// NewTrafficStore creates a new TrafficStore instance and loads existing data from disk (if any).
func NewTrafficStore(filePath string) *TrafficStore {
	ts := &TrafficStore{
		filePath: filePath,
		data: TrafficStoreData{
			Monthly:      make(map[string]map[string]*TrafficRecord),
			LastSnapshot: make(map[string]*TrafficRecord),
		},
	}
	ts.load()
	return ts
}

// load reads traffic data from the JSON file and unmarshals it into memory.
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
	if stored.LastSnapshot == nil {
		stored.LastSnapshot = make(map[string]*TrafficRecord)
	}
	ts.data = stored
	log.Printf("[TrafficStore] 已加载历史数据 (保存于 %s)", stored.SavedAt)
}

// save serializes the in-memory traffic data to a JSON file.
func (ts *TrafficStore) save() {
	ts.mu.RLock()
	ts.data.SavedAt = time.Now().In(chinaTZ).Format("2006-01-02 15:04:05")
	data, err := json.MarshalIndent(ts.data, "", "  ")
	ts.mu.RUnlock()
	if err != nil {
		log.Printf("[TrafficStore] 序列化数据失败: %v", err)
		return
	}
	if err := os.WriteFile(ts.filePath, data, 0644); err != nil {
		log.Printf("[TrafficStore] 保存数据失败: %v", err)
		return
	}
}

// Update processes current stats from the Xray Stats API.
// Xray traffic counters reset to 0 upon restart. This function compares
// the current uptime with LastUptime to detect restarts, and archives the
// LastSnapshot into Monthly records when a restart occurs.
func (ts *TrafficStore) Update(users []UserTraffic, uptime uint32) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	month := time.Now().In(chinaTZ).Format("2006-01")

	// Detect Xray restart: uptime decreased or went to 0
	restarted := uptime < ts.data.LastUptime && ts.data.LastUptime > 0

	if restarted {
		log.Printf("[TrafficStore] 检测到 Xray 重启 (uptime: %d -> %d)，保存上次快照到月度数据", ts.data.LastUptime, uptime)
		// Archive last snapshot into monthly data
		if ts.data.LastSnapshot != nil {
			if ts.data.Monthly[month] == nil {
				ts.data.Monthly[month] = make(map[string]*TrafficRecord)
			}
			for email, snap := range ts.data.LastSnapshot {
				if _, ok := ts.data.Monthly[month][email]; !ok {
					ts.data.Monthly[month][email] = &TrafficRecord{}
				}
				ts.data.Monthly[month][email].Uplink += snap.Uplink
				ts.data.Monthly[month][email].Downlink += snap.Downlink
			}
		}
		// Clear snapshot since Xray restarted
		ts.data.LastSnapshot = make(map[string]*TrafficRecord)
	}

	// Update snapshot with current values
	for _, u := range users {
		ts.data.LastSnapshot[u.Email] = &TrafficRecord{
			Uplink:   u.Uplink,
			Downlink: u.Downlink,
		}
	}

	ts.data.LastUptime = uptime
	ts.dirty = true
}

// GetHistoricalTraffic returns total accumulated traffic per user
// (sum of all monthly archived data + current live snapshot).
func (ts *TrafficStore) GetHistoricalTraffic() map[string]*TrafficRecord {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	result := make(map[string]*TrafficRecord)

	// Sum all monthly data
	for _, monthData := range ts.data.Monthly {
		for email, rec := range monthData {
			if result[email] == nil {
				result[email] = &TrafficRecord{}
			}
			result[email].Uplink += rec.Uplink
			result[email].Downlink += rec.Downlink
		}
	}

	// Add current snapshot
	for email, snap := range ts.data.LastSnapshot {
		if result[email] == nil {
			result[email] = &TrafficRecord{}
		}
		result[email].Uplink += snap.Uplink
		result[email].Downlink += snap.Downlink
	}

	return result
}

// StartAutoSave periodically checks the dirty flag and saves data to disk if changes occurred.
func (ts *TrafficStore) StartAutoSave(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			ts.mu.RLock()
			needSave := ts.dirty
			ts.mu.RUnlock()
			if needSave {
				ts.save()
				ts.mu.Lock()
				ts.dirty = false
				ts.mu.Unlock()
			}
		}
	}()
}

// ResetUser completely removes a specific user's traffic data (both current and historical)
// from the traffic store. Usually called when an administrator resets user traffic.
func (ts *TrafficStore) ResetUser(email string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	delete(ts.data.LastSnapshot, email)
	for _, monthData := range ts.data.Monthly {
		delete(monthData, email)
	}
	ts.dirty = true
}

// UserHistoryItem is the API response item
type UserHistoryItem struct {
	Email        string `json:"email"`
	HistUplink   int64  `json:"hist_uplink"`
	HistDownlink int64  `json:"hist_downlink"`
	HistTotal    int64  `json:"hist_total"`
}

// handleTrafficHistory API: GET /api/traffic-history
// Returns the total accumulated traffic history for all users.
func handleTrafficHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if trafficStore == nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "store not initialized"})
		return
	}

	hist := trafficStore.GetHistoricalTraffic()
	items := make([]UserHistoryItem, 0, len(hist))
	for email, rec := range hist {
		items = append(items, UserHistoryItem{
			Email:        email,
			HistUplink:   rec.Uplink,
			HistDownlink: rec.Downlink,
			HistTotal:    rec.Uplink + rec.Downlink,
		})
	}
	json.NewEncoder(w).Encode(items)
}
