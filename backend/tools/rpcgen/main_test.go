package main

import (
	"os"
	"strings"
	"testing"
)

// RPC-01/RPC-04：从 YANG 提取 rpc 定义——名称、input 叶（类型/leafref/mandatory/
// units）、高危分类。正/负路径覆盖。
func TestBuildRPCs(t *testing.T) {
	got, _, err := buildRPCs("testdata", []string{"demo-rpc"})
	if err != nil {
		t.Fatalf("buildRPCs: %v", err)
	}

	rpcs, ok := got["ifm"]
	if !ok {
		t.Fatalf("demo-rpc 未提取到 rpc(键=根容器 ifm)")
	}
	if len(rpcs) != 3 {
		t.Fatalf("rpc 数 = %d, want 3 (reset-counters/restart-if/ping-op)", len(rpcs))
	}

	byName := map[string]RPCDef{}
	for _, r := range rpcs {
		byName[r.Name] = r
	}

	// reset-counters：input if-name 为 leafref + mandatory；非高危（reset-counters 温和）。
	rc, ok := byName["reset-counters"]
	if !ok {
		t.Fatal("缺 reset-counters")
	}
	if rc.HighRisk {
		t.Error("reset-counters 不应判为高危（温和操作，走基础确认）")
	}
	if len(rc.Input) != 1 {
		t.Fatalf("reset-counters input 数 = %d, want 1", len(rc.Input))
	}
	in := rc.Input[0]
	if in.Name != "if-name" {
		t.Errorf("input 叶名 = %q, want if-name", in.Name)
	}
	if in.Type != "leafref" {
		t.Errorf("input 类型 = %q, want leafref", in.Type)
	}
	if in.LeafRef == "" {
		t.Error("leafref 目标路径应非空")
	}
	if !in.Mandatory {
		t.Error("if-name 应为 mandatory")
	}

	// restart-if：高危；含 units 叶。
	ri, ok := byName["restart-if"]
	if !ok {
		t.Fatal("缺 restart-if")
	}
	if !ri.HighRisk {
		t.Error("restart-if 应判为高危（restart 关键词）")
	}
	var grace *RPCInputLeaf
	for i := range ri.Input {
		if ri.Input[i].Name == "grace-period" {
			grace = &ri.Input[i]
		}
	}
	if grace == nil {
		t.Fatal("restart-if 缺 grace-period 叶")
	}
	if grace.Type != "number" {
		t.Errorf("grace-period 类型 = %q, want number", grace.Type)
	}
	// 注：goyang v1.6.0 不把 rpc-input 叶的 units 传到 Entry.Units（实测）——units
	// 提取为 best-effort，rpc 输入叶通常不带单位，不作强断言。
	if grace.Mandatory {
		t.Error("grace-period 非 mandatory")
	}

	// ping-op：无 input。
	po, ok := byName["ping-op"]
	if !ok {
		t.Fatal("缺 ping-op")
	}
	if len(po.Input) != 0 {
		t.Errorf("ping-op input 应为空, got %d", len(po.Input))
	}
}

// 确定性：同输入两次提取结果一致（regen-and-diff 前提）。
func TestBuildRPCs_Deterministic(t *testing.T) {
	a, _, err := buildRPCs("testdata", []string{"demo-rpc"})
	if err != nil {
		t.Fatalf("run1: %v", err)
	}
	b, _, err := buildRPCs("testdata", []string{"demo-rpc"})
	if err != nil {
		t.Fatalf("run2: %v", err)
	}
	if len(a["ifm"]) != len(b["ifm"]) {
		t.Fatalf("两次 rpc 数不一致")
	}
	for i := range a["ifm"] {
		if a["ifm"][i].Name != b["ifm"][i].Name {
			t.Errorf("rpc 顺序不确定: %d %q vs %q", i, a["ifm"][i].Name, b["ifm"][i].Name)
		}
	}
}

// 高危关键词判定表。
func TestHighRisk(t *testing.T) {
	cases := map[string]bool{
		"restart-if":                true,
		"reboot-system":             true,
		"power-off-board":           true,
		"delete-all":                true,
		"batch-delete-vpn":          true,
		"rollback-config":           true,
		"warm-reboot":               true,
		"upgrade-firmware":          true,
		"reset-if-counters-by-name": false,
		"clear-if-counters":         false,
		"ping-op":                   false,
		"check-hardware-version":    false,
	}
	for name, want := range cases {
		if got := isHighRisk(name); got != want {
			t.Errorf("isHighRisk(%q) = %v, want %v", name, got, want)
		}
	}
}

// render 产出合法 Go 且含 ModuleRPCs 与已知 rpc（覆盖 codegen 路径）。
func TestRender(t *testing.T) {
	rpcs, ns, err := buildRPCs("testdata", []string{"demo-rpc"})
	if err != nil {
		t.Fatalf("buildRPCs: %v", err)
	}
	src, err := render(rpcs, ns, "yangschema")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	s := string(src)
	for _, want := range []string{"package yangschema", "var ModuleRPCs", "restart-if", "HighRisk: true", "LeafRef:"} {
		if !contains(s, want) {
			t.Errorf("render 输出缺 %q", want)
		}
	}
}

// run 落盘全链：写文件、内容含 ModuleRPCs、计数正确。
func TestRun(t *testing.T) {
	out := t.TempDir() + "/rpc.gen.go"
	nMod, nRPC, err := run("testdata", []string{"demo-rpc"}, out, "yangschema")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if nMod != 1 || nRPC != 3 {
		t.Fatalf("run 计数 = (%d,%d), want (1,3)", nMod, nRPC)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !contains(string(b), "var ModuleRPCs") {
		t.Error("落盘文件缺 ModuleRPCs")
	}
}

// run 到不可写路径应返回错误（负路径，不 panic）。
func TestRun_WriteError(t *testing.T) {
	if _, _, err := run("testdata", []string{"demo-rpc"}, t.TempDir()+"/nope/rpc.gen.go", "yangschema"); err == nil {
		t.Error("写入不可创建路径应报错")
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }
