package ifm

import (
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
	"github.com/stretchr/testify/assert"
)

// TestDeviceClient_ResolveConn_FromStore: a registered device resolves to the
// stored connection info (credentials included), keyed by a plain DeviceID — the
// production path that fixes the SSH none-auth root cause.
func TestDeviceClient_ResolveConn_FromStore(t *testing.T) {
	store := device.NewStore()
	store.Put("192.168.1.1", client.DeviceConnectionInfo{
		IP: "192.168.1.1", Port: 830, Username: "admin", Password: "admin", Protocol: client.ProtocolNETCONF,
	})
	dc := &deviceClient{resolver: store}

	got := dc.resolveConn("192.168.1.1")
	assert.Equal(t, "192.168.1.1", got.IP)
	assert.Equal(t, "admin", got.Username)
	assert.Equal(t, "admin", got.Password)
	assert.Equal(t, 830, got.Port)
}

// TestDeviceClient_ResolveConn_UnregisteredFallback: an unregistered device
// degrades to an AUTO/no-credential connection (R08, no crash) rather than
// pulling stale creds.
func TestDeviceClient_ResolveConn_UnregisteredFallback(t *testing.T) {
	dc := &deviceClient{resolver: device.NewStore()}

	got := dc.resolveConn("10.9.9.9")
	assert.Equal(t, "10.9.9.9", got.IP)
	assert.Equal(t, client.ProtocolAUTO, got.Protocol)
	assert.Empty(t, got.Username, "未注册设备不应带凭据")
}

// TestDeviceClient_ResolveConn_NilResolver: with no store wired, the DeviceID is
// used as-is for the IP with an AUTO/no-credential connection (no string parsing).
func TestDeviceClient_ResolveConn_NilResolver(t *testing.T) {
	dc := &deviceClient{}

	got := dc.resolveConn("192.168.1.1")
	assert.Equal(t, "192.168.1.1", got.IP)
	assert.Equal(t, client.ProtocolAUTO, got.Protocol)
	assert.Empty(t, got.Username, "无 store 不应凭空产生凭据")
}
