package actor

import (
	"log"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/leezesi/usmp/internal/cache"
)

// DeviceActor represents a single network switch device
type DeviceActor struct {
	deviceInfo DeviceInfo
	running    bool
	connected  bool
	yangActors map[string]*actor.PID // key: YANG path
	cache      *cache.TTLLRUCache
}

// NewDeviceActor creates a new DeviceActor
func NewDeviceActor(info DeviceInfo) *DeviceActor {
	return &DeviceActor{
		deviceInfo: info,
		running:    false,
		connected:  false,
		yangActors: make(map[string]*actor.PID),
		cache:      cache.GetGlobalCache(),
	}
}

// Receive handles messages for DeviceActor
func (d *DeviceActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		log.Printf("DeviceActor started for %s", d.deviceInfo.IP)
	case *actor.Restarting:
		log.Printf("DeviceActor restarting for %s, cleaning up child actors", d.deviceInfo.IP)
		// Clean up all child actors before restart
		for _, pid := range d.yangActors {
			ctx.Poison(pid)
		}
		d.yangActors = make(map[string]*actor.PID)
		d.running = false
	case *StartDeviceRequest:
		d.handleStart(msg, ctx)
	case *StopDeviceRequest:
		d.handleStop(msg, ctx)
	case *GetDeviceStatusRequest:
		d.handleGetStatus(msg, ctx)
	case *GetYANGObjectActorRequest:
		d.handleGetYANGObject(msg, ctx)
	}
}

func (d *DeviceActor) handleStart(msg *StartDeviceRequest, ctx actor.Context) {
	d.deviceInfo = msg.Device
	d.running = true

	// TODO: Initialize NETCONF client (done in phase 4)
	d.connected = false

	// Get list of supported YANG modules from configuration
	// For now, we'll support common openconfig modules
	// This will be dynamically loaded from YANG model later
	supportedModules := []YangModuleInfo{
		{Name: "Interfaces", Path: "/interfaces", Description: "Network interfaces"},
		{Name: "VLANs", Path: "/vlans", Description: "VLAN configuration"},
		{Name: "System", Path: "/system", Description: "System information"},
	}

	// Spawn YANG Object Actor for each module
	for _, mod := range supportedModules {
		yangProps := actor.PropsFromProducer(func() actor.Actor {
			return NewYangObjectActor(d.deviceInfo, mod.Path, d.cache)
		})
		yangName := "yang-" + d.deviceInfo.IP + "-" + mod.Path
		pid, err := ctx.SpawnNamed(yangProps, yangName)
		if err != nil {
			log.Printf("Failed to spawn YANG actor for %s: %v", mod.Path, err)
			continue
		}
		d.yangActors[mod.Path] = pid
	}

	log.Printf("Device %s started with %d YANG modules", d.deviceInfo.IP, len(d.yangActors))
	ctx.Respond(&StartDeviceResponse{
		Success: true,
		Message: "Device started successfully",
	})
}

func (d *DeviceActor) handleStop(msg *StopDeviceRequest, ctx actor.Context) {
	// Stop all YANG child actors
	for _, pid := range d.yangActors {
		ctx.Poison(pid)
	}

	d.yangActors = make(map[string]*actor.PID)
	d.running = false
	d.connected = false

	log.Printf("Device %s stopped", d.deviceInfo.IP)
	ctx.Respond(&StopDeviceResponse{
		Success: true,
		Message: "Device stopped successfully",
	})
}

func (d *DeviceActor) handleGetStatus(msg *GetDeviceStatusRequest, ctx actor.Context) {
	ctx.Respond(&GetDeviceStatusResponse{
		Running:  d.running,
		Connected: d.connected,
	})
}

func (d *DeviceActor) handleGetYANGObject(msg *GetYANGObjectActorRequest, ctx actor.Context) {
	pid, exists := d.yangActors[msg.Path]
	ctx.Respond(&GetYANGObjectActorResponse{
		Exists: exists,
		PID:    pid,
	})
}
