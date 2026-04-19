package actor

import (
	"github.com/asynkron/protoactor-go/actor"
	"github.com/leezesi/usmp/internal/types"
)

// === ManagerActor messages ===

// AddDeviceRequest requests to add a new device
type AddDeviceRequest struct {
	Device types.DeviceInfo `json:"device"`
}

// AddDeviceResponse response for AddDeviceRequest
type AddDeviceResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	PID     *actor.PID `json:"pid"`
}

// ListDevicesRequest requests list all devices
type ListDevicesRequest struct{}

// ListDevicesResponse contains all devices
type ListDevicesResponse struct {
	Devices []types.DeviceInfo `json:"devices"`
}

// RemoveDeviceRequest requests to remove a device
type RemoveDeviceRequest struct {
	IP string `json:"ip"`
}

// RemoveDeviceResponse response
type RemoveDeviceResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// GetDeviceRequest requests a device PID by IP
type GetDeviceRequest struct {
	IP string `json:"ip"`
}

// GetDeviceResponse response with device PID
type GetDeviceResponse struct {
	Exists bool       `json:"exists"`
	PID    *actor.PID `json:"pid"`
}

// === DeviceActor messages ===

// StartDeviceRequest starts the device and spawn YANG object actors
type StartDeviceRequest struct {
	Device types.DeviceInfo `json:"device"`
}

// StartDeviceResponse response
type StartDeviceResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// StopDeviceRequest stops the device and terminates all child actors
type StopDeviceRequest struct{}

// StopDeviceResponse response
type StopDeviceResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// GetDeviceStatusRequest gets device connection status
type GetDeviceStatusRequest struct{}

// GetDeviceStatusResponse response
type GetDeviceStatusResponse struct {
	Running  bool    `json:"running"`
	Connected bool   `json:"connected"`
}

// GetYANGObjectActorRequest gets a YANG object actor PID by path
type GetYANGObjectActorRequest struct {
	Path string `json:"path"`
}

// GetYANGObjectActorResponse response
type GetYANGObjectActorResponse struct {
	Exists bool       `json:"exists"`
	PID    *actor.PID `json:"pid"`
}

// === YANG Object Actor messages ===

// GetConfigRequest requests to get configuration from YANG object
type GetConfigRequest struct {
	ForceRefresh bool `json:"force_refresh"`
}

// GetConfigResponse contains the configuration
type GetConfigResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	FromCache bool     `json:"from_cache"`
	Message string    `json:"message"`
}

// SetConfigRequest requests to set configuration to YANG object
type SetConfigRequest struct {
	Data interface{} `json:"data"`
}

// SetConfigResponse response
type SetConfigResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Committed bool `json:"committed"`
}

// === Common types ===
// Re-export from types for backward compatibility
type (
	DeviceInfo     = types.DeviceInfo
	YangModuleInfo = types.YangModuleInfo
)
