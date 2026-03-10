package main

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================
// Xray 服务控制逻辑 (依赖 systemctl)
// ============================================================

var xrayMu sync.Mutex

// xrayServiceCmd 执行 systemctl 相关的服务管理命令
func xrayServiceCmd(action string) (string, error) {
	xrayMu.Lock()
	defer xrayMu.Unlock()
	out, err := exec.Command("systemctl", action, "xray").CombinedOutput()
	result := strings.TrimSpace(string(out))

	// 针对部分没有配 reload 的 xray.service 脚本进行容错降级
	if action == "reload" && err != nil {
		log.Printf("[Admin] Xray reload 失败，自动降级为 restart。原错误: %v, 输出: %s", err, result)
		out, err = exec.Command("systemctl", "restart", "xray").CombinedOutput()
		result = strings.TrimSpace(string(out))
	}

	return result, err
}

// getXrayStatus 获取 Xray 服务当前是否正在运行
func getXrayStatus() string {
	out, _ := exec.Command("systemctl", "is-active", "xray").Output()
	return strings.TrimSpace(string(out))
}

// ============================================================
// 证书解析逻辑 (原生 Go 实现)
// ============================================================

// CertInfo 包含从 X.509 证书中提取的关键信息
type CertInfo struct {
	Subject    string   `json:"subject"`
	Issuer     string   `json:"issuer"`
	NotBefore  string   `json:"not_before"`
	NotAfter   string   `json:"not_after"`
	DaysLeft   int      `json:"days_left"`
	DNSNames   []string `json:"dns_names"`
	IsExpired  bool     `json:"is_expired"`
	IsExpiring bool     `json:"is_expiring"` // 剩余 30 天内
}

// parseCertificate 读取 PEM 格式的证书文件并解析其有效期和域名
func parseCertificate(certPath string) (*CertInfo, error) {
	data, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("读取证书失败: %v", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("证书 PEM 解码失败")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("解析证书失败: %v", err)
	}
	now := time.Now()
	daysLeft := int(cert.NotAfter.Sub(now).Hours() / 24)
	return &CertInfo{
		Subject:    cert.Subject.CommonName,
		Issuer:     cert.Issuer.CommonName,
		NotBefore:  cert.NotBefore.In(chinaTZ).Format("2006-01-02 15:04:05"),
		NotAfter:   cert.NotAfter.In(chinaTZ).Format("2006-01-02 15:04:05"),
		DaysLeft:   daysLeft,
		DNSNames:   cert.DNSNames,
		IsExpired:  now.After(cert.NotAfter),
		IsExpiring: daysLeft < 30 && daysLeft >= 0,
	}, nil
}

// ============================================================
// 后台 API 处理函数
// ============================================================

// handleAdminPage 提供后台管理 HTML 主页面
func handleAdminPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("CDN-Cache-Control", "no-store")
	w.Write([]byte(adminHTML))
}

// handleAdminUsers 接口：获取当前 Xray 配置中的所有用户及其 UUID
func handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if config.XrayConfigPath == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "未配置 xray_config_path"})
		return
	}
	cfg, err := readXrayConfig()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	clients := extractClients(cfg)
	if clients == nil {
		clients = []XrayClient{}
	}
	json.NewEncoder(w).Encode(clients)
}

// handleAdminAddUser 接口：向 Xray 中添加一个新用户 (VLESS)
func handleAdminAddUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(405)
		json.NewEncoder(w).Encode(map[string]string{"error": "方法不允许"})
		return
	}
	var req struct {
		Email string `json:"email"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Email == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "email 不能为空"})
		return
	}

	// 安全校验：禁止包含特殊字符，防止破坏 Xray API 的流量统计解析逻辑 ("XXX>>>YYY>>>traffic>>>downlink")
	if strings.ContainsAny(req.Email, "><:\\/\"'") {
		json.NewEncoder(w).Encode(map[string]string{"error": "email 包含非法特殊字符"})
		return
	}

	cfg, err := readXrayConfig()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	newUUID := uuid.New().String()
	client := XrayClient{ID: newUUID, Email: req.Email}

	if err := addClientToConfig(cfg, client); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if err := writeXrayConfig(cfg); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// 无论重载成功与否，都不应该向前端抛出致命 error，否则前端会卡住
	output, err := xrayServiceCmd("reload")
	reloadStatus := "ok"
	if err != nil {
		reloadStatus = "reload失败: " + output
		log.Printf("[Admin] 添加用户后重启 Xray 失败: %v, 输出: %s", err, output)
	}

	if tgBot != nil {
		tgBot.SendNotification(fmt.Sprintf("👤 *新用户添加*\nEmail: `%s`\nUUID: `%s`", req.Email, newUUID))
	}

	log.Printf("[Admin] 添加用户: %s (UUID: %s)", req.Email, newUUID)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"uuid":   newUUID,
		"reload": reloadStatus,
	})
}

// handleAdminDeleteUser 接口：根据 Email 删除用户并自动重载 Xray
func handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(405)
		json.NewEncoder(w).Encode(map[string]string{"error": "方法不允许"})
		return
	}
	var req struct {
		Email string `json:"email"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Email == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "email 不能为空"})
		return
	}

	cfg, err := readXrayConfig()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if err := removeClientFromConfig(cfg, req.Email); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if err := writeXrayConfig(cfg); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	output, err := xrayServiceCmd("reload")
	reloadStatus := "ok"
	if err != nil {
		reloadStatus = "reload失败: " + output
		log.Printf("[Admin] 删除用户后重启 Xray 失败: %v, 输出: %s", err, output)
	}

	if trafficStore != nil {
		trafficStore.ResetUser(req.Email)
	}

	if tgBot != nil {
		tgBot.SendNotification(fmt.Sprintf("🗑 *用户已删除*\nEmail: `%s`", req.Email))
	}

	log.Printf("[Admin] 删除用户: %s", req.Email)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"reload": reloadStatus,
	})
}

