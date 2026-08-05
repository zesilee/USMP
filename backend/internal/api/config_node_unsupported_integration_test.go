package api

import (
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	netsim "github.com/leezesi/usmp/backend/simulator/netconfsim"
	"github.com/stretchr/testify/assert"
)

// BR-12 端到端（tasks 3.3，T02）：sim 注入 → API 学习 → 快速失败（不打设备）
// → force 重试恢复。全默认导线（supportFromPool/fetchFromDevice），非注入 seam。
func TestGetConfig_NodeUnsupported_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim := netsim.NewSimulator()
	if err := sim.Start(); err != nil {
		t.Fatalf("start sim: %v", err)
	}
	defer sim.Stop()

	sc := netsim.NewScenarioConfig()
	sc.UnknownElementPaths = []string{"vlan/vlans"}
	sim.SetScenario(sc)

	mgr := manager.New()
	defer func() { _ = mgr.GetClientPool().CloseAll() }()
	mgr.GetDeviceStore().Put("sim", client.DeviceConnectionInfo{
		IP: sim.Addr(), Port: sim.Port(), Username: sim.Username(), Password: sim.Password(),
		Protocol: client.ProtocolNETCONF, Timeout: 5 * time.Second,
	})
	h := NewConfigHandler(mgr)

	const path = "/vlan:vlan/vlan:vlans"

	// 1) 首读：设备拒绝 → 当次即结构化 reason + 入集。
	code, _, data := decodeEnvelope(t, getConfigReqQS(h, "sim", path, ""))
	assert.NotEqual(t, 0, code)
	assert.Equal(t, "node-unsupported", data["reason"])

	// 2) 关掉注入后普通读仍被快速失败拒绝 → 证明未打设备（学习记忆生效）。
	sim.SetScenario(netsim.NewScenarioConfig())
	code, _, data = decodeEnvelope(t, getConfigReqQS(h, "sim", path, ""))
	assert.NotEqual(t, 0, code)
	assert.Equal(t, "node-unsupported", data["reason"])

	// 3) force_refresh 真打设备：本次成功 → 清标记 + 返回数据。
	code, _, _ = decodeEnvelope(t, getConfigReqQS(h, "sim", path, "force_refresh=true"))
	assert.Equal(t, 0, code)

	// 4) 恢复后常规读正常（缓存或设备均可）。
	code, _, _ = decodeEnvelope(t, getConfigReqQS(h, "sim", path, ""))
	assert.Equal(t, 0, code)
}
