// Package netsim 提供 NETCONF 模拟网元，用于 E2E 集成测试
package netsim

import (
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"sync"
)

// Simulator NETCONF 模拟服务器
type Simulator struct {
	mu          sync.RWMutex
	addr        string
	port        int
	listener    net.Listener
	running     bool
	vlans       map[int]*VLANConfig
	username    string
	password    string
	errorOnRPC  map[string]error
	connections []net.Conn
}

// VLANConfig 模拟网元上的 VLAN 配置
type VLANConfig struct {
	ID         int      `xml:"vlan-id"`
	Name       string   `xml:"name,omitempty"`
	AdminState string   `xml:"admin-state,omitempty"`
	TaggedPorts []string `xml:"tagged-ports>port,omitempty"`
	UntaggedPorts []string `xml:"untagged-ports>port,omitempty"`
}

// NewSimulator 创建新的模拟服务器
func NewSimulator() *Simulator {
	return &Simulator{
		vlans: map[int]*VLANConfig{
			1: {
				ID:         1,
				Name:       "default",
				AdminState: "UP",
				UntaggedPorts: []string{"GE0/1", "GE0/2"},
			},
			10: {
				ID:         10,
				Name:       "Management",
				AdminState: "UP",
				TaggedPorts: []string{"GE0/3"},
			},
			20: {
				ID:         20,
				Name:       "User_Network",
				AdminState: "UP",
				TaggedPorts: []string{"GE0/4", "GE0/5"},
			},
			30: {
				ID:         30,
				Name:       "Guest",
				AdminState: "DOWN",
			},
		},
		username:   "admin",
		password:   "admin",
		errorOnRPC: map[string]error{},
	}
}

// Start 启动模拟服务器
func (s *Simulator) Start() error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}

	s.listener = listener
	addr := listener.Addr().(*net.TCPAddr)
	s.addr = "127.0.0.1"
	s.port = addr.Port
	s.running = true

	go s.acceptLoop()
	return nil
}

// Stop 停止服务器
func (s *Simulator) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.running = false
	if s.listener != nil {
		s.listener.Close()
	}
	for _, conn := range s.connections {
		conn.Close()
	}
}

// Addr 返回服务器地址
func (s *Simulator) Addr() string { return s.addr }

// Port 返回服务器端口
func (s *Simulator) Port() int { return s.port }

// Username 返回用户名
func (s *Simulator) Username() string { return s.username }

// Password 返回密码
func (s *Simulator) Password() string { return s.password }

// AssertVlanExists 断言 VLAN 存在
func (s *Simulator) AssertVlanExists(t interface{ Errorf(string, ...interface{}) }, id int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.vlans[id]
	if !exists {
		t.Errorf("expected VLAN %d to exist, but not found", id)
	}
	return exists
}

// AssertVlanName 断言 VLAN 名称
func (s *Simulator) AssertVlanName(t interface{ Errorf(string, ...interface{}) }, id int, name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vlan, exists := s.vlans[id]
	if !exists {
		t.Errorf("expected VLAN %d to exist, but not found", id)
		return false
	}
	if vlan.Name != name {
		t.Errorf("expected VLAN %d name to be %q, got %q", id, name, vlan.Name)
		return false
	}
	return true
}

// GetVLAN 获取 VLAN 配置
func (s *Simulator) GetVLAN(id int) *VLANConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.vlans[id]
}

// GetAllVLANs 获取所有 VLAN
func (s *Simulator) GetAllVLANs() []*VLANConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vlans := make([]*VLANConfig, 0, len(s.vlans))
	for _, v := range s.vlans {
		vlans = append(vlans, v)
	}
	return vlans
}

// AddVLAN 添加 VLAN
func (s *Simulator) AddVLAN(vlan *VLANConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vlans[vlan.ID] = vlan
}

// DeleteVLAN 删除 VLAN
func (s *Simulator) DeleteVLAN(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.vlans, id)
}

// SetErrorOnRPC 设置某个 RPC 总是失败
func (s *Simulator) SetErrorOnRPC(rpcName string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errorOnRPC[rpcName] = err
}

// ClearErrors 清除所有错误场景
func (s *Simulator) ClearErrors() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errorOnRPC = map[string]error{}
}

func (s *Simulator) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.mu.RLock()
			running := s.running
			s.mu.RUnlock()
			if !running {
				return
			}
			continue
		}

		s.mu.Lock()
		s.connections = append(s.connections, conn)
		s.mu.Unlock()

		go s.handleSession(conn)
	}
}

func (s *Simulator) handleSession(conn net.Conn) {
	defer conn.Close()

	// 简单的 NETCONF 会话处理
	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("NETCONF sim read error: %v\n", err)
			}
			return
		}

		req := string(buf[:n])
		resp := s.handleRequest(req)
		conn.Write([]byte(resp))
	}
}

func (s *Simulator) handleRequest(req string) string {
	// 简单的 RPC 处理
	switch {
	case contains(req, "get-config"):
		return s.buildGetConfigResponse()
	case contains(req, "edit-config"):
		return s.buildEditConfigResponse(req)
	case contains(req, "commit"):
		return "<rpc-reply><ok/></rpc-reply>]]>]]>"
	default:
		return "<rpc-reply><ok/></rpc-reply>]]>]]>"
	}
}

func (s *Simulator) buildGetConfigResponse() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type VLANList struct {
		XMLName xml.Name `xml:"vlans"`
		VLANs   []*VLANConfig `xml:"vlan"`
	}

	vlans := VLANList{VLANs: make([]*VLANConfig, 0, len(s.vlans))}
	for _, v := range s.vlans {
		vlans.VLANs = append(vlans.VLANs, v)
	}

	data, _ := xml.MarshalIndent(vlans, "", "  ")
	return fmt.Sprintf(`<rpc-reply><data>%s</data></rpc-reply>]]>]]>`, string(data))
}

func (s *Simulator) buildEditConfigResponse(req string) string {
	// 简单解析 edit-config 中的 VLAN 配置
	// 实际项目中需要完整的 XML 解析
	return "<rpc-reply><ok/></rpc-reply>]]>]]>"
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
