package netconfsim

// ScenarioConfig configures fault injection for testing.
type ScenarioConfig struct {
	// ErrorOnRPC injects an error for specific RPC operations (keys: "get-config", "edit-config", "commit").
	ErrorOnRPC map[string]error
	// RejectAuth causes all authentication attempts to be rejected.
	RejectAuth bool
}

// NewScenarioConfig creates a new scenario with defaults (no errors).
func NewScenarioConfig() *ScenarioConfig {
	return &ScenarioConfig{
		ErrorOnRPC: make(map[string]error),
	}
}
