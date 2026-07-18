package client

import (
	"fmt"
	"sync"
	"time"
)

// ClientPool manages a pool of device connections
// Maintains one client per device IP
type ClientPool interface {
	// Get returns a client for the device, creating it if needed
	Get(info DeviceConnectionInfo) (Client, error)
	// Release releases a client back to the pool
	// No-op for this implementation since we keep one client per device
	Release(ip string)
	// CloseAll closes all connections in the pool
	CloseAll() error
	// Stats returns pool statistics
	Stats() PoolStats
}

// PoolStats contains statistics about the client pool
type PoolStats struct {
	// ActiveConnections is the number of active connections
	ActiveConnections int
	// TotalConnections is the total number of connections created
	TotalConnections int
	// Errors is the number of connection errors
	Errors int
}

// DefaultClientPool is the default implementation of ClientPool
type DefaultClientPool struct {
	clients map[string]Client
	mu      sync.RWMutex
	factory ClientFactory
	stats   PoolStats
}

// ClientFactory creates a new client for a device
type ClientFactory func(info DeviceConnectionInfo) (Client, error)

// NewDefaultClientPool creates a new DefaultClientPool
func NewDefaultClientPool(factory ClientFactory) ClientPool {
	return &DefaultClientPool{
		clients: make(map[string]Client),
		factory: factory,
		stats:   PoolStats{},
	}
}

// Get implements ClientPool interface
func (p *DefaultClientPool) Get(info DeviceConnectionInfo) (Client, error) {
	p.mu.RLock()
	client, ok := p.clients[info.IP]
	p.mu.RUnlock()

	if ok && client.IsConnected() {
		return client, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Check again after acquiring write lock
	if client, ok := p.clients[info.IP]; ok && client.IsConnected() {
		return client, nil
	}

	p.stats.TotalConnections++
	client, err := p.factory(info)
	if err != nil {
		p.stats.Errors++
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	p.clients[info.IP] = client
	if client.IsConnected() {
		p.stats.ActiveConnections++
	}

	return client, err
}

// Release implements ClientPool interface
func (p *DefaultClientPool) Release(ip string) {
	// No-op - we keep one client per device permanently
}

// CloseAll implements ClientPool interface
func (p *DefaultClientPool) CloseAll() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var lastErr error
	for ip, client := range p.clients {
		err := client.Close()
		if err != nil {
			lastErr = err
			p.stats.Errors++
		}
		delete(p.clients, ip)
	}

	p.stats.ActiveConnections = 0
	return lastErr
}

// Stats implements ClientPool interface
func (p *DefaultClientPool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stats
}

// DefaultClientFactory is the default client factory that creates clients
// based on the protocol
func DefaultClientFactory(defaultTimeout time.Duration) ClientFactory {
	return func(info DeviceConnectionInfo) (Client, error) {
		if info.Timeout == 0 {
			info.Timeout = defaultTimeout
		}

		switch info.Protocol {
		case ProtocolNETCONF:
			c, err := NewNETCONFClient(info)
			return c, err
		case ProtocolGNMI:
			// gNMI 为规划能力：空壳 client（Get/Set 发空请求的假成功路径）已删
			// （retire-idle-scaffolds），显式错误优于伪装成功（R08）。
			return nil, fmt.Errorf("gNMI 尚未实现（规划能力），设备 %s 请使用 NETCONF", info.IP)
		case ProtocolAUTO:
			// Auto-detect based on port
			if info.Port == 0 {
				info.Port = 830
				info.Protocol = ProtocolNETCONF
				c, err := NewNETCONFClient(info)
				return c, err
			}
			switch info.Port {
			case 830:
				info.Protocol = ProtocolNETCONF
				c, err := NewNETCONFClient(info)
				return c, err
			case 9339:
				// gNMI 端口显式未实现（规划能力，见 ProtocolGNMI 分支）。
				return nil, fmt.Errorf("gNMI 尚未实现（规划能力），设备 %s:9339 请使用 NETCONF(830)", info.IP)
			default:
				// Default to NETCONF
				info.Protocol = ProtocolNETCONF
				c, err := NewNETCONFClient(info)
				return c, err
			}
		default:
			return nil, fmt.Errorf("unsupported protocol: %s", info.Protocol)
		}
	}
}
