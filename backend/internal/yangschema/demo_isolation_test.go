package yangschema

import "testing"

// D7 隔离锚点：demo 模型（business-vlan-net）绝不出现在运行期 Load()。若此
// 测试爆红，说明有人把 businessdemo 接进了 schema 链——须先做多 Kind 业务
// API 泛化（另立 change）。本测试零 demo 包依赖（只查模块名缺席）。
func TestBusinessDemoNotInRuntimeLoad(t *testing.T) {
	s, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, mod := range s.Modules() {
		if mod.Name() == "business-vlan-net" {
			t.Fatal("business-vlan-net must NOT be wired into runtime Load() (design D7)")
		}
	}
}
