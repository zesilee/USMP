package main

import "testing"

// buildTree：层级/双语保留、叶子解析出根容器（RPC 不算）、模块文件缺失的叶
// 容器为空且不阻断（LT-01 正/负路径）。
func TestBuildTree(t *testing.T) {
	nodes, err := buildTree("testdata/left-tree.json", "testdata", "testdata/i18n")
	if err != nil {
		t.Fatalf("buildTree: %v", err)
	}
	if len(nodes) != 1 || nodes[0].Zh != "基础配置" || nodes[0].En != "Basic Configuration" {
		t.Fatalf("顶层分组不符: %+v", nodes)
	}
	sub := nodes[0].Children
	if len(sub) != 2 {
		t.Fatalf("二级节点数 = %d, want 2", len(sub))
	}
	leafGood := sub[0].Children[0]
	if leafGood.SourceModule != "demo-good" {
		t.Fatalf("sourceModule = %q", leafGood.SourceModule)
	}
	if len(leafGood.RootContainers) != 2 || leafGood.RootContainers[0] != "goodroot" || leafGood.RootContainers[1] != "secondroot" {
		t.Errorf("rootContainers = %v, want [goodroot secondroot]（有序、无 RPC）", leafGood.RootContainers)
	}
	leafMissing := sub[1]
	if leafMissing.SourceModule != "demo-missing" {
		t.Fatalf("missing sourceModule = %q", leafMissing.SourceModule)
	}
	if len(leafMissing.RootContainers) != 0 {
		t.Errorf("缺文件模块 rootContainers 应为空, got %v", leafMissing.RootContainers)
	}
	if len(leafMissing.Nodes) != 0 {
		t.Errorf("缺文件模块 Nodes 应为空, got %v", leafMissing.Nodes)
	}
}

// 叶子模块级 Nodes：container 在前、rpc 在后（各自有序平铺）、res 双语烘焙、
// 缺键回退原名、highRisk 与 rpcgen 同口径（LT-01 children 烘焙）。
func TestBuildTreeModuleNodes(t *testing.T) {
	nodes, err := buildTree("testdata/left-tree.json", "testdata", "testdata/i18n")
	if err != nil {
		t.Fatalf("buildTree: %v", err)
	}
	leaf := nodes[0].Children[0].Children[0]
	got := leaf.Nodes
	if len(got) != 4 {
		t.Fatalf("Nodes 数 = %d, want 4（2 container + 2 rpc）: %+v", len(got), got)
	}
	want := []ModuleNode{
		{Kind: "container", Name: "goodroot", Zh: "好根", En: "Good Root"},
		{Kind: "container", Name: "secondroot", Zh: "secondroot", En: "secondroot"}, // res 缺键回退原名
		{Kind: "rpc", Name: "do-thing", Zh: "做事", En: "Do Thing"},
		{Kind: "rpc", Name: "restart-thing", Zh: "restart-thing", En: "restart-thing", HighRisk: true},
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("Nodes[%d] = %+v, want %+v", i, got[i], w)
		}
	}
}

// res 目录整体缺失：全部回退原名，不阻断（负路径 R08）。
func TestBuildTreeMissingResDir(t *testing.T) {
	nodes, err := buildTree("testdata/left-tree.json", "testdata", "testdata/no-such-i18n")
	if err != nil {
		t.Fatalf("buildTree: %v", err)
	}
	leaf := nodes[0].Children[0].Children[0]
	if len(leaf.Nodes) != 4 {
		t.Fatalf("Nodes 数 = %d, want 4", len(leaf.Nodes))
	}
	if leaf.Nodes[0].Zh != "goodroot" || leaf.Nodes[0].En != "goodroot" {
		t.Errorf("res 缺失应回退原名, got %+v", leaf.Nodes[0])
	}
}

// 负路径：JSON 缺失/畸形明确报错（R08 不产半成品）。
func TestBuildTreeNegative(t *testing.T) {
	if _, err := buildTree("testdata/nope.json", "testdata", "testdata/i18n"); err == nil {
		t.Error("missing json should error")
	}
	if _, err := buildTree("testdata/demo-good.yang", "testdata", "testdata/i18n"); err == nil {
		t.Error("malformed json should error")
	}
}

// renderNodes/countLeaves：生成物字面量确定性与叶子计数（LT-01 生成器内核）。
func TestRenderNodesAndCount(t *testing.T) {
	nodes := []TreeNode{
		{Zh: "组", En: "G", Children: []TreeNode{
			{Zh: "叶", En: "L", SourceModule: "demo-good", RootContainers: []string{"a", "b"},
				Nodes: []ModuleNode{
					{Kind: "container", Name: "a", Zh: "甲", En: "A"},
					{Kind: "rpc", Name: "restart-x", Zh: "重启", En: "Restart X", HighRisk: true},
				}},
		}},
	}
	out := renderNodes(nodes, 0)
	for _, want := range []string{
		`Zh: "组"`, `SourceModule: "demo-good"`, `RootContainers: []string{"a", "b"}`,
		"Children: []LeftTreeNode{", "Nodes: []LeftTreeModuleNode{",
		`{Kind: "container", Name: "a", Zh: "甲", En: "A"}`,
		`{Kind: "rpc", Name: "restart-x", Zh: "重启", En: "Restart X", HighRisk: true}`,
	} {
		if !contains(out, want) {
			t.Errorf("renderNodes 缺 %q:\n%s", want, out)
		}
	}
	if renderNodes(nodes, 0) != out {
		t.Error("renderNodes 应确定性")
	}
	if n := countLeaves(nodes); n != 1 {
		t.Errorf("countLeaves = %d, want 1", n)
	}
}

// 重复构建字节一致（生成物确定性）。
func TestBuildTreeDeterministic(t *testing.T) {
	a, err := buildTree("testdata/left-tree.json", "testdata", "testdata/i18n")
	if err != nil {
		t.Fatalf("buildTree: %v", err)
	}
	b, err := buildTree("testdata/left-tree.json", "testdata", "testdata/i18n")
	if err != nil {
		t.Fatalf("buildTree: %v", err)
	}
	if renderNodes(a, 0) != renderNodes(b, 0) {
		t.Error("两次构建渲染应字节一致")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
