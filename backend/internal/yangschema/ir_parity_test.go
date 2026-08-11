package yangschema

import (
	"testing"
)

// TestLoadFromIR：IR 加载路径冒烟（blob 可解码、模块集非空、关键模块在位、
// 模块 namespace 已随 S4 直读源补全）。逐字节新鲜度门禁在 tools/schemagen
// 的 TestSourceVsBlobCompare。
func TestLoadFromIR(t *testing.T) {
	s, err := loadFromIR()
	if err != nil {
		t.Fatalf("loadFromIR: %v", err)
	}
	if len(s.Modules()) < 60 {
		t.Fatalf("模块数异常: %d", len(s.Modules()))
	}
	for _, name := range []string{"vlan", "ifm", "bgp", "business-vlan-service"} {
		m, ok := s.Module(name)
		if !ok {
			t.Fatalf("模块 %s 缺失", name)
		}
		if name != "business-vlan-service" && m.Vendor() != "huawei" {
			t.Fatalf("模块 %s vendor=%q", name, m.Vendor())
		}
	}
	if m, _ := s.Module("vlan"); m.Namespace() != "urn:huawei:yang:huawei-vlan" {
		t.Fatalf("vlan namespace 未补全: %q", m.Namespace())
	}
}
