package api

import (
	"context"
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
	netsim "github.com/leezesi/usmp/backend/simulator/netconfsim"
	"github.com/stretchr/testify/assert"
)

// BR-01（modified）：GET /config 读路径改发 <get>（WithStateData），回读含
// config=false 状态子树时原样带出 RFC7951 结构；无状态子树时行为与改动前等值。

// optCaptureClient records the GetOptions its Get call resolves, then returns
// canned XML like xmlClient.
type optCaptureClient struct {
	xml  string
	opts client.GetOptions
}

func (c *optCaptureClient) Get(_ context.Context, _ string, opts ...client.GetOption) (*client.GetResult, error) {
	resolved := client.GetOptions{}
	for _, o := range opts {
		o.Apply(&resolved)
	}
	c.opts = resolved
	return &client.GetResult{Data: []byte(c.xml)}, nil
}
func (c *optCaptureClient) Set(context.Context, []client.Change, ...client.SetOption) (*client.SetResult, error) {
	return nil, nil
}
func (c *optCaptureClient) Subscribe(context.Context, string, func(client.Notification)) error {
	return nil
}
func (c *optCaptureClient) Close() error                           { return nil }
func (c *optCaptureClient) IsConnected() bool                      { return true }
func (c *optCaptureClient) DiscardCandidate(context.Context) error { return nil }

// 读路径必须以 WithStateData 发起（<get>），否则设备不会返回状态数据。
func TestFetchFromDevice_RequestsStateData(t *testing.T) {
	cc := &optCaptureClient{xml: `<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm"><interfaces/></ifm>`}
	h := NewConfigHandler(fakePoolManager{pool: &fakePool{client: cc}})

	_, err := h.fetchFromDevice(context.Background(), "192.168.1.1", "/ifm:ifm/ifm:interfaces")
	assert.NoError(t, err)
	assert.True(t, cc.opts.IncludeState, "读路径应携 WithStateData（发 <get> 才能带回 config=false 数据）")
}

// 回读含状态子树：RFC7951 结构原样带出（前端只读控件由此回显）。
func TestFetchFromDevice_ParsesStateSubtree(t *testing.T) {
	xml := `<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm"><interfaces>` +
		`<interface><name>GE0/0/1</name><mtu>1500</mtu>` +
		`<dynamic><oper-status>1</oper-status><mac-address>00:aa:bb:cc:dd:ee</mac-address></dynamic>` +
		`</interface></interfaces></ifm>`
	p := &fakePool{client: &xmlClient{xml: xml}}
	h := NewConfigHandler(fakePoolManager{pool: p})

	got, err := h.fetchFromDevice(context.Background(), "192.168.1.1", "/ifm:ifm/ifm:interfaces")
	assert.NoError(t, err)

	m, ok := got.(map[string]interface{})
	assert.True(t, ok, "回读应为 RFC7951 map")
	if !ok {
		return
	}
	list, _ := m["interface"].([]interface{})
	assert.Len(t, list, 1)
	first, _ := list[0].(map[string]interface{})
	dyn, ok := first["dynamic"].(map[string]interface{})
	assert.True(t, ok, "config=false 状态容器 dynamic 应出现在回读结构中")
	if !ok {
		return
	}
	assert.Equal(t, "00:aa:bb:cc:dd:ee", dyn["mac-address"], "状态叶 mac-address 应原样带出")
	assert.NotEmpty(t, dyn["oper-status"], "状态叶 oper-status 应带出")
	assert.Equal(t, float64(1500), first["mtu"], "配置叶不受状态合并影响")
}

