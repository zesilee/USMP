package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
	"github.com/leezesi/usmp/backend/tools/ygotbridge"
)

// 直读源 vs gzip 往返：切换 schemagen 输入前的实测比对（S4 决策依据）。
// 若不等，firstDiffPath 指出首处差异供人工核对（差异应只可能是「直读比往返
// 信息更全」的增益面——须显式核对再拍板换基线，GD-01 联动前端黄金）。
func buildFromSource(t *testing.T) *schema.DefaultSchema {
	t.Helper()
	confH, err := ParseGenConfAt("../../internal/generated/huawei/gen.conf", "../../..")
	if err != nil {
		t.Fatal(err)
	}
	confB, err := ParseGenConfAt("../../internal/generated/business/gen.conf", "../../..")
	if err != nil {
		t.Fatal(err)
	}
	ds := schema.NewSchema()
	eh, err := ygotbridge.LoadModuleEntries(confH.YangPaths, confH.Modules)
	if err != nil {
		t.Fatal(err)
	}
	if err := ygotbridge.AddSourceModules(ds, eh, "huawei", true); err != nil {
		t.Fatal(err)
	}
	eb, err := ygotbridge.LoadModuleEntries(confB.YangPaths, confB.Modules)
	if err != nil {
		t.Fatal(err)
	}
	if err := ygotbridge.AddSourceModules(ds, eb, "usmp", true); err != nil {
		t.Fatal(err)
	}
	return ds
}

// TestSourceVsBlobCompare 兼作 blob 新鲜度门禁：入库 blob 必须与直读源重建
// 逐字节一致（YANG 源/gen.conf/转换逻辑变更后未重跑 make gen-yang 即红）。
func TestSourceVsBlobCompare(t *testing.T) {
	src := buildFromSource(t)
	got, err := schema.EncodeIR(src)
	if err != nil {
		t.Fatal(err)
	}
	blob, err := os.ReadFile("../../internal/yangschema/schema.ir.gz")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(got, blob) {
		return
	}
	// 差异定位：解码两树逐模块比对模块集合与首个不同模块。
	a, err := schema.DecodeIR(got)
	if err != nil {
		t.Fatal(err)
	}
	b, err := schema.DecodeIR(blob)
	if err != nil {
		t.Fatal(err)
	}
	am, bm := map[string]bool{}, map[string]bool{}
	for _, m := range a.Modules() {
		am[m.Name()] = true
	}
	for _, m := range b.Modules() {
		bm[m.Name()] = true
	}
	var onlyA, onlyB []string
	for n := range am {
		if !bm[n] {
			onlyA = append(onlyA, n)
		}
	}
	for n := range bm {
		if !am[n] {
			onlyB = append(onlyB, n)
		}
	}
	t.Fatalf("直读源与 blob 不一致（%d vs %d bytes）；仅直读有: %v；仅 blob 有: %v",
		len(got), len(blob), strings.Join(onlyA, ","), strings.Join(onlyB, ","))
}
