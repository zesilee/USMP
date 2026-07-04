package api

import (
	"encoding/json"
	"net/http/httptest"
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
// with correct per-module vendors (not all "huawei").
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
	if byName["interfaces"].Vendor != "openconfig" {
		t.Errorf("interfaces vendor = %q, want openconfig", byName["interfaces"].Vendor)
	}
}
