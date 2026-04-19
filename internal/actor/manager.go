package actor

import (
	"encoding/json"
	"log"
	"os"
	"sync"

	"github.com/asynkron/protoactor-go/actor"
)

// ManagerActor is the top-level actor that manages all DeviceActors
type ManagerActor struct {
	devices map[string]*actor.PID // key: device IP
	deviceInfo map[string]DeviceInfo
	mu       sync.RWMutex
}

// NewManagerActor creates a new ManagerActor
func NewManagerActor() *ManagerActor {
	return &ManagerActor{
		devices:    make(map[string]*actor.PID),
		deviceInfo: make(map[string]DeviceInfo),
	}
}

// Receive handles messages for ManagerActor
func (m *ManagerActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		log.Println("ManagerActor started")
		m.loadDevicesFromFile(ctx)
	case *actor.Restarting:
		// Manager itself is restarting
		log.Println("ManagerActor restarting")
	case *AddDeviceRequest:
		m.handleAddDevice(msg, ctx)
	case *ListDevicesRequest:
		m.handleListDevices(msg, ctx)
	case *GetDeviceRequest:
		m.handleGetDevice(msg, ctx)
	case *RemoveDeviceRequest:
		m.handleRemoveDevice(msg, ctx)
	}
}

func (m *ManagerActor) handleAddDevice(msg *AddDeviceRequest, ctx actor.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ip := msg.Device.IP
	if _, exists := m.devices[ip]; exists {
		ctx.Respond(&AddDeviceResponse{
			Success: false,
			Message: "Device already exists: " + ip,
		})
		return
	}

	// Spawn DeviceActor
	deviceProps := actor.PropsFromProducer(func() actor.Actor {
		return NewDeviceActor(msg.Device)
	})
	deviceName := "device-" + ip
	pid, err := ctx.SpawnNamed(deviceProps, deviceName)
	if err != nil {
		ctx.Respond(&AddDeviceResponse{
			Success: false,
			Message: "Failed to spawn DeviceActor: " + err.Error(),
		})
		return
	}

	// Send start request (async, don't wait for response)
	ctx.Send(pid, &StartDeviceRequest{Device: msg.Device})

	m.devices[ip] = pid
	m.deviceInfo[ip] = msg.Device

	ctx.Respond(&AddDeviceResponse{
		Success: true,
		Message: "Device added successfully",
		PID:     pid,
	})
	// Save to file after respond to avoid blocking future
	go m.saveDevicesToFile()
	log.Printf("Device added: %s", ip)
}

func (m *ManagerActor) handleListDevices(msg *ListDevicesRequest, ctx actor.Context) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	devices := make([]DeviceInfo, 0, len(m.deviceInfo))
	for _, info := range m.deviceInfo {
		devices = append(devices, info)
	}

	ctx.Respond(&ListDevicesResponse{
		Devices: devices,
	})
}

func (m *ManagerActor) handleGetDevice(msg *GetDeviceRequest, ctx actor.Context) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pid, exists := m.devices[msg.IP]
	ctx.Respond(&GetDeviceResponse{
		Exists: exists,
		PID:    pid,
	})
}

func (m *ManagerActor) handleRemoveDevice(msg *RemoveDeviceRequest, ctx actor.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ip := msg.IP
	pid, exists := m.devices[ip]
	if !exists {
		ctx.Respond(&RemoveDeviceResponse{
			Success: false,
			Message: "Device not found: " + ip,
		})
		return
	}

	// Stop the DeviceActor
	ctx.Poison(pid)
	delete(m.devices, ip)
	delete(m.deviceInfo, ip)

	// Save to file after respond to avoid deadlock (we already hold the lock)
	go m.saveDevicesToFile()

	ctx.Respond(&RemoveDeviceResponse{
		Success: true,
		Message: "Device removed successfully",
	})
	log.Printf("Device removed: %s", ip)
}

func (m *ManagerActor) loadDevicesFromFile(ctx actor.Context) {
	configDir := "internal/config"
	devicesFile := configDir + "/devices.json"

	// Create directory if not exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			log.Printf("Failed to create config directory: %v", err)
			return
		}
	}

	// Check if file exists
	if _, err := os.Stat(devicesFile); os.IsNotExist(err) {
		// Create empty file
		emptyJSON := "[]\n"
		os.WriteFile(devicesFile, []byte(emptyJSON), 0644)
		log.Println("No devices file found, created empty file")
		return
	}

	// Read and parse
	data, err := os.ReadFile(devicesFile)
	if err != nil {
		log.Printf("Failed to read devices file: %v", err)
		return
	}

	var devices []DeviceInfo
	if err := json.Unmarshal(data, &devices); err != nil {
		log.Printf("Failed to parse devices file: %v", err)
		return
	}

	// Add all devices
	for _, dev := range devices {
		deviceProps := actor.PropsFromProducer(func() actor.Actor {
			return NewDeviceActor(dev)
		})
		deviceName := "device-" + dev.IP
		pid, err := ctx.SpawnNamed(deviceProps, deviceName)
		if err != nil {
			log.Printf("Failed to spawn DeviceActor for %s: %v", dev.IP, err)
			continue
		}

		// Send start request (async, don't wait for response)
		ctx.Send(pid, &StartDeviceRequest{Device: dev})
		m.devices[dev.IP] = pid
		m.deviceInfo[dev.IP] = dev
	}

	log.Printf("Loaded %d devices from file", len(devices))
}

func (m *ManagerActor) saveDevicesToFile() {
	configDir := "internal/config"
	devicesFile := configDir + "/devices.json"

	m.mu.RLock()
	devices := make([]DeviceInfo, 0, len(m.deviceInfo))
	for _, info := range m.deviceInfo {
		devices = append(devices, info)
	}
	m.mu.RUnlock()

	data, err := json.MarshalIndent(devices, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal devices: %v", err)
		return
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Printf("Failed to create config directory: %v", err)
		return
	}

	if err := os.WriteFile(devicesFile, data, 0644); err != nil {
		log.Printf("Failed to write devices file: %v", err)
	}
}
