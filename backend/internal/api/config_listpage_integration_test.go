package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	netsim "github.com/leezesi/usmp/backend/simulator/netconfsim"
	"github.com/stretchr/testify/assert"
)

// B2 集成（tasks 3.1/3.2）：netconfsim 注入千行级 FIB 状态子树（五键复合 route
// 挂在三键 unicast-af 内）→ 经 GET /config 分页读取端到端验证：首读回填快照、
// 翻页零设备往返、force_refresh 重拉、过滤命中、无参形状回归。

// fibStateXML 造 1 个 unicast-af（三键）内含 n 条 route（五键复合）的状态树。
func fibStateXML(n int) []byte {
	var b strings.Builder
	b.WriteString(`<fib xmlns="urn:huawei:yang:huawei-fib"><unicast-afs><unicast-af>`)
	b.WriteString(`<vrf-name>_public_</vrf-name><af-type>ipv4unicast</af-type><position>0</position><routes>`)
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, `<route><destination>10.0.%d.%d</destination><mask>32</mask>`+
			`<nexthop>192.168.1.%d</nexthop><if-name>GE1/0/%d</if-name><tunnel-id>0</tunnel-id></route>`,
			i/256, i%256, i%254+1, i%4)
	}
	b.WriteString(`</routes></unicast-af></unicast-afs></fib>`)
	return []byte(b.String())
}

// startFibSim 起 sim（千行 FIB 状态）+ 真实 ClientPool/DeviceStore 的 handler，
// fetchState 包一层计数器（每次调用 = 一次真实 <get> 到 sim）。
func startFibSim(t *testing.T, n int) (*ConfigHandler, *int, func()) {
	t.Helper()
	sim := netsim.NewSimulator()
	if err := sim.Start(); err != nil {
		t.Fatalf("start sim: %v", err)
	}
	// 注意：不要包 <config> 壳——SetRunningConfigXML 以传入 XML 为 running 树根，
	// 带壳时 <get-config> 的 subtree filter 匹配不到（带壳的存量用例都只走 <get>）。
	sim.SetRunningConfigXML([]byte(`<vlan xmlns="urn:huawei:yang:huawei-vlan"><vlans>` +
		vlanEntriesXML(500) + `</vlans></vlan>`))
	if err := sim.SetStateDataXML(fibStateXML(n)); err != nil {
		t.Fatalf("SetStateDataXML: %v", err)
	}

	pool := client.NewDefaultClientPool(client.DefaultClientFactory(10 * time.Second))
	ds := device.NewStore()
	ds.Put("sim", client.DeviceConnectionInfo{
		IP: sim.Addr(), Port: sim.Port(), Username: sim.Username(), Password: sim.Password(), Protocol: client.ProtocolNETCONF,
	})
	// 嵌入真 manager：GetConfig 需要真 RunningCache（fakePoolManager 缺省嵌 nil）。
	h := NewConfigHandler(fakePoolManager{Manager: manager.New(), pool: pool, store: ds})
	deviceCalls := 0
	realFetchState := h.fetchState
	h.fetchState = func(ctx context.Context, ip, path string) (interface{}, error) {
		deviceCalls++
		return realFetchState(ctx, ip, path)
	}
	realFetch := h.fetch
	h.fetch = func(ctx context.Context, ip, path string) (interface{}, error) {
		deviceCalls++
		return realFetch(ctx, ip, path)
	}
	cleanup := func() {
		pool.CloseAll()
		sim.Stop()
	}
	return h, &deviceCalls, cleanup
}

// vlanEntriesXML 造 n 条 vlan 配置行（任务 3.2 配置类大表）。
func vlanEntriesXML(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, `<vlan><id>%d</id><name>v%d</name></vlan>`, i+2, i+2)
	}
	return b.String()
}

const fibRoutesPath = "/fib:fib/fib:unicast-afs/fib:unicast-af[vrf-name=_public_][af-type=ipv4unicast][position=0]/fib:routes"

