package yangschema

import (
	"bytes"
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/business"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
	"github.com/leezesi/usmp/backend/tools/ygotbridge"
)

// loadUncached 重建旧链路树（generated Schema()→ygotbridge 转换；原 load.go
// 同名函数，阶段1.5 迁入测试文件——运行时闭包自此零 generated/ygot 引用）。
func loadUncached(t *testing.T) *schema.DefaultSchema {
	t.Helper()
	ds := schema.NewSchema()
	hs, err := huawei.Schema()
	if err != nil {
		t.Fatalf("load huawei schema: %v", err)
	}
	if err := ygotbridge.AddYgotSchemaWithVendor(ds, hs, "huawei"); err != nil {
		t.Fatalf("convert huawei schema: %v", err)
	}
	bs, err := business.Schema()
	if err != nil {
		t.Fatalf("load business schema: %v", err)
	}
	if err := ygotbridge.AddYgotSchemaWithVendor(ds, bs, "usmp"); err != nil {
		t.Fatalf("convert business schema: %v", err)
	}
	return ds
}

// TestIRBlobMatchesLegacyChain 是阶段1（Schema IR 自立）的 schema 通道对拍
// （YN-06）：入库 IR blob 必须与「旧链路在本机即时构建的树」的 IR 编码逐字节一致。
//
// 字节级等价蕴含树内容等价——两条路径喂给 /yang/schema 等全部消费方的是同一棵树，
// 故此对拍强于任何按端点抽样的 JSON 对比。同时它兼作新鲜度门禁：generated 包
// schema 变更而未重跑 `go generate ./internal/yangschema` 时，本测试即红
// （与 CG-03 regen-and-diff 同精神）。
func TestIRBlobMatchesLegacyChain(t *testing.T) {
	ds := loadUncached(t)
	want, err := schema.EncodeIR(ds)
	if err != nil {
		t.Fatalf("EncodeIR(legacy): %v", err)
	}
	if !bytes.Equal(want, schemaIRBlob) {
		t.Fatalf("schema.ir.gz 与旧链路树不一致（%d vs %d bytes）——generated schema 变更后未重跑 go generate ./internal/yangschema？", len(schemaIRBlob), len(want))
	}
}

// TestLoadFromIR：IR 路径可独立加载出与旧链路同模块集的树（对拍的可用性冒烟）。
func TestLoadFromIR(t *testing.T) {
	s, err := loadFromIR()
	if err != nil {
		t.Fatalf("loadFromIR: %v", err)
	}
	legacy := loadUncached(t)
	lm, im := legacy.Modules(), s.Modules()
	if len(im) == 0 || len(im) != len(lm) {
		t.Fatalf("module count: ir=%d legacy=%d", len(im), len(lm))
	}
	for _, m := range lm {
		got, ok := s.Module(m.Name())
		if !ok {
			t.Fatalf("module %s missing from IR path", m.Name())
		}
		if got.Vendor() != m.Vendor() || got.Namespace() != m.Namespace() {
			t.Fatalf("module %s meta mismatch", m.Name())
		}
	}
}
