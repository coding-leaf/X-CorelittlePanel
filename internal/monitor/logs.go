package monitor

import (
	"io"
	"os"
	"regexp"
	"strings"

	"xray-panel/internal/types"
)

var (
	accessLogRe = regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})(?:\.\d+)? from ([\d\.:]+) (?:accepted|rejected) (.*?) \[(.*?)\](?: email: (.*?))?$`)
	errorLogRe  = regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})(?:\.\d+)? \[(.*?)\] (.*?) (.*?): (.*?)(?:  > (.*))?$`)
)

// ParseAccessLogLine 解析单条 Xray 访问日志
func ParseAccessLogLine(line string) *types.AccessLogEntry {
	m := accessLogRe.FindStringSubmatch(line)
	if len(m) < 6 {
		return nil
	}
	route := m[4]
	if idx := strings.LastIndex(route, "-> "); idx >= 0 {
		route = strings.TrimSpace(route[idx+3:])
	} else if idx := strings.LastIndex(route, ">> "); idx >= 0 {
		route = strings.TrimSpace(route[idx+3:])
	}
	return &types.AccessLogEntry{
		Time:   m[1],
		FromIP: m[2],
		Target: m[3],
		Route:  route,
		Email:  m[5],
	}
}

// ParseErrorLogLine 解析单条 Xray 错误日志
func ParseErrorLogLine(line string) *types.ErrorLogEntry {
	m := errorLogRe.FindStringSubmatch(line)
	if len(m) < 6 {
		return nil
	}
	entry := &types.ErrorLogEntry{
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

// ReadLastNLines 从文件末尾高效读取最后 N 行
func ReadLastNLines(filePath string, n int) ([]string, error) {
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
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	return lines, nil
}
