package api

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/leezesi/usmp/backend/internal/generated/native/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	netsim "github.com/leezesi/usmp/backend/simulator/netconfsim"
)

// newChangesetSimStack：真实 manager + 模拟网元 + 真 TxCoordinator（CS-04 B2）。
// 设备注册进 manager 自带 DeviceStore，handler 走生产构造（NewChangesetHandler）。
func newChangesetSimStack(t *testing.T) (*netsim.Simulator, *ChangesetHandler, manager.Manager, string) {
	t.Helper()
	sim := netsim.NewSimulator()
	if err := sim.Start(); err != nil {
		t.Fatalf("start sim: %v", err)
	}
	t.Cleanup(sim.Stop)

	mgr := manager.New()
	deviceID := "sim-changeset"
	mgr.GetDeviceStore().Put(deviceID, client.DeviceConnectionInfo{
		IP: sim.Addr(), Port: sim.Port(),
		Username: sim.Username(), Password: sim.Password(),
		Protocol: client.ProtocolNETCONF,
	})
	t.Cleanup(func() { _ = mgr.GetClientPool().CloseAll() })
	return sim, NewChangesetHandler(mgr), mgr, deviceID
}

// CS-04 场景「跨模块原子提交成功」：vlan+ifm 同变更集单次提交，两模块同时生效，
// desired 落地、审计逐条目。
func TestChangesetCommit_Integration_AtomicCrossModule(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim, h, mgr, dev := newChangesetSimStack(t)

	body := `{"device":"` + dev + `","entries":[
		{"op":"create","path":"/vlan:vlan/vlan:vlans","payload":{"vlan":[{"id":50,"name":"batch50","description":"batch commit"}]}},
		{"op":"update","path":"/ifm:ifm/ifm:interfaces","payload":{"interface":[{"name":"GE0/0/1","description":"batch-if"}]}}
	]}`
	w := commitReq(h, body, "")
	assert.Equal(t, 0, envelopeCode(t, w), "body: %s", w.Body.String())

	vlans := sim.RunningHuaweiVLANs()
	assert.Contains(t, vlans, uint16(50))
	ifs := sim.RunningHuaweiInterfaces()
	if assert.Contains(t, ifs, "GE0/0/1") {
		assert.Equal(t, "batch-if", ifs["GE0/0/1"].Description)
	}

	// desired 两锚点均落地
	sv, err := mgr.GetConfigStore().Get(dev, vlanAnchor)
	assert.NoError(t, err)
	assert.Contains(t, sv.(*huawei.HuaweiVlan_Vlan_Vlans).Vlan, uint16(50))
	si, err := mgr.GetConfigStore().Get(dev, "/ifm:ifm/ifm:interfaces")
	assert.NoError(t, err)
	assert.Contains(t, si.(*huawei.HuaweiIfm_Ifm_Interfaces).Interface, "GE0/0/1")

	assert.Len(t, mgr.GetAuditStore().List(), 2, "审计逐条目")
}

// CS-04 场景「中途失败整体回退」：第 2 条 edit-config 被设备拒绝（删除不存在
// 条目 → data-missing），candidate 整体 discard——第 1 条也不得生效。
func TestChangesetCommit_Integration_MidFailureRollsBack(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim, h, mgr, dev := newChangesetSimStack(t)

	body := `{"device":"` + dev + `","entries":[
		{"op":"create","path":"/vlan:vlan/vlan:vlans","payload":{"vlan":[{"id":90,"name":"doomed"}]}},
		{"op":"delete","path":"/vlan:vlan/vlan:vlans","key":"999"}
	]}`
	w := commitReq(h, body, "")
	assert.Equal(t, 502, envelopeCode(t, w), "body: %s", w.Body.String())

	assert.NotContains(t, sim.RunningHuaweiVLANs(), uint16(90), "整体回退：先入 candidate 的条目不得生效")
	stored, _ := mgr.GetConfigStore().Get(dev, vlanAnchor)
	assert.Nil(t, stored, "失败不得写 desired")
	assert.Empty(t, mgr.GetAuditStore().List(), "失败不得写成功审计")
}

