package netconf

import (
	"log"
	"sync"
	"time"

	"github.com/leezesi/usmp/internal/types"
)

// SessionManager manages the NETCONF session for a device
type SessionManager struct {
	device        types.DeviceInfo
	client        *Client
	connected     bool
	mu            sync.RWMutex
	reconnecting  bool
	reconnectStop chan struct{}
	retryCount    int
	maxRetries    int
	lastError     error
}

// NewSessionManager creates a new SessionManager
func NewSessionManager(device types.DeviceInfo) *SessionManager {
	return &SessionManager{
		device:        device,
		client:        NewClient(device),
		connected:     false,
		reconnectStop: make(chan struct{}),
		retryCount:    0,
		maxRetries:    5,
	}
}

// Connect establishes a connection with retry
func (m *SessionManager) Connect() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	err := m.client.Connect()
	if err != nil {
		m.connected = false
		m.lastError = err
		return err
	}

	m.connected = true
	m.lastError = nil
	log.Printf("NETCONF session connected to %s:%d", m.device.IP, m.device.Port)
	return nil
}

// Disconnect closes the connection
func (m *SessionManager) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return nil
	}

	err := m.client.Disconnect()
	if err != nil {
		m.connected = false
		return err
	}

	m.connected = false
	return nil
}

// IsConnected returns the current connection state
func (m *SessionManager) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected && m.client.IsConnected()
}

// GetClient returns the underlying NETCONF client
func (m *SessionManager) GetClient() *Client {
	return m.client
}

// CheckConnection verifies the connection is still alive
func (m *SessionManager) CheckConnection() bool {
	if !m.IsConnected() {
		return false
	}

	// A simple get-config with empty filter will tell us if connection is alive
	_, err := m.client.GetConfig("/")
	if err != nil {
		m.mu.Lock()
		m.connected = false
		m.lastError = err
		m.mu.Unlock()
		log.Printf("Connection check failed for %s: %v", m.device.IP, err)
		return false
	}

	return true
}

// LastError returns the last connection error
func (m *SessionManager) LastError() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastError
}

// StartBackgroundReconnect starts the background reconnection loop
func (m *SessionManager) StartBackgroundReconnect() {
	m.mu.Lock()
	if m.reconnecting {
		m.mu.Unlock()
		return
	}
	m.reconnecting = true
	m.mu.Unlock()

	go m.reconnectLoop()
}

// StopBackgroundReconnect stops the background reconnection loop
func (m *SessionManager) StopBackgroundReconnect() {
	m.mu.Lock()
	if m.reconnecting {
		close(m.reconnectStop)
		m.reconnecting = false
	}
	m.mu.Unlock()
}

func (m *SessionManager) reconnectLoop() {
	backoff := 1 * time.Second
	maxBackoff := 5 * time.Minute

	for {
		select {
		case <-m.reconnectStop:
			return
		default:
		}

		if m.IsConnected() {
			// Reset retry count when connected
			m.mu.Lock()
			m.retryCount = 0
			m.mu.Unlock()
			time.Sleep(30 * time.Second)
			continue
		}

		m.mu.Lock()
		if m.retryCount >= m.maxRetries {
			m.mu.Unlock()
			// Stop retrying after max retries, wait longer before next attempt
			time.Sleep(10 * time.Minute)
			continue
		}
		m.retryCount++
		m.mu.Unlock()

		log.Printf("Attempting reconnect to %s (attempt %d/%d)", m.device.IP, m.retryCount, m.maxRetries)
		err := m.Connect()
		if err == nil {
			log.Printf("Successfully reconnected to %s", m.device.IP)
			continue
		}

		// Exponential backoff
		log.Printf("Reconnect failed for %s: %v, retrying in %v", m.device.IP, err, backoff)
		select {
		case <-m.reconnectStop:
			return
		case <-time.After(backoff):
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}
