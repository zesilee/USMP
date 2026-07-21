package netconfsim

import (
	"strings"
	"testing"
)

// DemoSeedConfig 必须可被 datastore 解析并按 IFM 子树过滤回读：5 条接口
// （3 main + 2 sub），sub 行携带 parent-name（供通用控制台行级 when 演示）。
func TestDemoSeedConfigLoadsAndFilters(t *testing.T) {
	ds := newTreeDatastore()
	if err := ds.SetRunning([]byte(DemoSeedConfig)); err != nil {
		t.Fatalf("SetRunning(DemoSeedConfig): %v", err)
	}

	out := string(ds.GetRunning())
	if got := strings.Count(out, "<interface>"); got != 5 {
		t.Fatalf("seeded interfaces = %d, want 5\n%s", got, out)
	}
	for _, want := range []string{
		"<name>200GE0/1/0</name>", "<name>200GE0/1/1</name>", "<name>200GE0/1/2</name>",
		"<name>200GE0/1/0.1</name>", "<name>200GE0/1/1.1</name>",
		"<parent-name>200GE0/1/0</parent-name>", "<parent-name>200GE0/1/1</parent-name>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("running config missing %s", want)
		}
	}

	// IFM 子树过滤（后端 get-config filter 形态）应仅返回 ifm 子树且含全部接口。
	filtered, err := ds.GetConfigFiltered([]byte(`<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm"><interfaces/></ifm>`))
	if err != nil {
		t.Fatalf("GetConfigFiltered: %v", err)
	}
	fs := string(filtered)
	if got := strings.Count(fs, "<interface>"); got != 5 {
		t.Fatalf("filtered interfaces = %d, want 5\n%s", got, fs)
	}
}

// DemoStateSeed 必须与 DemoSeedConfig 键对齐：5 条接口全部命中并各带 dynamic
// 状态容器；枚举取设备侧数字形态（oper-status 全数字），uint64 带宽存在。
func TestDemoStateSeedAlignsWithConfigSeed(t *testing.T) {
	ds := newTreeDatastore()
	if err := ds.SetRunning([]byte(DemoSeedConfig)); err != nil {
		t.Fatalf("SetRunning(DemoSeedConfig): %v", err)
	}
	if err := ds.SetState([]byte(DemoStateSeed)); err != nil {
		t.Fatalf("SetState(DemoStateSeed): %v", err)
	}

	out, err := ds.GetFiltered(nil)
	if err != nil {
		t.Fatalf("GetFiltered: %v", err)
	}
	got := treeFromXML(t, string(out))
	ifaces := got.find("ifm", "interfaces").children("interface")
	if len(ifaces) != 5 {
		t.Fatalf("merged interfaces = %d, want 5（状态种子不得引入幽灵条目）\n%s", len(ifaces), out)
	}
	for _, it := range ifaces {
		name := it.child("name").leafText()
		dyn := it.child("dynamic")
		if dyn == nil {
			t.Errorf("interface %s missing dynamic state（种子键未对齐）", name)
			continue
		}
		oper := dyn.child("oper-status")
		if oper == nil || strings.TrimFunc(oper.leafText(), func(r rune) bool { return r >= '0' && r <= '9' }) != "" {
			t.Errorf("interface %s oper-status 应为设备侧数字形态, got %v", name, oper)
		}
		if mac := dyn.child("mac-address"); mac == nil || mac.leafText() == "" {
			t.Errorf("interface %s missing mac-address", name)
		}
		if bw := dyn.child("bandwidth"); bw == nil || bw.leafText() == "" {
			t.Errorf("interface %s missing bandwidth", name)
		}
		// 配置叶保留（合并非替换）
		if it.child("admin-status") == nil {
			t.Errorf("interface %s config leaf admin-status lost in merge", name)
		}
	}
	// get-config 隔离：配置通道不得泄漏状态种子
	cfg, err := ds.GetConfigFiltered(nil)
	if err != nil {
		t.Fatalf("GetConfigFiltered: %v", err)
	}
	if strings.Contains(string(cfg), "dynamic") {
		t.Fatalf("get-config leaked DemoStateSeed: %s", cfg)
	}
}