// CS-04 场景「含删除条目的提交」：先建两条，再批量删一条——设备与 desired
// 同步移除且不误删同表其它键。
func TestChangesetCommit_Integration_DeleteEntry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim, h, mgr, dev := newChangesetSimStack(t)

	seed := `{"device":"` + dev + `","entries":[{"op":"create","path":"/vlan:vlan/vlan:vlans","payload":{"vlan":[{"id":70,"name":"del-me"},{"id":71,"name":"keep-me"}]}}]}`
	assert.Equal(t, 0, envelopeCode(t, commitReq(h, seed, "")))
	assert.Contains(t, sim.RunningHuaweiVLANs(), uint16(70))

	del := `{"device":"` + dev + `","entries":[{"op":"delete","path":"/vlan:vlan/vlan:vlans","key":"70"}]}`
	assert.Equal(t, 0, envelopeCode(t, commitReq(h, del, "")))

	vlans := sim.RunningHuaweiVLANs()
	assert.NotContains(t, vlans, uint16(70))
	assert.Contains(t, vlans, uint16(71), "不得误删同表其它键")
	stored, err := mgr.GetConfigStore().Get(dev, vlanAnchor)
	assert.NoError(t, err)
	v := stored.(*huawei.HuaweiVlan_Vlan_Vlans)
	assert.NotContains(t, v.Vlan, uint16(70))
	assert.Contains(t, v.Vlan, uint16(71))
}

// CS-05 场景「模拟网元端到端删除叶」：清除 description → 设备该叶消失、
// 条目其余叶保持。
func TestChangesetCommit_Integration_ClearedLeafEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim, h, _, dev := newChangesetSimStack(t)

	seed := `{"device":"` + dev + `","entries":[{"op":"create","path":"/vlan:vlan/vlan:vlans","payload":{"vlan":[{"id":80,"name":"keep-name","description":"clear-me"}]}}]}`
	assert.Equal(t, 0, envelopeCode(t, commitReq(h, seed, "")))
	full := sim.RunningHuaweiVLANsFull()
	if assert.Contains(t, full, uint16(80)) {
		assert.Equal(t, "clear-me", full[uint16(80)].Description)
	}

	clear := `{"device":"` + dev + `","entries":[{"op":"update","path":"/vlan:vlan/vlan:vlans","payload":{"vlan":[{"id":80}]},"cleared":["description"]}]}`
	assert.Equal(t, 0, envelopeCode(t, commitReq(h, clear, "")), "clear commit")

	full = sim.RunningHuaweiVLANsFull()
	if assert.Contains(t, full, uint16(80)) {
		assert.Empty(t, full[uint16(80)].Description, "cleared 叶必须从设备删除")
		assert.Equal(t, "keep-name", full[uint16(80)].Name, "其余叶必须保持")
	}
}

// CS-01 场景「预览不产生副作用」对真机通道：preview（基线=实时回读）后设备
// running 与 desired 均不变。
func TestChangesetPreview_Integration_NoDeviceWrite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim, h, mgr, dev := newChangesetSimStack(t)

	seed := `{"device":"` + dev + `","entries":[{"op":"create","path":"/vlan:vlan/vlan:vlans","payload":{"vlan":[{"id":60,"name":"base"}]}}]}`
	assert.Equal(t, 0, envelopeCode(t, commitReq(h, seed, "")))
	before := sim.RunningHuaweiVLANs()

	// 清空 desired 与缓存，强制预览走「实时回读」基线（三级链末级）
	mgr.GetRunningCache().InvalidatePrefix(dev + "|")

	preview := `{"device":"` + dev + `","entries":[{"op":"update","path":"/vlan:vlan/vlan:vlans","payload":{"vlan":[{"id":60,"description":"preview-only"}]}}]}`
	d := decodePreview(t, previewReq(h, preview))
	if assert.Len(t, d.Entries, 1) {
		assert.Contains(t, d.Entries[0].ForwardXML, "preview-only")
	}
	assert.Equal(t, before, sim.RunningHuaweiVLANs(), "预览不得改设备 running")
}
