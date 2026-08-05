package netconfsim

// ScenarioConfig configures fault injection for testing.
type ScenarioConfig struct {
	// ErrorOnRPC injects an error for specific RPC operations (keys: "get", "get-config", "edit-config", "commit").
	ErrorOnRPC map[string]error
	// RejectAuth causes all authentication attempts to be rejected.
	RejectAuth bool
	// DisableConfirmedCommit removes the :confirmed-commit capability from the
	// hello and rejects <commit><confirmed/> RPCs, for capability-missing
	// negative-path tests (NS-07).
	DisableConfirmedCommit bool
	// UnknownElementPaths injects Huawei-style unknown-element rejections (CN-04
	// 地基): get/get-config/edit-config whose body contains one of these
	// local-name chains (e.g. "devm/cards") is refused with error-info-code 313.
	UnknownElementPaths []string
}

// NewScenarioConfig creates a new scenario with defaults (no errors).
func NewScenarioConfig() *ScenarioConfig {
	return &ScenarioConfig{
		ErrorOnRPC: make(map[string]error),
	}
}
