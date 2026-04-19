package actor

import (
	"log"
	"strings"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/leezesi/usmp/internal/cache"
	"github.com/leezesi/usmp/internal/netconf"
)

// YangObjectActor is the base class for a YANG configuration object actor
type YangObjectActor struct {
	device    DeviceInfo
	yangPath  string
	cache     *cache.TTLLRUCache
	config    interface{}        // Current configuration (ygot struct)
	session   *netconf.SessionManager // NETCONF session manager
}

// NewYangObjectActor creates a new YANG Object Actor
func NewYangObjectActor(device DeviceInfo, yangPath string, cache *cache.TTLLRUCache) *YangObjectActor {
	return &YangObjectActor{
		device:   device,
		yangPath: yangPath,
		cache:    cache,
		config:   nil,
		session:  netconf.NewSessionManager(device),
	}
}

// GetCacheKey generates the cache key for this YANG object
func (y *YangObjectActor) GetCacheKey() string {
	// Normalize path to avoid duplicates
	path := strings.ReplaceAll(y.yangPath, "/", "-")
	if strings.HasPrefix(path, "-") {
		path = path[1:]
	}
	return y.device.IP + "/" + path
}

// Receive handles messages for YangObjectActor
func (y *YangObjectActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		log.Printf("YangObjectActor started for %s %s", y.device.IP, y.yangPath)
	case *GetConfigRequest:
		y.handleGetConfig(msg, ctx)
	case *SetConfigRequest:
		y.handleSetConfig(msg, ctx)
	}
}

func (y *YangObjectActor) handleGetConfig(msg *GetConfigRequest, ctx actor.Context) {
	cacheKey := y.GetCacheKey()

	// Check cache first, unless force refresh
	if !msg.ForceRefresh {
		if cached, ok := y.cache.Get(cacheKey); ok {
			ctx.Respond(&GetConfigResponse{
				Success:    true,
				Data:       cached,
				FromCache:  true,
				Message:    "Config from cache",
			})
			return
		}
	}

	// Check if connected, connect if not
	if !y.session.IsConnected() {
		err := y.session.Connect()
		if err != nil {
			ctx.Respond(&GetConfigResponse{
				Success:    false,
				Data:       nil,
				FromCache:  false,
				Message:    "Failed to connect: " + err.Error(),
			})
			return
		}
	}

	// Fetch from NETCONF
	xmlData, err := y.session.GetClient().GetConfig(y.yangPath)
	if err != nil {
		ctx.Respond(&GetConfigResponse{
			Success:    false,
			Data:       nil,
			FromCache:  false,
			Message:    "NETCONF get-config failed: " + err.Error(),
		})
		return
	}

	// TODO: Decode XML to ygot struct when we have generated code
	// For now, cache the raw XML
	y.config = xmlData
	y.cache.Set(cacheKey, xmlData)

	ctx.Respond(&GetConfigResponse{
		Success:    true,
		Data:       xmlData,
		FromCache:  false,
		Message:    "Config from NETCONF",
	})
}

func (y *YangObjectActor) handleSetConfig(msg *SetConfigRequest, ctx actor.Context) {
	cacheKey := y.GetCacheKey()
	y.config = msg.Data

	// Check if connected, connect if not
	if !y.session.IsConnected() {
		err := y.session.Connect()
		if err != nil {
			ctx.Respond(&SetConfigResponse{
				Success:  false,
				Message:  "Failed to connect: " + err.Error(),
				Committed: false,
			})
			return
		}
	}

	// Apply configuration via NETCONF
	err := y.session.GetClient().EditConfigAndCommit(y.yangPath, msg.Data)
	if err != nil {
		ctx.Respond(&SetConfigResponse{
			Success:  false,
			Message:  "NETCONF commit failed: " + err.Error(),
			Committed: false,
		})
		return
	}

	// Invalidate cache after successful commit so next read will fetch fresh
	y.cache.Invalidate(cacheKey)

	ctx.Respond(&SetConfigResponse{
		Success:  true,
		Message:  "Configuration committed successfully",
		Committed: true,
	})
}

// GetConfig returns the current cached configuration
func (y *YangObjectActor) GetConfig() interface{} {
	return y.config
}

// SetConfig sets the configuration (used for testing)
func (y *YangObjectActor) SetConfig(config interface{}) {
	y.config = config
}