// 任务 3.1：FIB 千行状态 list 端到端——快照分页四步曲。
func TestFibStatePagination_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	h, deviceCalls, cleanup := startFibSim(t, 1000)
	defer cleanup()

	// ① 首读：打设备一次（取数路径截到 unicast-afs 父容器、整列表连键回读），
	// 回填快照，谓词锚定下钻后 rows=前 50 行。
	env := decodePage(t, getConfigPageReq(h, "sim", fibRoutesPath, "include_state=true&limit=50&offset=0"))
	assert.True(t, env.Success, "首读应成功: %s", env.Message)
	assert.Equal(t, 1, *deviceCalls)
	assert.Equal(t, 1000, env.Data.Data.Total)
	assert.Len(t, env.Data.Data.Rows, 50)
	assert.Equal(t, "10.0.0.0", env.Data.Data.Rows[0]["destination"])
	assert.Equal(t, "device", env.Data.Source)

	// ② 翻页：命中快照，零设备往返；快照内顺序稳定（ygot 按键序）——
	// offset=100 首行 = 全量视图第 101 行。
	full := decodePage(t, getConfigPageReq(h, "sim", fibRoutesPath, "include_state=true&limit=200&offset=0"))
	env = decodePage(t, getConfigPageReq(h, "sim", fibRoutesPath, "include_state=true&limit=50&offset=100"))
	assert.Equal(t, 1, *deviceCalls, "翻页必须切自快照，不得再发 <get>")
	assert.Equal(t, "cache", env.Data.Source)
	assert.Equal(t, full.Data.Data.Rows[100]["destination"], env.Data.Data.Rows[0]["destination"],
		"翻页窗口必须与全量视图同序对齐")

	// ③ 过滤：if-name 等值命中 250 行（i%4==2），仍零设备往返。
	env = decodePage(t, getConfigPageReq(h, "sim", fibRoutesPath,
		"include_state=true&limit=10&filter=if-name%3D%3DGE1%2F0%2F2"))
	assert.Equal(t, 1, *deviceCalls)
	assert.Equal(t, 250, env.Data.Data.Total, "1000 行中 if-name=GE1/0/2 应为 250")
	assert.Equal(t, "GE1/0/2", env.Data.Data.Rows[0]["if-name"])

	// ④ force_refresh：绕快照重拉设备并覆盖回填。
	env = decodePage(t, getConfigPageReq(h, "sim", fibRoutesPath,
		"include_state=true&limit=50&force_refresh=true"))
	assert.Equal(t, 2, *deviceCalls, "force_refresh 必须重打设备")
	assert.Equal(t, "device", env.Data.Source)
	env = decodePage(t, getConfigPageReq(h, "sim", fibRoutesPath, "include_state=true&limit=50"))
	assert.Equal(t, 2, *deviceCalls, "force 结果应回填快照")
	assert.Equal(t, "cache", env.Data.Source)
}

// 任务 3.2：配置类大表（500 行 vlan）——无参形状回归 + 带参分页一致性。
func TestVlanConfigPagination_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	h, deviceCalls, cleanup := startFibSim(t, 10)
	defer cleanup()

	// 配置通道（<get-config>，不带 include_state）：500 行 vlan 配置大表。
	const vlansPath = "/vlan:vlan/vlan:vlans"

	// 无参：整树形状（vlan 数组直挂），回归锚点。打设备一次并回填运行缓存。
	w := getConfigPageReq(h, "sim", vlansPath, "")
	var raw struct {
		Data struct {
			Data map[string]interface{} `json:"data"`
		} `json:"data"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &raw))
	all, ok := raw.Data.Data["vlan"].([]interface{})
	assert.True(t, ok, "无参响应必须保持整树形状")
	assert.Len(t, all, 500)
	assert.Equal(t, 1, *deviceCalls)

	// 带参：total = 无参行数、末页截断正确；同键命中运行缓存零设备往返。
	env := decodePage(t, getConfigPageReq(h, "sim", vlansPath, "limit=100&offset=450"))
	assert.Equal(t, 500, env.Data.Data.Total, "带参 total 必须与无参行数一致")
	assert.Len(t, env.Data.Data.Rows, 50, "offset=450 末页应截断为 50 行")
	assert.Equal(t, 1, *deviceCalls, "分页读应命中无参读回填的同一缓存")
	assert.Equal(t, "cache", env.Data.Source)

	// 过滤+排序下推：id 数值降序首行应为 501；name 包含过滤命中。
	env = decodePage(t, getConfigPageReq(h, "sim", vlansPath, "limit=1&sort=id&sort_dir=desc"))
	assert.Equal(t, float64(501), env.Data.Data.Rows[0]["id"])
	env = decodePage(t, getConfigPageReq(h, "sim", vlansPath, "limit=10&filter=name%3D%3Dv100"))
	assert.Equal(t, 1, env.Data.Data.Total)
	assert.Equal(t, float64(100), env.Data.Data.Rows[0]["id"])
	assert.Equal(t, 1, *deviceCalls, "过滤排序全部切自缓存")
}
