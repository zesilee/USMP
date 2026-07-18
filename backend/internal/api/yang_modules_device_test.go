package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/leezesi/usmp/backend/internal/yangschema"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	netsim "github.com/leezesi/usmp/backend/simulator/netconfsim"
)

// negotiatedData 是 ?device= 形态的 data 负载（CN-02/BR-12）。
type negotiatedData struct {
	Modules    []YangModuleInfo `json:"modules"`
	Negotiated bool             `json:"negotiated"`
}

func newDeviceYangHarness(t *testing.T, caps []string) (*YangHandler, string, func()) {
	t.Helper()
	sim := netsim.NewSimulator()
	if caps != nil {
		sim.SetCapabilities(caps)
	}
	if err := sim.Start(); err != nil {
		t.Fatalf("start simulator: %v", err)
	}
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("Load schema: %v", err)
	}
	mgr := manager.New(manager.WithSchema(s))
	deviceID := "sim-dev"
	if err := mgr.GetDeviceStore().Put(deviceID, client.DeviceConnectionInfo{
		IP: sim.Addr(), Port: sim.Port(), Username: sim.Username(), Password: sim.Password(),
		Protocol: client.ProtocolNETCONF, Timeout: 5 * time.Second,
	}); err != nil {
		t.Fatalf("device store put: %v", err)
	}
	cleanup := func() {
		mgr.GetClientPool().CloseAll()
		sim.Stop()
	}
	return NewYangHandler(mgr), deviceID, cleanup
}

func listModulesWithQuery(t *testing.T, h *YangHandler, query string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/yang/modules"+query, nil)
	h.ListModules(c)
	return w
}

// TestListModulesDeviceNegotiated（CN-02 正路径）：设备 hello 仅声明 vlan/ifm
// → ?device= 返回协商子集 + negotiated:true。
func TestListModulesDeviceNegotiated(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	gin.SetMode(gin.TestMode)
	h, deviceID, cleanup := newDeviceYangHarness(t, []string{
		"urn:huawei:params:xml:ns:yang:huawei-vlan?module=huawei-vlan&revision=2020-02-07",
		"urn:huawei:params:xml:ns:yang:huawei-ifm?module=huawei-ifm&revision=2020-02-15",
	})
	defer cleanup()

	w := listModulesWithQuery(t, h, "?device="+deviceID)
	if w.Code != 200 {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	var data negotiatedData
	decodeData(t, w.Body.Bytes(), &data)
	if !data.Negotiated {
		t.Error("negotiated = false, want true（hello 能力可得）")
	}
	names := map[string]bool{}
	for _, m := range data.Modules {
		names[m.Name] = true
	}
	if !names["vlan"] || !names["ifm"] {
		t.Errorf("协商子集应含 vlan+ifm，got %v", names)
	}
	if names["system"] || names["bgp"] {
		t.Errorf("设备未声明的模块不应出现，got %v", names)
	}
}

// TestListModulesDeviceOffline（CN-02 降级）：设备已注册但不可达 → 全量 + negotiated:false，不 5xx。
func TestListModulesDeviceOffline(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("Load schema: %v", err)
	}
	mgr := manager.New(manager.WithSchema(s))
	if err := mgr.GetDeviceStore().Put("dead", client.DeviceConnectionInfo{
		IP: "127.0.0.1", Port: 1, Protocol: client.ProtocolNETCONF, Timeout: 500 * time.Millisecond,
	}); err != nil {
		t.Fatalf("put: %v", err)
	}
	h := NewYangHandler(mgr)

	w := listModulesWithQuery(t, h, "?device=dead")
	if w.Code != 200 {
		t.Fatalf("离线降级应 200，got %d", w.Code)
	}
	var data negotiatedData
	decodeData(t, w.Body.Bytes(), &data)
	if data.Negotiated {
		t.Error("negotiated 应为 false（能力不可得）")
	}
	if len(data.Modules) < 5 {
		t.Errorf("降级应返回全量模块，got %d", len(data.Modules))
	}
}

// TestListModulesDeviceUnknown（CN-02 负路径）：未注册设备 → 404。
func TestListModulesDeviceUnknown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newYangHandlerWithSchema(t)
	w := listModulesWithQuery(t, h, "?device=nope")
	var env struct {
		Code    int  `json:"code"`
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Success || env.Code != 404 {
		t.Fatalf("未注册设备应信封 404，got %d body=%s", env.Code, w.Body.String())
	}
}

// TestListModulesBlacklistAnnotation（CN-03）：blacklist 命中的模块（huawei-system）
// 附 blacklisted:true 且仍在列表；未命中模块无该键。
func TestListModulesBlacklistAnnotation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newYangHandlerWithSchema(t)
	w := listModulesWithQuery(t, h, "")
	var mods []YangModuleInfo
	decodeData(t, w.Body.Bytes(), &mods)
	byName := map[string]YangModuleInfo{}
	for _, m := range mods {
		byName[m.Name] = m
	}
	sys, ok := byName["system"]
	if !ok {
		t.Fatal("system 模块应仍在列表（注解不裁剪）")
	}
	if !sys.Blacklisted {
		t.Error("system 应标记 blacklisted:true（blacklist.xml 命中）")
	}
	raw, err := json.Marshal(byName["vlan"])
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if json.Valid(raw) && string(raw) != "" {
		if containsKey(raw, "blacklisted") {
			t.Errorf("vlan 未命中黑名单不应有 blacklisted 键: %s", raw)
		}
	}
}

func containsKey(raw []byte, key string) bool {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return false
	}
	_, ok := m[key]
	return ok
}
