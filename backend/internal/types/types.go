package types

// DeviceInfo stores device metadata
type DeviceInfo struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// YangModuleInfo describes a supported YANG module
type YangModuleInfo struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description"`
	Type        string `json:"type"`
}
