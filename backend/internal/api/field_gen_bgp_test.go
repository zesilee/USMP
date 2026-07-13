package api

import (
	"strings"
	"testing"

	"github.com/leezesi/usmp/backend/internal/yangschema"
)

// TestBgpSchemaCarriesConstraints 验证 schema 端点把 BGP 的 when/must 约束透出到
// FieldDef——公网 BGP 的 enable/as 耦合约束（must "(enable='true' and as) or
// (enable='false' and not(as))"）与 as 的 when（"../enable='true'"）驱动前端条件
// 显隐与校验（R05/§9）。服务端不做 when/must 校验（与 VLAN/IFM 一致，域约束除外），
// 约束由前端渲染 + 设备 edit-config 兜底；本用例锁死「约束元数据对 BGP 也 100%
// 数据驱动透出」，防前端拿不到约束而放行非法组合。
func TestBgpSchemaCarriesConstraints(t *testing.T) {
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	mod, ok := s.Module("bgp")
	if !ok {
		t.Fatal("bgp module not loaded")
	}
	ys := buildYangSchemaNested(mod)

	// as 叶带 when 门控（enable=true 才可配 as）
	as, ok := findFieldBySuffix(ys.Fields, "as")
	if !ok {
		t.Fatal("base-process/as field not found in BGP schema")
	}
	if as.When != "../enable='true'" {
		t.Errorf("as When = %q, want %q", as.When, "../enable='true'")
	}

	// enable 叶带 must 约束（enable 与 as 的存在性耦合）
	enable, ok := findFieldBySuffix(ys.Fields, "enable")
	if !ok {
		t.Fatal("base-process/enable field not found in BGP schema")
	}
	if len(enable.Must) == 0 {
		t.Fatalf("enable 应携带 must 约束（enable/as 耦合），got 0 条")
	}
	foundCoupling := false
	for _, m := range enable.Must {
		if strings.Contains(m.Expr, "enable") && strings.Contains(m.Expr, "as") {
			foundCoupling = true
		}
	}
	if !foundCoupling {
		t.Errorf("enable 的 must 未含 enable/as 耦合表达式，got %+v", enable.Must)
	}
}
