package actor

import (
	"testing"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/stretchr/testify/assert"
)

func TestManagerActor_AddRemoveDevice(t *testing.T) {
	system := actor.NewActorSystem()
	root := system.Root

	props := actor.PropsFromProducer(func() actor.Actor {
		return NewManagerActor()
	})
	pid := root.Spawn(props)

	// Test add device
	req := &AddDeviceRequest{
		Device: DeviceInfo{
			IP:       "192.168.1.1",
			Port:     830,
			Username: "admin",
			Password: "admin",
		},
	}

	future := root.RequestFuture(pid, req, 15*time.Second)
	res, err := future.Result()
	assert.NoError(t, err)

	addRes, ok := res.(*AddDeviceResponse)
	assert.True(t, ok)
	assert.True(t, addRes.Success)

	// Test list devices
	listReq := &ListDevicesRequest{}
	future = root.RequestFuture(pid, listReq, 15*time.Second)
	res, err = future.Result()
	assert.NoError(t, err)

	listRes, ok := res.(*ListDevicesResponse)
	assert.True(t, ok)
	assert.Len(t, listRes.Devices, 1)
	assert.Equal(t, "192.168.1.1", listRes.Devices[0].IP)

	// Test get device
	getReq := &GetDeviceRequest{IP: "192.168.1.1"}
	future = root.RequestFuture(pid, getReq, 15*time.Second)
	res, err = future.Result()
	assert.NoError(t, err)

	getRes, ok := res.(*GetDeviceResponse)
	assert.True(t, ok)
	assert.True(t, getRes.Exists)
	assert.NotNil(t, getRes.PID)

	// Test remove device
	removeReq := &RemoveDeviceRequest{IP: "192.168.1.1"}
	future = root.RequestFuture(pid, removeReq, 15*time.Second)
	res, err = future.Result()
	assert.NoError(t, err)

	removeRes, ok := res.(*RemoveDeviceResponse)
	assert.True(t, ok)
	assert.True(t, removeRes.Success)

	// Verify removed
	getReq = &GetDeviceRequest{IP: "192.168.1.1"}
	future = root.RequestFuture(pid, getReq, 15*time.Second)
	res, err = future.Result()
	assert.NoError(t, err)

	getRes, ok = res.(*GetDeviceResponse)
	assert.True(t, ok)
	assert.False(t, getRes.Exists)

	root.Poison(pid)
}

func TestManagerActor_DuplicateDevice(t *testing.T) {
	system := actor.NewActorSystem()
	root := system.Root

	props := actor.PropsFromProducer(func() actor.Actor {
		return NewManagerActor()
	})
	pid := root.Spawn(props)

	// Device IP
	ip := "192.168.1.1"

	// If device already exists (from file), remove it first
	getReq := &GetDeviceRequest{IP: ip}
	future := root.RequestFuture(pid, getReq, 15*time.Second)
	res, err := future.Result()
	if err == nil {
		getRes, ok := res.(*GetDeviceResponse)
		if ok && getRes.Exists {
			// Remove it before adding again
			removeReq := &RemoveDeviceRequest{IP: ip}
			future := root.RequestFuture(pid, removeReq, 15*time.Second)
			future.Result()
		}
	}

	// Add first device
	req := &AddDeviceRequest{
		Device: DeviceInfo{
			IP:       ip,
			Port:     830,
			Username: "admin",
			Password: "admin",
		},
	}
	future = root.RequestFuture(pid, req, 15*time.Second)
	res, err = future.Result()
	assert.NoError(t, err)
	addRes := res.(*AddDeviceResponse)
	assert.True(t, addRes.Success)

	// Add duplicate - should fail
	req2 := &AddDeviceRequest{
		Device: DeviceInfo{
			IP:       ip,
			Port:     830,
			Username: "admin",
			Password: "admin",
		},
	}
	future = root.RequestFuture(pid, req2, 15*time.Second)
	res, err = future.Result()
	assert.NoError(t, err)
	addRes2 := res.(*AddDeviceResponse)
	assert.False(t, addRes2.Success)

	root.Poison(pid)
}
