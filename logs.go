package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	accessLogRe = regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})(?:\.\d+)? from ([\d\.:]+) (?:accepted|rejected) (.*?) \[(.*?)\](?: email: (.*?))?$`)
	errorLogRe  = regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})(?:\.\d+)? \[(.*?)\] (.*?) (.*?): (.*?)(?:  > (.*))?$`)
)

// parseAccessLogLine 解析单条 Xray 访问日志
func parseAccessLogLine(line string) *AccessLogEntry {
	m := accessLogRe.FindStringSubmatch(line)
	if len(m) < 6 {
		return nil
	}
	// 从路由链 "inbound -> outbound" 或 "inbound >> outbound" 中提取出站标签
	route := m[4]
	if idx := strings.LastIndex(route, "-> "); idx >= 0 {
		route = strings.TrimSpace(route[idx+3:])
	} else if idx := strings.LastIndex(route, ">> "); idx >= 0 {
		route = strings.TrimSpace(route[idx+3:])
	}
	return &AccessLogEntry{
		Time:   m[1],
		FromIP: m[2],
		Target: m[3],
		Route:  route,
		Email:  m[5],
	}
}

// parseErrorLogLine 解析单条 Xray 错误日志
func parseErrorLogLine(line string) *ErrorLogEntry {
	m := errorLogRe.FindStringSubmatch(line)
	if len(m) < 6 {
		return nil
	}
	entry := &ErrorLogEntry{
		Time:    m[1],
		Level:   m[2],
		Module:  m[3],
		Message: m[4] + ": " + m[5],
	}
	if len(m) > 6 {
		entry.Error = m[6]
	}
	return entry
}

// readLastNLines 核心原生函数：利用 Seek 逆向高效读取文件末尾 N 行。
// 原理：不一次性读取整个文件，而是从末尾不断向前跳跃 64KB 的块进行搜索，直到找到足够的换行符。
// 这种方案即使面对几 GB 的日志文件，内存和 CPU 消耗也极低，完美替代 linux tail。
func readLastNLines(filePath string, n int) ([]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := fi.Size()
	if size == 0 {
		return nil, nil
	}

	var lines []string
	bufSize := int64(64 * 1024)
	offset := size
	remainder := ""

	for offset > 0 && len(lines) < n+1 {
		readSize := bufSize
		if offset < readSize {
			readSize = offset
		}
		offset -= readSize
		buf := make([]byte, readSize)
		_, err := f.ReadAt(buf, offset)
		if err != nil && err != io.EOF {
			return nil, err
		}
		chunk := string(buf) + remainder
		parts := strings.Split(chunk, "\n")
		remainder = parts[0]
		for i := len(parts) - 1; i >= 1; i-- {
			line := strings.TrimSpace(parts[i])
			if line != "" {
				lines = append(lines, line)
			}
		}
	}
	if remainder != "" && len(lines) < n {
		lines = append(lines, strings.TrimSpace(remainder))
	}

	if len(lines) > n {
		lines = lines[:n]
	}
	// 翻转为正序
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	return lines, nil
}

// handleAccessLogs API: 获取访问日志
func handleAccessLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if config.AccessLog == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "未配置访问日志路径"})
		return
	}

	count := 100
	if c := r.URL.Query().Get("count"); c != "" {
		if val, err := strconv.Atoi(c); err == nil && val > 0 {
			count = val
		}
	}

	emailFilter := r.URL.Query().Get("email")

	lines, err := readLastNLines(config.AccessLog, count)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	entries := make([]AccessLogEntry, 0)
	for _, line := range lines {
		entry := parseAccessLogLine(line)
		if entry != nil {
			if emailFilter != "" && entry.Email != emailFilter {
				continue
			}
			entries = append(entries, *entry)
		}
	}
	json.NewEncoder(w).Encode(entries)
}

// handleErrorLogs API: 获取错误日志
func handleErrorLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if config.ErrorLog == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "未配置错误日志路径"})
		return
	}

	count := 100
	if c := r.URL.Query().Get("count"); c != "" {
		if val, err := strconv.Atoi(c); err == nil && val > 0 {
			count = val
		}
	}

	lines, err := readLastNLines(config.ErrorLog, count)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	entries := make([]ErrorLogEntry, 0)
	for _, line := range lines {
		entry := parseErrorLogLine(line)
		if entry != nil {
			entries = append(entries, *entry)
		}
	}
	json.NewEncoder(w).Encode(entries)
}

// handleClearLogs API: 清除指定类型的日志
func handleClearLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(405)
		json.NewEncoder(w).Encode(map[string]string{"error": "方法不允许"})
		return
	}

	var req struct {
		Type string `json:"type"` // "access" or "error"
	}
	json.NewDecoder(r.Body).Decode(&req)

	var path string
	switch req.Type {
	case "access":
		path = config.AccessLog
	case "error":
		path = config.ErrorLog
	default:
		json.NewEncoder(w).Encode(map[string]string{"error": "无效类型"})
		return
	}

	if path == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "未配置日志路径"})
		return
	}

	err := os.Truncate(path, 0)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "清除失败: " + err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
