package xray

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "xray-panel/proto"
	"xray-panel/internal/types"
)

// Client 封装与 Xray gRPC API 的通信
type Client struct {
	apiAddr string
}

// NewClient 创建 Xray API 客户端
func NewClient(apiAddr string) *Client {
	return &Client{apiAddr: apiAddr}
}

// GetStatsConn 创建 gRPC 连接
func (c *Client) GetStatsConn() (*grpc.ClientConn, error) {
	return grpc.Dial(c.apiAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(3*time.Second),
	)
}

// QueryAllStats 查询全量流量指标
func (c *Client) QueryAllStats(client pb.StatsServiceClient) ([]*pb.Stat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := client.QueryStats(ctx, &pb.QueryStatsRequest{Pattern: ""})
	if err != nil {
		return nil, err
	}
	return resp.Stat, nil
}

// GetSysStats 获取 Xray 内部系统指标
func (c *Client) GetSysStats(client pb.StatsServiceClient) (*types.SysStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := client.GetSysStats(ctx, &pb.SysStatsRequest{})
	if err != nil {
		return nil, err
	}
	return &types.SysStats{
		Uptime:       resp.Uptime,
		NumGoroutine: resp.NumGoroutine,
		Alloc:        resp.Alloc,
		TotalAlloc:   resp.TotalAlloc,
		Sys:          resp.Sys,
		LiveObjects:  resp.LiveObjects,
	}, nil
}

// ParseStats 将原始 Stat 解析为结构化类型
func ParseStats(stats []*pb.Stat) ([]types.UserTraffic, []types.InboundTraffic, []types.OutboundTraffic) {
	userMap := make(map[string]*types.UserTraffic)
	inboundMap := make(map[string]*types.InboundTraffic)
	outboundMap := make(map[string]*types.OutboundTraffic)

	for _, s := range stats {
		parts := strings.Split(s.Name, ">>>")
		if len(parts) != 4 {
			continue
		}
		category := parts[0]
		name := parts[1]
		direction := parts[3]

		switch category {
		case "user":
			if _, ok := userMap[name]; !ok {
				userMap[name] = &types.UserTraffic{Email: name}
			}
			if direction == "uplink" {
				userMap[name].Uplink = s.Value
			} else {
				userMap[name].Downlink = s.Value
			}
		case "inbound":
			if name == "api" {
				continue
			}
			if _, ok := inboundMap[name]; !ok {
				inboundMap[name] = &types.InboundTraffic{Tag: name}
			}
			if direction == "uplink" {
				inboundMap[name].Uplink = s.Value
			} else {
				inboundMap[name].Downlink = s.Value
			}
		case "outbound":
			if _, ok := outboundMap[name]; !ok {
				outboundMap[name] = &types.OutboundTraffic{Tag: name}
			}
			if direction == "uplink" {
				outboundMap[name].Uplink = s.Value
			} else {
				outboundMap[name].Downlink = s.Value
			}
		}
	}

	users := make([]types.UserTraffic, 0, len(userMap))
	for _, u := range userMap {
		u.Total = u.Uplink + u.Downlink
		users = append(users, *u)
	}
	sort.Slice(users, func(i, j int) bool { return users[i].Total > users[j].Total })

	inbounds := make([]types.InboundTraffic, 0, len(inboundMap))
	for _, ib := range inboundMap {
		ib.Total = ib.Uplink + ib.Downlink
		inbounds = append(inbounds, *ib)
	}
	sort.Slice(inbounds, func(i, j int) bool { return inbounds[i].Total > inbounds[j].Total })

	outbounds := make([]types.OutboundTraffic, 0, len(outboundMap))
	for _, ob := range outboundMap {
		ob.Total = ob.Uplink + ob.Downlink
		outbounds = append(outbounds, *ob)
	}
	sort.Slice(outbounds, func(i, j int) bool { return outbounds[i].Total > outbounds[j].Total })

	return users, inbounds, outbounds
}

// ============== Xray Config 操作 ==============

var configMutex sync.Mutex

// StripJSONC 移除 JSONC 注释和尾部逗号
func StripJSONC(data []byte) []byte {
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

// ReadXrayConfig 从磁盘读取 Xray 配置文件并解析为 map
func ReadXrayConfig(path string) (map[string]interface{}, error) {
	configMutex.Lock()
	defer configMutex.Unlock()

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}
	cleaned := StripJSONC(raw)
	var cfg map[string]interface{}
	if err := json.Unmarshal(cleaned, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}
	return cfg, nil
}

// WriteXrayConfig 将 map 序列化回 JSON 并写入 Xray 配置文件，同时创建备份
func WriteXrayConfig(path string, cfg map[string]interface{}) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	backupPath := path + ".bak"
	if orig, err := os.ReadFile(path); err == nil {
		os.WriteFile(backupPath, orig, 0644)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入配置失败: %v", err)
	}
	return nil
}

// ExtractClients 从配置中提取所有 VLESS 用户
func ExtractClients(cfg map[string]interface{}) []types.XrayClient {
	clients := make(map[string]types.XrayClient)
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
				clients[email] = types.XrayClient{ID: id, Email: email}
			}
		}
	}
	result := make([]types.XrayClient, 0, len(clients))
	for _, c := range clients {
		result = append(result, c)
	}
	return result
}

// AddClientToConfig 将新用户添加到所有 VLESS 入站中
func AddClientToConfig(cfg map[string]interface{}, client types.XrayClient) error {
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

// RemoveClientFromConfig 从配置中删除指定 Email 的用户
func RemoveClientFromConfig(cfg map[string]interface{}, email string) error {
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
