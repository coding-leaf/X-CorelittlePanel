//go:build linux

package monitor

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"xray-panel/internal/types"
)

// CollectSysInfo 汇总收集当前服务器的各项性能指标 (Linux 实现)
func CollectSysInfo() types.HostInfo {
	hostname, _ := os.Hostname()
	info := types.HostInfo{
		Hostname: hostname,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		CPUs:     runtime.NumCPU(),
	}

	loadData, err := os.ReadFile("/proc/loadavg")
	if err == nil {
		parts := strings.Fields(string(loadData))
		if len(parts) >= 3 {
			info.Load = strings.Join(parts[:3], " ")
		}
	} else {
		info.Load = runCmd("uptime")
	}
	info.UptimeStr = runCmd("uptime", "-p")

	info.Memory = getNativeMemInfo()
	info.Disk = getNativeDiskInfo("/")

	psOut := runCmd("ps", "--no-headers", "-eo", "pid,comm,%cpu,%mem", "--sort=-%cpu")
	if psOut != "" {
		lines := strings.Split(psOut, "\n")
		count := 10
		if len(lines) < count {
			count = len(lines)
		}
		for i := 0; i < count; i++ {
			fields := strings.Fields(lines[i])
			if len(fields) >= 4 {
				info.Processes = append(info.Processes, types.ProcessInfo{
					PID:  fields[0],
					Name: fields[1],
					CPU:  fields[2],
					Mem:  fields[3],
				})
			}
		}
	}

	return info
}

func getNativeMemInfo() *types.MemInfo {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return nil
	}

	var total, available uint64
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		val, _ := strconv.ParseUint(fields[1], 10, 64)
		switch key {
		case "MemTotal":
			total = val * 1024
		case "MemAvailable":
			available = val * 1024
		}
	}

	if total == 0 {
		return nil
	}

	used := total - available
	usageRate := float64(used) / float64(total) * 100

	return &types.MemInfo{
		Total:     FormatBytes(total),
		Used:      FormatBytes(used),
		Free:      FormatBytes(available),
		UsageRate: fmt.Sprintf("%.1f%%", usageRate),
	}
}

func getNativeDiskInfo(path string) []types.DiskInfo {
	var fs syscall.Statfs_t
	err := syscall.Statfs(path, &fs)
	if err != nil {
		return nil
	}

	total := fs.Blocks * uint64(fs.Bsize)
	free := fs.Bfree * uint64(fs.Bsize)
	used := total - free

	percent := 0.0
	if total > 0 {
		percent = float64(used) / float64(total) * 100
	}

	return []types.DiskInfo{{
		Mount:   path,
		Total:   FormatBytes(total),
		Used:    FormatBytes(used),
		Avail:   FormatBytes(free),
		Percent: fmt.Sprintf("%.1f%%", percent),
	}}
}
