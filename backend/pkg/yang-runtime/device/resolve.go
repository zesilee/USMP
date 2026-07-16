package device

import (
	"log"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
)

// ResolveConn resolves connection info for deviceID via the shared DeviceStore
// (DS-06 单一建连解析口). Registered devices return the stored full info;
// unregistered devices (or a nil store, legacy path) degrade to an
// AUTO/no-credential connection so authentication fails cleanly (R08) rather
// than crash. Protocol must be set or the client factory hits its
// "unsupported protocol" default branch.
func ResolveConn(store Store, deviceID string) (client.DeviceConnectionInfo, bool) {
	if store != nil {
		if info, ok := store.Get(deviceID); ok {
			return info, true
		}
	}
	log.Printf("device: %q not registered in DeviceStore; using AUTO/no-credential connection", deviceID)
	return client.DeviceConnectionInfo{IP: deviceID, Protocol: client.ProtocolAUTO}, false
}
