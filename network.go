package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// PingTarget defines a remote host structure to be checked for TCP connectivity.
type PingTarget struct {
	Name string `json:"name"` // Display name, e.g. "Google"
	Host string `json:"host"` // Host address with port, e.g. "google.com:443"
}

// PingResult holds the result of a single TCP connectivity check.
type PingResult struct {
	Name    string `json:"name"`       // Target name
	Host    string `json:"host"`       // Target host
	Latency int64  `json:"latency_ms"` // Response time in ms, -1 indicates failure
	Status  string `json:"status"`     // "ok" or detailed error message
}

var defaultTargets = []PingTarget{
	{Name: "Cloudflare", Host: "cloudflare.com:443"},
	{Name: "Google", Host: "google.com:443"},
	{Name: "YouTube", Host: "youtube.com:443"},
	{Name: "GitHub", Host: "github.com:443"},
	{Name: "Apple", Host: "apple.com:443"},
	{Name: "Microsoft", Host: "microsoft.com:443"},
	{Name: "Telegram", Host: "telegram.org:443"},
	{Name: "Netflix", Host: "netflix.com:443"},
	{Name: "Steam", Host: "store.steampowered.com:443"},
	{Name: "Twitter/X", Host: "x.com:443"},
}

// tcpPing attempts to establish a TCP connection to the specified host within the timeout period.
// Returns the duration taken to establish the connection, representing network latency.
func tcpPing(host string, timeout time.Duration) (time.Duration, error) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", host, timeout)
	if err != nil {
		return 0, err
	}
	conn.Close()
	return time.Since(start), nil
}

// handlePing API: POST /api/ping
// Concurrently measures the TCP latency to multiple target servers.
// Uses a default list of common websites unless a custom "targets" JSON array is provided in the request body.
func handlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		w.WriteHeader(405)
		json.NewEncoder(w).Encode(map[string]string{"error": "方法不允许"})
		return
	}

	targets := defaultTargets

	results := make([]PingResult, len(targets))
	var wg sync.WaitGroup

	for i, t := range targets {
		wg.Add(1)
		go func(idx int, target PingTarget) {
			defer wg.Done()
			latency, err := tcpPing(target.Host, 5*time.Second)
			if err != nil {
				results[idx] = PingResult{
					Name:    target.Name,
					Host:    target.Host,
					Latency: -1,
					Status:  fmt.Sprintf("失败: %v", err),
				}
			} else {
				results[idx] = PingResult{
					Name:    target.Name,
					Host:    target.Host,
					Latency: latency.Milliseconds(),
					Status:  "ok",
				}
			}
		}(i, t)
	}

	wg.Wait()
	json.NewEncoder(w).Encode(results)
}
