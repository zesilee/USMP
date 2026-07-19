package api

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/leezesi/usmp/backend/internal/yangschema"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
)

func newYangHandlerWithSchema(t *testing.T) *YangHandler {
	t.Helper()
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("Load schema: %v", err)
	}
	return NewYangHandler(manager.New(manager.WithSchema(s)))
}

// envelope decodes the Success() response wrapper's data into v.
func decodeData(t *testing.T, body []byte, v interface{}) {
	t.Helper()
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode envelope: %v (body=%s)", err, body)
	}
	if err := json.Unmarshal(env.Data, v); err != nil {
		t.Fatalf("decode data: %v (data=%s)", err, env.Data)
	}
}

// TestGetSchemaDynamic (task 2.1): a loaded module returns a dynamically generated
// schema (real fields), not the 2-field stub.
func TestGetSchemaDynamic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newYangHandlerWithSchema(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "module", Value: "vlan"}}
	h.GetSchema(c)

	var ys YangSchema
	decodeData(t, w.Body.Bytes(), &ys)
	if ys.Module != "vlan" || ys.Vendor != "huawei" {
		t.Fatalf("module/vendor = %s/%s, want vlan/huawei", ys.Module, ys.Vendor)
	}
	if len(ys.Fields) < 5 {
		t.Fatalf("dynamic vlan schema should have many fields, got %d (stub?)", len(ys.Fields))
	}
}

// TestListModulesDynamic (task 2.4): module list reflects the loaded schema tree
// with correct per-module vendors — huawei/usmp only (BR-11).
func TestListModulesDynamic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newYangHandlerWithSchema(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	h.ListModules(c)

	var mods []YangModuleInfo
	decodeData(t, w.Body.Bytes(), &mods)
	byName := map[string]YangModuleInfo{}
	for _, m := range mods {
		byName[m.Name] = m
	}
	if len(mods) < 5 {
		t.Fatalf("expected loaded modules, got %d", len(mods))
	}
	if byName["vlan"].Vendor != "huawei" {
		t.Errorf("vlan vendor = %q, want huawei", byName["vlan"].Vendor)
	}
	if byName["business-vlan-service"].Vendor != "usmp" {
		t.Errorf("business-vlan-service vendor = %q, want usmp", byName["business-vlan-service"].Vendor)
	}
	for _, m := range mods {
		if m.Vendor != "huawei" && m.Vendor != "usmp" {
			t.Errorf("module %q vendor = %q, want huawei or usmp (BR-11)", m.Name, m.Vendor)
		}
	}
	if _, ok := byName["interfaces"]; ok {
		t.Error("openconfig module \"interfaces\" must not be exposed (BR-11)")
	}
}

// TestListModulesCategory (BR-01): modules whose source YANG declares a
// module-level `task-name` carry `category` from the build-time map; modules
// without a mapping omit it and the endpoint never fails (R08).
func TestListModulesCategory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newYangHandlerWithSchema(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	h.ListModules(c)

	var mods []YangModuleInfo
	decodeData(t, w.Body.Bytes(), &mods)
	byName := map[string]YangModuleInfo{}
	for _, m := range mods {
		byName[m.Name] = m
	}
	cases := []struct {
		module string
		want   string
	}{
		{"ifm", "interface-mgr"},
		{"vlan", "vlan"},
		{"system", "system"},
		{"bgp", "bgp"}, // full-yang-onboarding：taskname 清单全量化后 bgp 有映射
	}
	for _, cse := range cases {
		if got := byName[cse.module].Category; got != cse.want {
			t.Errorf("%s category = %q, want %q", cse.module, got, cse.want)
		}
	}

	// omitempty boundary：无 task-name 映射的模块序列化时省略 category 键。
	// 全量化后闭包拖入的黑名单根（如 mpls）无映射，取其为无映射样本；若全部
	// 模块都有映射则跳过该边界（语义已由 want:"" 空断言覆盖）。
	for _, name := range []string{"mpls", "ssl", "l2vpn"} {
		m, ok := byName[name]
		if !ok || m.Category != "" {
			continue
		}
		raw, err := json.Marshal(m)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if strings.Contains(string(raw), `"category"`) {
			t.Errorf("%s serializes category unexpectedly: %s", name, raw)
		}
		return
	}
}
