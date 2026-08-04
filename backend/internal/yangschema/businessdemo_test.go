package yangschema

import (
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/businessdemo"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
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
	schema.AddYgotSchemaWithVendor(ds, bs, "usmp")

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

// D7 隔离锚点：demo 模型绝不出现在运行期 Load()。若此测试爆红，说明有人把
// businessdemo 接进了 load.go——须先做多 Kind 业务 API 泛化（另立 change）。
func TestBusinessDemoNotInRuntimeLoad(t *testing.T) {
	s, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, mod := range s.Modules() {
		if mod.Name() == "business-vlan-net" {
			t.Fatal("business-vlan-net must NOT be wired into runtime Load() (design D7): the business console is single-Kind today")
		}
	}
}
