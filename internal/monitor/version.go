package monitor

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// VersionInfo 版本信息
type VersionInfo struct {
	Current    string `json:"current"`
	Latest     string `json:"latest"`
	HasUpdate  bool   `json:"has_update"`
	CheckedAt  string `json:"checked_at"`
	Error      string `json:"error,omitempty"`
}

// VersionChecker 版本检查器
type VersionChecker struct {
	mu      sync.RWMutex
	info    VersionInfo
	chinaTZ *time.Location
}

// NewVersionChecker 创建版本检查器
func NewVersionChecker(tz *time.Location) *VersionChecker {
	return &VersionChecker{chinaTZ: tz}
}

// GetInfo 返回缓存的版本信息
func (vc *VersionChecker) GetInfo() VersionInfo {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return vc.info
}

// Check 立即执行一次版本检查
func (vc *VersionChecker) Check() VersionInfo {
	current := GetLocalXrayVersion()
	latest, err := GetLatestXrayVersion()

	vc.mu.Lock()
	defer vc.mu.Unlock()

	vc.info.Current = current
	vc.info.CheckedAt = time.Now().In(vc.chinaTZ).Format("2006-01-02 15:04:05")

	if err != nil {
		vc.info.Error = err.Error()
		vc.info.Latest = ""
		vc.info.HasUpdate = false
	} else {
		vc.info.Latest = latest
		vc.info.Error = ""
		vc.info.HasUpdate = latest != "" && current != "" && normalizeVersion(latest) != normalizeVersion(current)
	}

	return vc.info
}

// StartPeriodicCheck 启动定期检查 (每6小时)，发现新版本时通过 onNewVersion 回调通知
func (vc *VersionChecker) StartPeriodicCheck(onNewVersion func(info VersionInfo)) {
	// 启动后延迟30秒首次检查
	go func() {
		time.Sleep(30 * time.Second)
		info := vc.Check()
		log.Printf("[VersionCheck] 当前: %s, 最新: %s, 有更新: %v", info.Current, info.Latest, info.HasUpdate)
		if info.HasUpdate && onNewVersion != nil {
			onNewVersion(info)
		}

		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		lastNotified := info.Latest
		for range ticker.C {
			info = vc.Check()
			if info.HasUpdate && info.Latest != lastNotified && onNewVersion != nil {
				onNewVersion(info)
				lastNotified = info.Latest
			}
		}
	}()
}

// GetLocalXrayVersion 获取本地 Xray 版本
func GetLocalXrayVersion() string {
	out, err := exec.Command("xray", "version").CombinedOutput()
	if err != nil {
		// 尝试完整路径
		out, err = exec.Command("/usr/local/bin/xray", "version").CombinedOutput()
		if err != nil {
			return ""
		}
	}
	// 解析输出: "Xray 26.3.27 (Xray, Penetrates Everything.) Custom (go1.26.1 linux/amd64)"
	lines := strings.Split(string(out), "\n")
	if len(lines) > 0 {
		parts := strings.Fields(lines[0])
		if len(parts) >= 2 {
			return parts[1] // "26.3.27" 或 "v26.3.27"
		}
	}
	return ""
}

// GetLatestXrayVersion 从 GitHub API 获取最新 release tag
func GetLatestXrayVersion() (string, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/XTLS/Xray-core/releases/latest", nil)
	req.Header.Set("User-Agent", "XrayPanel/2.0")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求 GitHub API 失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API 返回 %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("解析 GitHub 响应失败: %v", err)
	}
	return release.TagName, nil
}

// normalizeVersion 统一版本格式去掉 v 前缀
func normalizeVersion(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

// RunXrayUpdate 执行 Xray 更新脚本
func RunXrayUpdate() (string, error) {
	cmd := exec.Command("bash", "-c",
		`bash -c "$(curl -L https://github.com/XTLS/Xray-install/raw/main/install-release.sh)" @ install 2>&1`)
	cmd.Env = append(cmd.Environ(), "LANG=en_US.UTF-8")
	out, err := cmd.CombinedOutput()
	result := string(out)
	if err != nil {
		return result, fmt.Errorf("更新失败: %v\n%s", err, result)
	}
	return result, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
