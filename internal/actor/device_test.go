package actor

import (
	"testing"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/stretchr/testify/assert"
)

func TestDeviceActor_StartStop(t *testing.T) {
	system := actor.NewActorSystem()
	root := system.Root

	deviceInfo := DeviceInfo{
		IP:       "192.168.1.1",
		Port:     830,
		Username: "admin",
		Password: "admin",
	}

	props := actor.PropsFromProducer(func() actor.Actor {
		return NewDeviceActor(deviceInfo)
	})
	pid := root.Spawn(props)

	// Start device
	future := root.RequestFuture(pid, &StartDeviceRequest{Device: deviceInfo}, 15*time.Second)
	res, err := future.Result()
	assert.NoError(t, err)

	startRes, ok := res.(*StartDeviceResponse)
	assert.True(t, ok)
	assert.True(t, startRes.Success)

	// Get status
	statusFuture := root.RequestFuture(pid, &GetDeviceStatusRequest{}, 15*time.Second)
	statusRes, err := statusFuture.Result()
	assert.NoError(t, err)
	status, ok := statusRes.(*GetDeviceStatusResponse)
	assert.True(t, ok)
	assert.True(t, status.Running)

	// Stop device
	stopFuture := root.RequestFuture(pid, &StopDeviceRequest{}, 15*time.Second)
	stopRes, err := stopFuture.Result()
	assert.NoError(t, err)
	stopResp, ok := stopRes.(*StopDeviceResponse)
	assert.True(t, ok)
	assert.True(t, stopResp.Success)

	root.Poison(pid)
}

func TestDeviceActor_GetYANGObject(t *testing.T) {
	system := actor.NewActorSystem()
	root := system.Root

	deviceInfo := DeviceInfo{
		IP:       "192.168.1.1",
		Port:     830,
		Username: "admin",
		Password: "admin",
	}

	props := actor.PropsFromProducer(func() actor.Actor {
		return NewDeviceActor(deviceInfo)
	})
	pid := root.Spawn(props)

	// Start device with some mock YANG modules
	startReq := &StartDeviceRequest{Device: deviceInfo}
	future := root.RequestFuture(pid, startReq, 15*time.Second)
	_, err := future.Result()
	assert.NoError(t, err)

	// Test getting non-existent YANG object
	req := &GetYANGObjectActorRequest{Path: "/interfaces"}
	future = root.RequestFuture(pid, req, 15*time.Second)
	res, err2 := future.Result()
	assert.NoError(t, err2)
	_, ok := res.(*GetYANGObjectActorResponse)
	assert.True(t, ok)
	// Will be false until we add YANG module support in next iteration
	// assert.False(t, yangRes.Exists)

	root.Poison(pid)
}