// handleXrayRestart 接口：重启 Xray 背景服务
func handleXrayRestart(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(405)
		json.NewEncoder(w).Encode(map[string]string{"error": "方法不允许"})
		return
	}
	output, err := xrayServiceCmd("restart")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "重启失败: " + output})
		return
	}
	if tgBot != nil {
		tgBot.SendNotification("🔄 *Xray 已重启*")
	}
	log.Printf("[Admin] Xray 重启")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleXrayReload 接口：平滑重载 Xray 配置
func handleXrayReload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(405)
		json.NewEncoder(w).Encode(map[string]string{"error": "方法不允许"})
		return
	}
	output, err := xrayServiceCmd("reload")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "重载失败: " + output})
		return
	}
	if tgBot != nil {
		tgBot.SendNotification("♻️ *Xray 配置已重载*")
	}
	log.Printf("[Admin] Xray 重载配置")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "message": strings.TrimSpace(output)})
}

// handleXrayStatus 接口：返回 Xray 当前运行状态
func handleXrayStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status := getXrayStatus()
	json.NewEncoder(w).Encode(map[string]string{"status": status})
}

// handleConfigGet 接口：获取原始的 Xray 配置文件内容 (用于编辑器)
func handleConfigGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if config.XrayConfigPath == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "未配置 xray_config_path"})
		return
	}
	data, err := os.ReadFile(config.XrayConfigPath)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "读取失败: " + err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"content": string(data)})
}

// handleConfigSave 接口：保存前端修改后的 Xray 配置文件
func handleConfigSave(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(405)
		json.NewEncoder(w).Encode(map[string]string{"error": "方法不允许"})
		return
	}
	var req struct {
		Content string `json:"content"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Content == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "内容不能为空"})
		return
	}

	backupPath := config.XrayConfigPath + ".bak"
	if orig, err := os.ReadFile(config.XrayConfigPath); err == nil {
		os.WriteFile(backupPath, orig, 0644)
	}

	if err := os.WriteFile(config.XrayConfigPath, []byte(req.Content), 0644); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "写入失败: " + err.Error()})
		return
	}

	log.Printf("[Admin] 配置文件已保存")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleConfigValidate 接口：调用 Xray 二进制进行配置文件语法校验
func handleConfigValidate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(405)
		json.NewEncoder(w).Encode(map[string]string{"error": "方法不允许"})
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Content == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "配置内容为空"})
		return
	}

	xrayBin := config.XrayBinPath
	if xrayBin == "" {
		xrayBin = "xray"
		if _, err := exec.LookPath(xrayBin); err != nil {
			for _, p := range []string{"/usr/local/bin/xray", "/usr/bin/xray"} {
				if _, err := os.Stat(p); err == nil {
					xrayBin = p
					break
				}
			}
		}
	}

	// 安全校验：防止 config.json 中的 xray_bin_path 被恶意篡改为其他危险命令（命令注入防范）
	if !strings.HasSuffix(xrayBin, "/xray") && xrayBin != "xray" && !strings.HasSuffix(xrayBin, "\\xray.exe") && xrayBin != "xray.exe" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid":   false,
			"message": "安全错误: 无效的 xray 执行路径 (" + xrayBin + ")",
		})
		return
	}

	// 将前端传来的内容写入临时文件进行验证
	tmpFile, err := os.CreateTemp("", "xray-test-*.json")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid":   false,
			"message": "无法创建临时测试文件: " + err.Error(),
		})
		return
	}
	defer os.Remove(tmpFile.Name()) // 确保验证完毕后删除临时文件

	tmpFile.Write([]byte(req.Content))
	tmpFile.Close()

	out, err := exec.Command(xrayBin, "-test", "-config", tmpFile.Name()).CombinedOutput()
	result := strings.TrimSpace(string(out))
	if err != nil {
		// exec.Command returns an error if the exit code is non-zero, which means config is invalid
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid":   false,
			"message": result, // The output contains the actual syntax error
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":   true,
		"message": result,
	})
}

// handleCertStatus 接口：查看当前配置的 SSL 证书状态
func handleCertStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if config.CertPath == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "未配置 cert_path"})
		return
	}
	info, err := parseCertificate(config.CertPath)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(info)
}

// handleConfigRestore 接口：从备份文件还原 Xray 配置
func handleConfigRestore(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(405)
		json.NewEncoder(w).Encode(map[string]string{"error": "方法不允许"})
		return
	}
	backupPath := config.XrayConfigPath + ".bak"
	data, err := os.ReadFile(backupPath)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "备份文件不存在: " + err.Error()})
		return
	}
	if err := os.WriteFile(config.XrayConfigPath, data, 0644); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "还原失败: " + err.Error()})
		return
	}
	log.Printf("[Admin] 配置文件已从备份还原")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
