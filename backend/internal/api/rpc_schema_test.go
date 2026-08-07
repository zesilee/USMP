package api

import (
	"net/http"
	"testing"

	"github.com/leezesi/usmp/backend/internal/yangschema"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
)

// RPC-02：/yang/schema/:module 响应含 rpcs 数组，input 为 FieldDef（含 leafref/
// mandatory）；无 rpc 模块 rpcs 为空且不报错。
func TestGetSchema_IncludesRPCs(t *testing.T) {
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("load schema: %v", err)
	}
	h := NewYangHandler(manager.New(manager.WithSchema(s)))

	// ifm 含 rpc（reset-if-counters-by-name / restart-if …）。
	ys := getNestedSchema(t, h, "ifm")
	if len(ys.RPCs) == 0 {
		t.Fatal("ifm schema 应含 rpcs")
	}

	byName := map[string]RPCSchema{}
	for _, r := range ys.RPCs {
		byName[r.Name] = r
	}

	rc, ok := byName["reset-if-counters-by-name"]
	if !ok {
		t.Fatalf("缺 reset-if-counters-by-name，got %v", keysOfRPC(ys.RPCs))
	}
	if rc.HighRisk {
		t.Error("reset-if-counters-by-name 不应为高危")
	}
	if len(rc.Input) != 1 {
		t.Fatalf("reset-if-counters-by-name input 数 = %d, want 1", len(rc.Input))
	}
	in := rc.Input[0]
	if in.Path == "" || in.Label == "" {
		t.Errorf("input 字段应有 path/label: %+v", in)
	}
	if !in.Required {
		t.Error("if-name 应 Required（mandatory）")
	}
	if in.LeafRef == "" {
		t.Error("if-name 应携带 leafRef 目标（供前端下拉）")
	}

	// restart-if 高危。
	if ri, ok := byName["restart-if"]; !ok || !ri.HighRisk {
		t.Errorf("restart-if 应存在且高危: ok=%v", ok)
	}
}

// RPC-02 边界：无 rpc 的模块 rpcs 为空且不报错。
func TestGetSchema_NoRPCModuleEmpty(t *testing.T) {
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("load schema: %v", err)
	}
	h := NewYangHandler(manager.New(manager.WithSchema(s)))

	// system 无 rpc（不在 rpc.gen.go 键集）。
	ys := getNestedSchema(t, h, "system")
	if len(ys.RPCs) != 0 {
		t.Errorf("无 rpc 模块 rpcs 应为空, got %d", len(ys.RPCs))
	}
	if ys.Module != "system" {
		t.Errorf("module = %q", ys.Module)
	}
}

func getNestedSchema(t *testing.T, h *YangHandler, module string) YangSchema {
	t.Helper()
	c, w := newTestContext(http.MethodGet, "/api/v1/yang/schema/"+module+"?form=nested", nil, "module", module)
	h.GetSchema(c)
	var ys YangSchema
	decodeData(t, w.Body.Bytes(), &ys)
	return ys
}

func keysOfRPC(rs []RPCSchema) []string {
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		out = append(out, r.Name)
	}
	return out
}
