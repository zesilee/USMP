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
)

// offlineDeviceHarness registers an unreachable device "dead"（协商恒不可得）.
func offlineDeviceHarness(t *testing.T) *YangHandler {
	t.Helper()
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
	return NewYangHandler(mgr)
}

func getLeftTree(t *testing.T, h *YangHandler, query string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/yang/left-tree"+query, nil)
	h.LeftTree(c)
	return w
}

// findLeaf 深度优先找 sourceModule 匹配的叶子。
func findLeaf(nodes []LeftTreeNodeDTO, sourceModule string) *LeftTreeNodeDTO {
	for i := range nodes {
		if nodes[i].SourceModule == sourceModule {
			return &nodes[i]
		}
		if f := findLeaf(nodes[i].Children, sourceModule); f != nil {
			return f
		}
	}
	return nil
}

func countTreeLeaves(nodes []LeftTreeNodeDTO) int {
	c := 0
	for _, n := range nodes {
		if n.SourceModule != "" {
			c++
		}
		c += countTreeLeaves(n.Children)
	}
	return c
}

// TestLeftTreeShape（LT-02）：14 顶层组、65 叶全在；vlan 已加载叶 available+module；
// 未生成模块叶 available:false 仍在树。
func TestLeftTreeShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newYangHandlerWithSchema(t)
	w := getLeftTree(t, h, "")
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	var tree []LeftTreeNodeDTO
	decodeData(t, w.Body.Bytes(), &tree)
	if len(tree) != 14 {
		t.Fatalf("顶层分组 = %d, want 14", len(tree))
	}
	if n := countTreeLeaves(tree); n != 65 {
		t.Fatalf("叶子数 = %d, want 65（全树+占位）", n)
	}
	vlan := findLeaf(tree, "huawei-vlan")
	if vlan == nil || vlan.Available == nil || !*vlan.Available || vlan.Module != "vlan" {
		t.Fatalf("huawei-vlan 叶应 available:true module:vlan, got %+v", vlan)
	}
	dsa := findLeaf(tree, "huawei-dsa")
	if dsa == nil || dsa.Available == nil || *dsa.Available {
		t.Fatalf("huawei-dsa（未生成）应 available:false 且在树中, got %+v", dsa)
	}
	if dsa.Module != "" {
		t.Errorf("不可用叶不应携带 module, got %q", dsa.Module)
	}
	// 双语字段存在
	if tree[0].Zh == "" || tree[0].En == "" {
		t.Errorf("顶层组应带双语名: %+v", tree[0])
	}
}

// TestLeftTreeDeviceSupported（LT-02 设备叠加）：设备仅声明 vlan → vlan 叶
// supported:true、ifm 叶 supported:false；协商省略语义见 offline 用例。
func TestLeftTreeDeviceSupported(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	gin.SetMode(gin.TestMode)
	h, deviceID, cleanup := newDeviceYangHarness(t, []string{
		"urn:huawei:params:xml:ns:yang:huawei-vlan?module=huawei-vlan&revision=2020-02-07",
	})
	defer cleanup()

	w := getLeftTree(t, h, "?device="+deviceID)
	var tree []LeftTreeNodeDTO
	decodeData(t, w.Body.Bytes(), &tree)
	vlan := findLeaf(tree, "huawei-vlan")
	if vlan == nil || vlan.Supported == nil || !*vlan.Supported {
		t.Fatalf("vlan 叶应 supported:true, got %+v", vlan)
	}
	ifm := findLeaf(tree, "huawei-ifm")
	if ifm == nil || ifm.Supported == nil || *ifm.Supported {
		t.Fatalf("ifm 叶应 supported:false, got %+v", ifm)
	}
}

// TestLeftTreeDeviceOfflineOmitsSupported（LT-02 降级）：协商不可得 → 全树省略
// supported（unknown ≠ 不支持）。
func TestLeftTreeDeviceOfflineOmitsSupported(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := offlineDeviceHarness(t)
	w := getLeftTree(t, h, "?device=dead")
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	raw := w.Body.Bytes()
	if json.Valid(raw) && containsKeyDeep(raw, "supported") {
		t.Error("协商不可得时不应出现 supported 字段")
	}
}

// TestLeftTreeDeviceUnknown（负路径）：未注册设备信封 404。
func TestLeftTreeDeviceUnknown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newYangHandlerWithSchema(t)
	w := getLeftTree(t, h, "?device=nope")
	var env struct {
		Code    int  `json:"code"`
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Success || env.Code != 404 {
		t.Fatalf("应信封 404, got %d", env.Code)
	}
}

func containsKeyDeep(raw []byte, key string) bool {
	return json.Valid(raw) && (stringContains(string(raw), `"`+key+`"`))
}

func stringContains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
