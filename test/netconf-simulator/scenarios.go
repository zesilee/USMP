package netsim

import "time"

// ScenarioConfig configures failure/injection scenarios
type ScenarioConfig struct {
	// RejectAuth causes all authentication attempts to be rejected
	RejectAuth bool
	// DropConnection drops the connection immediately after acceptance
	DropConnection bool
	// ConnectionDelay delays connection acceptance by this duration
	ConnectionDelay time.Duration
	// ResponseDelay delays each RPC response by this duration
	ResponseDelay time.Duration
	// ErrorOnRPC returns an error for the specified RPC operation name
	ErrorOnRPC map[string]error
	// TimeoutOnRPC does not respond to the specified RPC operation
	TimeoutOnRPC map[string]bool
	// PartialSuccess indicates a partial success response
	PartialSuccess bool
}

// NewScenarioConfig creates a new empty scenario config
func NewScenarioConfig() *ScenarioConfig {
	return &ScenarioConfig{
		ErrorOnRPC:   make(map[string]error),
		TimeoutOnRPC: make(map[string]bool),
	}
}

// ShouldRejectAuth checks if authentication should be rejected
func (c *ScenarioConfig) ShouldRejectAuth() bool {
	if c == nil {
		return false
	}
	return c.RejectAuth
}

// GetErrorForRPC returns an error for the specified RPC if configured
func (c *ScenarioConfig) GetErrorForRPC(rpc string) (error, bool) {
	if c == nil || c.ErrorOnRPC == nil {
		return nil, false
	}
	err, ok := c.ErrorOnRPC[rpc]
	return err, ok
}

// IsTimeoutForRPC checks if RPC should timeout
func (c *ScenarioConfig) IsTimeoutForRPC(rpc string) bool {
	if c == nil || c.TimeoutOnRPC == nil {
		return false
	}
	return c.TimeoutOnRPC[rpc]
}

// GetResponseDelay returns the configured response delay
func (c *ScenarioConfig) GetResponseDelay() time.Duration {
	if c == nil {
		return 0
	}
	return c.ResponseDelay
}
