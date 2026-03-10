// Code generated manually to match Xray stats gRPC service.
// This avoids requiring protoc/protoc-gen-go on the build machine.

package proto

import (
	"context"

	"google.golang.org/grpc"
)

// Request/Response types

type GetStatsRequest struct {
	Name   string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Reset_ bool   `protobuf:"varint,2,opt,name=reset,proto3" json:"reset,omitempty"`
}

type Stat struct {
	Name  string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Value int64  `protobuf:"varint,2,opt,name=value,proto3" json:"value,omitempty"`
}

type GetStatsResponse struct {
	Stat *Stat `protobuf:"bytes,1,opt,name=stat,proto3" json:"stat,omitempty"`
}

type QueryStatsRequest struct {
	Pattern  string   `protobuf:"bytes,1,opt,name=pattern,proto3" json:"pattern,omitempty"`
	Reset_   bool     `protobuf:"varint,2,opt,name=reset,proto3" json:"reset,omitempty"`
	Patterns []string `protobuf:"bytes,3,rep,name=patterns,proto3" json:"patterns,omitempty"`
	Regexp   bool     `protobuf:"varint,4,opt,name=regexp,proto3" json:"regexp,omitempty"`
}

type QueryStatsResponse struct {
	Stat []*Stat `protobuf:"bytes,1,rep,name=stat,proto3" json:"stat,omitempty"`
}

type SysStatsRequest struct{}

type SysStatsResponse struct {
	NumGoroutine uint32 `protobuf:"varint,1,opt,name=NumGoroutine,proto3" json:"NumGoroutine,omitempty"`
	NumGC        uint32 `protobuf:"varint,2,opt,name=NumGC,proto3" json:"NumGC,omitempty"`
	Alloc        uint64 `protobuf:"varint,3,opt,name=Alloc,proto3" json:"Alloc,omitempty"`
	TotalAlloc   uint64 `protobuf:"varint,4,opt,name=TotalAlloc,proto3" json:"TotalAlloc,omitempty"`
	Sys          uint64 `protobuf:"varint,5,opt,name=Sys,proto3" json:"Sys,omitempty"`
	Mallocs      uint64 `protobuf:"varint,6,opt,name=Mallocs,proto3" json:"Mallocs,omitempty"`
	Frees        uint64 `protobuf:"varint,7,opt,name=Frees,proto3" json:"Frees,omitempty"`
	LiveObjects  uint64 `protobuf:"varint,8,opt,name=LiveObjects,proto3" json:"LiveObjects,omitempty"`
	PauseTotalNs uint64 `protobuf:"varint,9,opt,name=PauseTotalNs,proto3" json:"PauseTotalNs,omitempty"`
	Uptime       uint32 `protobuf:"varint,10,opt,name=Uptime,proto3" json:"Uptime,omitempty"`
}

// Implement proto.Message interface minimally for gRPC
func (m *GetStatsRequest) Reset()         {}
func (m *GetStatsRequest) String() string { return "" }
func (m *GetStatsRequest) ProtoMessage()  {}

func (m *GetStatsResponse) Reset()         {}
func (m *GetStatsResponse) String() string { return "" }
func (m *GetStatsResponse) ProtoMessage()  {}

func (m *Stat) Reset()         {}
func (m *Stat) String() string { return "" }
func (m *Stat) ProtoMessage()  {}

func (m *QueryStatsRequest) Reset()         {}
func (m *QueryStatsRequest) String() string { return "" }
func (m *QueryStatsRequest) ProtoMessage()  {}

func (m *QueryStatsResponse) Reset()         {}
func (m *QueryStatsResponse) String() string { return "" }
func (m *QueryStatsResponse) ProtoMessage()  {}

func (m *SysStatsRequest) Reset()         {}
func (m *SysStatsRequest) String() string { return "" }
func (m *SysStatsRequest) ProtoMessage()  {}

func (m *SysStatsResponse) Reset()         {}
func (m *SysStatsResponse) String() string { return "" }
func (m *SysStatsResponse) ProtoMessage()  {}

// StatsService client

type StatsServiceClient interface {
	GetStats(ctx context.Context, in *GetStatsRequest, opts ...grpc.CallOption) (*GetStatsResponse, error)
	QueryStats(ctx context.Context, in *QueryStatsRequest, opts ...grpc.CallOption) (*QueryStatsResponse, error)
	GetSysStats(ctx context.Context, in *SysStatsRequest, opts ...grpc.CallOption) (*SysStatsResponse, error)
}

type statsServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewStatsServiceClient(cc grpc.ClientConnInterface) StatsServiceClient {
	return &statsServiceClient{cc}
}

func (c *statsServiceClient) GetStats(ctx context.Context, in *GetStatsRequest, opts ...grpc.CallOption) (*GetStatsResponse, error) {
	out := new(GetStatsResponse)
	err := c.cc.Invoke(ctx, "/xray.app.stats.command.StatsService/GetStats", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *statsServiceClient) QueryStats(ctx context.Context, in *QueryStatsRequest, opts ...grpc.CallOption) (*QueryStatsResponse, error) {
	out := new(QueryStatsResponse)
	err := c.cc.Invoke(ctx, "/xray.app.stats.command.StatsService/QueryStats", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *statsServiceClient) GetSysStats(ctx context.Context, in *SysStatsRequest, opts ...grpc.CallOption) (*SysStatsResponse, error) {
	out := new(SysStatsResponse)
	err := c.cc.Invoke(ctx, "/xray.app.stats.command.StatsService/GetSysStats", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
