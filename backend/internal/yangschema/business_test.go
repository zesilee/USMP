package yangschema

import "testing"

// BVS-01/D2 —— 意图模型接入管线：usmp-business-vlan 经 ygot 生成并由 Load 暴露
// （/yang/modules、/yang/schema 的数据源），task-name 扩展经构建期收割驱动
// 前端「业务网络配置」菜单组（FE-17 的后端前提）。

func TestLoadExposesBusinessVlanService(t *testing.T) {
	s, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var found bool
	for _, mod := range s.Modules() {
		if mod.Name() == "business-vlan-service" {
			found = true
			if v := mod.Vendor(); v != "usmp" {
				t.Errorf("business-vlan-service vendor = %q, want usmp", v)
			}
			root := mod.Root()
			if root == nil {
				t.Fatal("business-vlan-service root is nil")
			}
			for _, leaf := range []string{"vlan-id", "name", "devices"} {
				if child, ok := root.Child(leaf); !ok || child == nil {
					t.Errorf("business-vlan-service schema missing child %q", leaf)
				}
			}
		}
	}
	if !found {
		t.Fatal("Load() does not expose business-vlan-service module")
	}
}

func TestCategoryBusinessNetwork(t *testing.T) {
	if got := Category("business-vlan-service"); got != "business-network" {
		t.Errorf("Category(business-vlan-service) = %q, want business-network", got)
	}
	// 存量映射不回归。
	if got := Category("vlan"); got != "vlan" {
		t.Errorf("Category(vlan) = %q, want vlan (存量回归)", got)
	}
}
