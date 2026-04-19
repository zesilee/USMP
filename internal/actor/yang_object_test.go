package actor

import (
	"testing"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/leezesi/usmp/internal/cache"
	"github.com/stretchr/testify/assert"
)

func TestYangObjectActor_Basic(t *testing.T) {
	cache.InitGlobalCache()
	system := actor.NewActorSystem()
	root := system.Root

	deviceInfo := DeviceInfo{
		IP:       "192.168.1.1",
		Port:     830,
		Username: "admin",
		Password: "admin",
	}

	globalCache := cache.GetGlobalCache()
	props := actor.PropsFromProducer(func() actor.Actor {
		return NewYangObjectActor(deviceInfo, "/interfaces", globalCache)
	})
	pid := root.Spawn(props)

	// Test GetConfig request (will fail because no NETCONF connection in test)
	req := &GetConfigRequest{ForceRefresh: false}
	future := root.RequestFuture(pid, req, 15*time.Second)
	res, err := future.Result()
	// We expect it to respond even if without NETCONF
	assert.NoError(t, err)

	_, ok := res.(*GetConfigResponse)
	assert.True(t, ok)
	// Without NETCONF, success will be false which is expected
	// assert.False(t, getRes.Success)

	root.Poison(pid)
}

func TestYangObjectActor_CacheKey(t *testing.T) {
	deviceInfo := DeviceInfo{IP: "192.168.1.1"}
	actor := NewYangObjectActor(deviceInfo, "/interfaces", nil)

	cacheKey := actor.GetCacheKey()
	expected := "192.168.1.1/interfaces"
	assert.Equal(t, expected, cacheKey)
}
