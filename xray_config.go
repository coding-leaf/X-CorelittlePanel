package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

var configMutex sync.Mutex

// XrayClient 表示 Xray VLESS/VMess 入站中的单个用户信息
type XrayClient struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// XrayInbound 表示 Xray 配置文件中的一个 inbound 块
type XrayInbound struct {
	Tag      string          `json:"tag"`
	Protocol string          `json:"protocol"`
	Settings json.RawMessage `json:"settings"`
}

// XrayInboundSettings 表示 VLESS/VMess 的 settings 字段
type XrayInboundSettings struct {
	Clients    []XrayClient    `json:"clients,omitempty"`
	Decryption interface{}     `json:"decryption,omitempty"`
	Other      json.RawMessage `json:"-"`
}

// ============================================================
// JSONC 助手函数 — 剥离 Xray 配置中的注释
// ============================================================

// stripJSONC 从字节流中移除 // 和 /* */ 类型的注释，并处理 JSON 协议不严谨导致的尾部冗余逗号。
// Xray 配置文件通常允许注释，但标准 JSON 解析库不支持，因此必须先执行此过滤。
func stripJSONC(data []byte) []byte {
	var result []byte
	i := 0
	n := len(data)
	for i < n {
		if data[i] == '"' {
			result = append(result, data[i])
			i++
			for i < n && data[i] != '"' {
				if data[i] == '\\' && i+1 < n {
					result = append(result, data[i], data[i+1])
					i += 2
				} else {
					result = append(result, data[i])
					i++
				}
			}
			if i < n {
				result = append(result, data[i])
				i++
			}
			continue
		}
		if i+1 < n && data[i] == '/' && data[i+1] == '/' {
			for i < n && data[i] != '\n' {
				i++
			}
			continue
		}
		if i+1 < n && data[i] == '/' && data[i+1] == '*' {
			i += 2
			for i+1 < n && !(data[i] == '*' && data[i+1] == '/') {
				i++
			}
			if i+1 < n {
				i += 2
			}
			continue
		}
		result = append(result, data[i])
		i++
	}

	cleaned := make([]byte, 0, len(result))
	for i := 0; i < len(result); i++ {
		if result[i] == ',' {
			j := i + 1
			for j < len(result) && (result[j] == ' ' || result[j] == '\t' || result[j] == '\n' || result[j] == '\r') {
				j++
			}
			if j < len(result) && (result[j] == '}' || result[j] == ']') {
				continue
			}
		}
		cleaned = append(cleaned, result[i])
	}
	return cleaned
}

// readXrayConfig 从磁盘读取 Xray 配置文件并解析为 map
func readXrayConfig() (map[string]interface{}, error) {
	configMutex.Lock()
	defer configMutex.Unlock()

	raw, err := os.ReadFile(config.XrayConfigPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}
	cleaned := stripJSONC(raw)
	var cfg map[string]interface{}
	if err := json.Unmarshal(cleaned, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}
	return cfg, nil
}

// writeXrayConfig 将 map 序列化回 JSON 并写入 Xray 配置文件，同时创建备份
func writeXrayConfig(cfg map[string]interface{}) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	backupPath := config.XrayConfigPath + ".bak"
	if orig, err := os.ReadFile(config.XrayConfigPath); err == nil {
		os.WriteFile(backupPath, orig, 0644)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}
	if err := os.WriteFile(config.XrayConfigPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置失败: %v", err)
	}
	return nil
}

// extractClients 从配置文件中提取所有 VLESS 协议的用户列表
func extractClients(cfg map[string]interface{}) []XrayClient {
	clients := make(map[string]XrayClient)
	inbounds, ok := cfg["inbounds"].([]interface{})
	if !ok {
		return nil
	}
	for _, ib := range inbounds {
		ibMap, ok := ib.(map[string]interface{})
		if !ok {
			continue
		}
		protocol, _ := ibMap["protocol"].(string)
		if protocol != "vless" {
			continue
		}
		settings, ok := ibMap["settings"].(map[string]interface{})
		if !ok {
			continue
		}
		clientList, ok := settings["clients"].([]interface{})
		if !ok {
			continue
		}
		for _, c := range clientList {
			cMap, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			id, _ := cMap["id"].(string)
			email, _ := cMap["email"].(string)
			if id != "" && email != "" {
				clients[email] = XrayClient{ID: id, Email: email}
			}
		}
	}
	result := make([]XrayClient, 0, len(clients))
	for _, c := range clients {
		result = append(result, c)
	}
	return result
}

// addClientToConfig 将新用户添加到所有 VLESS 入站中
func addClientToConfig(cfg map[string]interface{}, client XrayClient) error {
	inbounds, ok := cfg["inbounds"].([]interface{})
	if !ok {
		return fmt.Errorf("未找到 inbounds 节点")
	}
	added := false
	for _, ib := range inbounds {
		ibMap, ok := ib.(map[string]interface{})
		if !ok {
			continue
		}
		protocol, _ := ibMap["protocol"].(string)
		if protocol != "vless" {
			continue
		}
		settings, ok := ibMap["settings"].(map[string]interface{})
		if !ok {
			continue
		}
		clientList, _ := settings["clients"].([]interface{})

		exists := false
		for _, c := range clientList {
			cMap, _ := c.(map[string]interface{})
			if email, _ := cMap["email"].(string); email == client.Email {
				exists = true
				break
			}
		}
		if !exists {
			newClient := map[string]interface{}{
				"id":    client.ID,
				"email": client.Email,
			}
			clientList = append(clientList, newClient)
			settings["clients"] = clientList
			added = true
		}
	}
	if !added {
		return fmt.Errorf("未找到对应的 VLESS 入站或用户已存在")
	}
	return nil
}

// removeClientFromConfig 从配置中删除指定 Email 的用户
func removeClientFromConfig(cfg map[string]interface{}, email string) error {
	inbounds, ok := cfg["inbounds"].([]interface{})
	if !ok {
		return fmt.Errorf("未找到 inbounds 节点")
	}
	removed := false
	for _, ib := range inbounds {
		ibMap, ok := ib.(map[string]interface{})
		if !ok {
			continue
		}
		protocol, _ := ibMap["protocol"].(string)
		if protocol != "vless" {
			continue
		}
		settings, ok := ibMap["settings"].(map[string]interface{})
		if !ok {
			continue
		}
		clientList, ok := settings["clients"].([]interface{})
		if !ok {
			continue
		}

		newClients := make([]interface{}, 0)
		for _, c := range clientList {
			cMap, _ := c.(map[string]interface{})
			if e, _ := cMap["email"].(string); e != email {
				newClients = append(newClients, c)
			} else {
				removed = true
			}
		}
		settings["clients"] = newClients
	}
	if !removed {
		return fmt.Errorf("未找到对应的用户 Email")
	}
	return nil
}