// B2 端到端（3.3）：sim（配置+状态 seed）→ 真实 ClientPool/DeviceStore →
// fetchFromDevice → RFC7951 结构含状态字段。IFM dynamic 与 VLAN status 两场景。
func TestConfigStateRead_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	sim := netsim.NewSimulator()
	if err := sim.Start(); err != nil {
		t.Fatalf("start sim: %v", err)
	}
	defer sim.Stop()

	sim.SetRunningConfigXML([]byte(`<config>` +
		`<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm"><interfaces>` +
		`<interface><name>GE0/0/1</name><mtu>1500</mtu></interface></interfaces></ifm>` +
		`<vlan xmlns="urn:huawei:params:xml:ns:yang:huawei-vlan"><vlans>` +
		`<vlan><id>100</id><name>office</name></vlan></vlans></vlan>` +
		`</config>`))
	if err := sim.SetStateDataXML([]byte(
		`<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm"><interfaces>` +
			`<interface><name>GE0/0/1</name>` +
			`<dynamic><oper-status>1</oper-status><mac-address>00:aa:bb:cc:dd:ee</mac-address></dynamic>` +
			`</interface></interfaces></ifm>` +
			`<vlan xmlns="urn:huawei:params:xml:ns:yang:huawei-vlan"><vlans>` +
			`<vlan><id>100</id><statistics><inbound-packets>12345</inbound-packets><inbound-bytes>67890</inbound-bytes></statistics></vlan></vlans></vlan>`)); err != nil {
		t.Fatalf("SetStateDataXML: %v", err)
	}

	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()
	ds := device.NewStore()
	ds.Put("sim", client.DeviceConnectionInfo{
		IP: sim.Addr(), Port: sim.Port(), Username: sim.Username(), Password: sim.Password(), Protocol: client.ProtocolNETCONF,
	})
	h := NewConfigHandler(fakePoolManager{pool: pool, store: ds})
	ctx := context.Background()

	// IFM：dynamic 状态容器随配置一起回读
	got, err := h.fetchFromDevice(ctx, "sim", "/ifm:ifm/ifm:interfaces")
	assert.NoError(t, err)
	m, ok := got.(map[string]interface{})
	assert.True(t, ok, "ifm 回读应为 RFC7951 map")
	if ok {
		list, _ := m["interface"].([]interface{})
		assert.Len(t, list, 1)
		first, _ := list[0].(map[string]interface{})
		dyn, hasDyn := first["dynamic"].(map[string]interface{})
		assert.True(t, hasDyn, "端到端回读应含 dynamic 状态容器")
		if hasDyn {
			assert.Equal(t, "00:aa:bb:cc:dd:ee", dyn["mac-address"])
			assert.NotEmpty(t, dyn["oper-status"])
		}
		// 配置叶保留——<config> 壳合并层级错误时状态树会被当独立树返回、mtu 丢失
		assert.Equal(t, float64(1500), first["mtu"], "配置叶不得因状态合并丢失")
	}

	// VLAN：statistics 状态容器（config false 计数器）随配置一起回读。
	// 注意 RFC7951 语义：uint64 序列化为字符串。
	got, err = h.fetchFromDevice(ctx, "sim", "/vlan:vlan/vlan:vlans")
	assert.NoError(t, err)
	m, ok = got.(map[string]interface{})
	assert.True(t, ok, "vlan 回读应为 RFC7951 map")
	if ok {
		list, _ := m["vlan"].([]interface{})
		assert.Len(t, list, 1)
		first, _ := list[0].(map[string]interface{})
		assert.Equal(t, "office", first["name"], "配置叶不受状态合并影响")
		stats, hasStats := first["statistics"].(map[string]interface{})
		assert.True(t, hasStats, "VLAN statistics 状态容器应回读带出")
		if hasStats {
			assert.Equal(t, "12345", stats["inbound-packets"], "uint64 状态叶按 RFC7951 为字符串")
		}
	}
}

// 设备无状态数据：回读结构与改动前等值，不构造空状态占位。
func TestFetchFromDevice_NoStateDataUnchanged(t *testing.T) {
	xml := `<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm"><interfaces>` +
		`<interface><name>GE0/0/1</name><mtu>1500</mtu></interface></interfaces></ifm>`
	p := &fakePool{client: &xmlClient{xml: xml}}
	h := NewConfigHandler(fakePoolManager{pool: p})

	got, err := h.fetchFromDevice(context.Background(), "192.168.1.1", "/ifm:ifm/ifm:interfaces")
	assert.NoError(t, err)

	m, ok := got.(map[string]interface{})
	assert.True(t, ok)
	if !ok {
		return
	}
	list, _ := m["interface"].([]interface{})
	assert.Len(t, list, 1)
	first, _ := list[0].(map[string]interface{})
	_, hasDyn := first["dynamic"]
	assert.False(t, hasDyn, "无状态数据时不得构造空 dynamic 占位")
	assert.Equal(t, "GE0/0/1", first["name"])
}
