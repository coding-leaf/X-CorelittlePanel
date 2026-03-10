package main

import (
	"context"
	"sort"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "xray-panel/proto"
)

// getStatsConn 创建与 Xray gRPC 控制接口的连接。
// 默认连接到 Xray 的 10085 端口。
func getStatsConn() (*grpc.ClientConn, error) {
	return grpc.Dial(config.XrayAPI,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(3*time.Second), // 设置连接超时，防止卡死
	)
}

// queryAllStats 向 Xray 请求全量流量指标数据。
// Pattern 为空代表查询所有已注册的 stats 项。
func queryAllStats(client pb.StatsServiceClient) ([]*pb.Stat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.QueryStats(ctx, &pb.QueryStatsRequest{Pattern: ""})
	if err != nil {
		return nil, err
	}
	return resp.Stat, nil
}

// getSysStats 获取 Xray 内部的系统性能指标（Uptime, Memory 等）。
// 注意：这部分是 Xray 进程自身的资源占用，不是整机的。
func getSysStats(client pb.StatsServiceClient) (*SysStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.GetSysStats(ctx, &pb.SysStatsRequest{})
	if err != nil {
		return nil, err
	}
	return &SysStats{
		Uptime:       resp.Uptime,
		NumGoroutine: resp.NumGoroutine,
		Alloc:        resp.Alloc,
		TotalAlloc:   resp.TotalAlloc,
		Sys:          resp.Sys,
		LiveObjects:  resp.LiveObjects,
	}, nil
}

// parseStats 是解析器的核心。它将 Xray 原始的 "category>>>name>>>traffic>>>direction" 格式字符串
// 映射为结构化的 UserTraffic、InboundTraffic 和 OutboundTraffic 对象，并按总流量降序排列。
func parseStats(stats []*pb.Stat) ([]UserTraffic, []InboundTraffic, []OutboundTraffic) {
	userMap := make(map[string]*UserTraffic)
	inboundMap := make(map[string]*InboundTraffic)
	outboundMap := make(map[string]*OutboundTraffic)

	for _, s := range stats {
		parts := strings.Split(s.Name, ">>>")
		if len(parts) != 4 {
			continue
		}
		category := parts[0] // user, inbound, outbound
		name := parts[1]
		// parts[2] = "traffic"
		direction := parts[3] // uplink, downlink

		switch category {
		case "user":
			if _, ok := userMap[name]; !ok {
				userMap[name] = &UserTraffic{Email: name}
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
				inboundMap[name] = &InboundTraffic{Tag: name}
			}
			if direction == "uplink" {
				inboundMap[name].Uplink = s.Value
			} else {
				inboundMap[name].Downlink = s.Value
			}
		case "outbound":
			if _, ok := outboundMap[name]; !ok {
				outboundMap[name] = &OutboundTraffic{Tag: name}
			}
			if direction == "uplink" {
				outboundMap[name].Uplink = s.Value
			} else {
				outboundMap[name].Downlink = s.Value
			}
		}
	}

	users := make([]UserTraffic, 0, len(userMap))
	for _, u := range userMap {
		u.Total = u.Uplink + u.Downlink
		users = append(users, *u)
	}
	sort.Slice(users, func(i, j int) bool { return users[i].Total > users[j].Total })

	inbounds := make([]InboundTraffic, 0, len(inboundMap))
	for _, ib := range inboundMap {
		ib.Total = ib.Uplink + ib.Downlink
		inbounds = append(inbounds, *ib)
	}
	sort.Slice(inbounds, func(i, j int) bool { return inbounds[i].Total > inbounds[j].Total })

	outbounds := make([]OutboundTraffic, 0, len(outboundMap))
	for _, ob := range outboundMap {
		ob.Total = ob.Uplink + ob.Downlink
		outbounds = append(outbounds, *ob)
	}
	sort.Slice(outbounds, func(i, j int) bool { return outbounds[i].Total > outbounds[j].Total })

	return users, inbounds, outbounds
}
