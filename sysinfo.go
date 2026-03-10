package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

// HostInfo 包含服务器的基本硬件和系统状态
type HostInfo struct {
	Hostname  string        `json:"hostname"`
	OS        string        `json:"os"`
	Arch      string        `json:"arch"`
	CPUs      int           `json:"cpus"`
	Memory    *MemInfo      `json:"memory"`
	Disk      []DiskInfo    `json:"disk"`
	Load      string        `json:"load"`
	Processes []ProcessInfo `json:"processes"`
	UptimeStr string        `json:"uptime"`
}

// MemInfo 表示内存使用情况
type MemInfo struct {
	Total     string `json:"total"`
	Used      string `json:"used"`
	Free      string `json:"free"`
	UsageRate string `json:"usage_rate"`
}

// DiskInfo 表示磁盘分区的使用情况
type DiskInfo struct {
	Mount   string `json:"mount"`
	Total   string `json:"total"`
	Used    string `json:"used"`
	Avail   string `json:"avail"`
	Percent string `json:"percent"`
}

// ProcessInfo 表示单个进程的资源占用
type ProcessInfo struct {
	PID  string `json:"pid"`
	Name string `json:"name"`
	CPU  string `json:"cpu"`
	Mem  string `json:"mem"`
}

// runCmd 简单的外部命令执行封装 (兜底用)
func runCmd(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// collectSysInfo 汇总收集当前服务器的各项性能指标
func collectSysInfo() HostInfo {
	hostname, _ := os.Hostname()
	info := HostInfo{
		Hostname: hostname,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		CPUs:     runtime.NumCPU(),
	}

	// 1. 系统负载与运行时间 (Linux 原生读取 /proc/loadavg)
	loadData, err := os.ReadFile("/proc/loadavg")
	if err == nil {
		parts := strings.Fields(string(loadData))
		if len(parts) >= 3 {
			info.Load = strings.Join(parts[:3], " ")
		}
	} else {
		info.Load = runCmd("uptime") // 兜底
	}
	info.UptimeStr = runCmd("uptime", "-p")

	// 2. 内存信息 (原生读取 /proc/meminfo)
	info.Memory = getNativeMemInfo()

	// 3. 磁盘信息 (原生系统调用)
	info.Disk = getNativeDiskInfo("/")

	// 4. 进程信息 (暂时保留 ps 命令，因为原生解析 /proc 过于复杂且容易出错)
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
				info.Processes = append(info.Processes, ProcessInfo{
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

// getNativeMemInfo 通过解析 /proc/meminfo 获取内存数据
func getNativeMemInfo() *MemInfo {
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

	return &MemInfo{
		Total:     formatBytes(total),
		Used:      formatBytes(used),
		Free:      formatBytes(available),
		UsageRate: fmt.Sprintf("%.1f%%", usageRate),
	}
}

// getNativeDiskInfo 使用 syscall 直接获取指定挂载点的磁盘字节数据。
// 比起 df -h，这种方式更准确且独立于 shell 环境。
func getNativeDiskInfo(path string) []DiskInfo {
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

	return []DiskInfo{{
		Mount:   path,
		Total:   formatBytes(total),
		Used:    formatBytes(used),
		Avail:   formatBytes(free),
		Percent: fmt.Sprintf("%.1f%%", percent),
	}}
}

// formatBytes 将字节数 (Byte) 转换为常见的人类可读单位 (KB, MB, GB, TB)。
func formatBytes(b uint64) string {
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
