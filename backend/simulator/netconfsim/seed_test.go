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
