package netsim

import (
	"fmt"
	"net"
	"sync"

	"golang.org/x/crypto/ssh"
)

// ServerConfig 配置模拟交换机NETCONF服务器
type ServerConfig struct {
	Addr     string `json:"addr"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// Server 是NETCONF模拟服务器
type Server struct {
	config   *ServerConfig
	scenario *ScenarioConfig

	listener net.Listener
	sshConf  *ssh.ServerConfig

	datastore *Datastore
	sessions  map[*Session]struct{}

	running bool
	mu      sync.RWMutex
}

// New 创建一个新的NETCONF模拟服务器
func New(cfg *ServerConfig) *Server {
	if cfg.Addr == "" {
		cfg.Addr = "127.0.0.1"
	}

	s := &Server{
		config:    cfg,
		datastore: NewDatastore(),
		sessions:  make(map[*Session]struct{}),
		scenario:  &ScenarioConfig{},
	}

	s.sshConf = &ssh.ServerConfig{
		PasswordCallback: s.passwordCallback,
	}

	return s
}

// SetScenario 设置场景配置，用于错误注入
func (s *Server) SetScenario(sc *ScenarioConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scenario = sc
}

// GetScenario 获取当前场景配置
func (s *Server) GetScenario() *ScenarioConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.scenario
}

// Start 启动服务器
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("server already running")
	}

	// 生成host key
	signer, err := generateHostKey()
	if err != nil {
		return fmt.Errorf("generate host key: %w", err)
	}
	s.sshConf.AddHostKey(signer)

	addr := fmt.Sprintf("%s:%d", s.config.Addr, s.config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	s.listener = listener
	s.running = true

	// 接受连接循环
	go s.acceptLoop()

	return nil
}

// Stop 停止服务器
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false

	// 关闭listener
	if s.listener != nil {
		s.listener.Close()
	}

	// 关闭所有session
	for sess := range s.sessions {
		sess.Close()
		delete(s.sessions, sess)
	}

	return nil
}

// IsRunning 返回服务器是否正在运行
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// Port 返回实际监听的端口（当配置Port=0时随机分配）
func (s *Server) Port() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.listener == nil {
		return 0
	}
	return s.listener.Addr().(*net.TCPAddr).Port
}

// GetDatastore 获取数据存储
func (s *Server) GetDatastore() *Datastore {
	return s.datastore
}

// GetRunningConfig 获取当前运行配置
func (s *Server) GetRunningConfig() *Device {
	return s.datastore.GetRunning()
}

// SetRunningConfig 设置运行配置
func (s *Server) SetRunningConfig(dev *Device) {
	s.datastore.SetRunning(dev)
}

func (s *Server) passwordCallback(c ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	sc := s.GetScenario()
	if sc.RejectAuth {
		return nil, fmt.Errorf("authentication rejected by scenario")
	}

	if string(password) != s.config.Password || c.User() != s.config.Username {
		return nil, fmt.Errorf("permission denied")
	}

	return &ssh.Permissions{}, nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.mu.RLock()
			if !s.running {
				return
			}
			s.mu.RUnlock()
			continue
		}

		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	sc := s.GetScenario()
	if sc.ConnectionDelay > 0 {
		// TODO: implement delay
	}

	// 处理SSH握手
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.sshConf)
	if err != nil {
		return
	}

	if sc.DropConnection {
		sshConn.Close()
		return
	}

	// 处理全局请求
	go ssh.DiscardRequests(reqs)

	// 处理新通道
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}

		go s.handleSession(sshConn, channel, requests)
	}
}

func (s *Server) handleSession(conn *ssh.ServerConn, channel ssh.Channel, requests <-chan *ssh.Request) {
	session := NewSession(s, channel)

	s.mu.Lock()
	s.sessions[session] = struct{}{}
	s.mu.Unlock()

	defer func() {
		session.Close()
		s.mu.Lock()
		delete(s.sessions, session)
		s.mu.Unlock()
	}()

	session.Serve()
}
