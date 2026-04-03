//go:build !linux

package monitor

import (
	"os"
	"runtime"

	"xray-panel/internal/types"
)

// CollectSysInfo 非 Linux 平台的 stub 实现 (Windows/macOS 开发环境)
func CollectSysInfo() types.HostInfo {
	hostname, _ := os.Hostname()
	return types.HostInfo{
		Hostname: hostname,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		CPUs:     runtime.NumCPU(),
	}
}
