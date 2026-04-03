package monitor

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"

	"xray-panel/internal/types"
)

// runCmd 简单的外部命令执行封装
func runCmd(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// FormatBytes 将字节数转换为人类可读单位
func FormatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// FormatBytesGo 格式化字节数 (int64 版本，用于 Telegram 等)
func FormatBytesGo(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	const k = 1024
	sizes := []string{"B", "KB", "MB", "GB", "TB"}
	i := 0
	b := float64(bytes)
	for b >= k && i < len(sizes)-1 {
		b /= k
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%.0f %s", b, sizes[i])
	}
	return fmt.Sprintf("%.2f %s", b, sizes[i])
}

// FormatUptimeGo 格式化运行时间
func FormatUptimeGo(seconds uint32) string {
	d := seconds / 86400
	h := (seconds % 86400) / 3600
	m := (seconds % 3600) / 60
	if d > 0 {
		return fmt.Sprintf("%d天%d时", d, h)
	}
	if h > 0 {
		return fmt.Sprintf("%d时%d分", h, m)
	}
	return fmt.Sprintf("%d分", m)
}

// ===== 网络检测 =====

// DefaultTargets 默认检测目标
var DefaultTargets = []types.PingTarget{
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

// TCPPing 执行 TCP 连通性测试
func TCPPing(host string, timeout time.Duration) (time.Duration, error) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", host, timeout)
	if err != nil {
		return 0, err
	}
	conn.Close()
	return time.Since(start), nil
}

// RunPingAll 并发检测所有目标
func RunPingAll(targets []types.PingTarget) []types.PingResult {
	results := make([]types.PingResult, len(targets))
	var wg sync.WaitGroup

	for i, t := range targets {
		wg.Add(1)
		go func(idx int, target types.PingTarget) {
			defer wg.Done()
			latency, err := TCPPing(target.Host, 5*time.Second)
			if err != nil {
				results[idx] = types.PingResult{
					Name: target.Name, Host: target.Host,
					Latency: -1, Status: fmt.Sprintf("失败: %v", err),
				}
			} else {
				results[idx] = types.PingResult{
					Name: target.Name, Host: target.Host,
					Latency: latency.Milliseconds(), Status: "ok",
				}
			}
		}(i, t)
	}
	wg.Wait()
	return results
}
