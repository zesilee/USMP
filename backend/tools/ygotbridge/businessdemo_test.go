package ygotbridge

import (
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
	"github.com/leezesi/usmp/backend/tools/businessdemo"
)

// C2Y-05 —— crd2yang 生成的示例模型（usmp-business-vlan-net）经既有 gen.conf
// 管线出 ygot 包后，schema 可注册进 schema 树（AddYgotSchemaWithVendor 兼容），
// 树形与约束承载完整；同时锁定 D7 决策：demo 不接线运行期 Load()（业务控制台
// 单 Kind 写死，挂菜单会出无 API 支撑的死页面——接线属后续 change）。

func TestBusinessDemoSchemaRegisters(t *testing.T) {
	bs, err := businessdemo.Schema()
	if err != nil {
		t.Fatalf("businessdemo.Schema: %v", err)
	}
	ds := schema.NewSchema()
	if err := AddYgotSchemaWithVendor(ds, bs, "usmp"); err != nil {
		t.Fatalf("AddYgotSchemaWithVendor: %v", err)
	}

	var found bool
	for _, mod := range ds.Modules() {
		if mod.Name() != "business-vlan-net" {
			continue
		}
		found = true
		if v := mod.Vendor(); v != "usmp" {
			t.Errorf("business-vlan-net vendor = %q, want usmp", v)
		}
		root := mod.Root()
		if root == nil {
			t.Fatal("business-vlan-net root is nil")
		}
		// 顶层叶/容器/list 全在位（CRD spec → YANG 树形完整性）。
		for _, name := range []string{"vlan-id", "name", "description", "vlan-type", "enabled", "qos", "devices"} {
			if child, ok := root.Child(name); !ok || child == nil {
				t.Errorf("business-vlan-net schema missing child %q", name)
			}
		}
		// 约束元数据存活：range/mandatory/pattern 穿过 CRD→YANG→ygot→schema 树全链。
		if v, ok := root.Child("vlan-id"); ok && v != nil {
			leaf := v.(schema.LeafNode)
			if !leaf.Mandatory() {
				t.Error("vlan-id should stay mandatory through the pipeline")
			}
			if lo, ok := leaf.RangeMin(); !ok || lo != 1 {
				t.Errorf("vlan-id RangeMin = %v,%v want 1,true", lo, ok)
			}
			if hi, ok := leaf.RangeMax(); !ok || hi != 4094 {
				t.Errorf("vlan-id RangeMax = %v,%v want 4094,true", hi, ok)
			}
		}
		if n, ok := root.Child("name"); ok && n != nil {
			if p := n.(schema.LeafNode).Pattern(); p != "[A-Za-z0-9_-]{1,31}" {
				t.Errorf("name pattern = %q, want huawei-vlan aligned pattern", p)
			}
		}
		// 嵌套容器与 list 子级也要可达（qos/priority、devices/ip+access-ports）。
		if qos, ok := root.Child("qos"); ok && qos != nil {
			if prio, ok := qos.(schema.ContainerNode).Child("priority"); !ok || prio == nil {
				t.Error("qos/priority missing in schema tree")
			}
		}
		if devices, ok := root.Child("devices"); ok && devices != nil {
			for _, name := range []string{"ip", "access-ports", "trunk-ports"} {
				if child, ok := devices.(schema.ContainerNode).Child(name); !ok || child == nil {
					t.Errorf("devices/%s missing in schema tree", name)
				}
			}
		}
	}
	if !found {
		t.Fatal("AddYgotSchemaWithVendor did not expose business-vlan-net")
	}
}
