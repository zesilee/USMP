package main

import "testing"

// buildTree：层级/双语保留、叶子解析出根容器（RPC 不算）、模块文件缺失的叶
// 容器为空且不阻断（LT-01 正/负路径）。
func TestBuildTree(t *testing.T) {
	nodes, err := buildTree("testdata/left-tree.json", "testdata")
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
}

// 负路径：JSON 缺失/畸形明确报错（R08 不产半成品）。
func TestBuildTreeNegative(t *testing.T) {
	if _, err := buildTree("testdata/nope.json", "testdata"); err == nil {
		t.Error("missing json should error")
	}
	if _, err := buildTree("testdata/demo-good.yang", "testdata"); err == nil {
		t.Error("malformed json should error")
	}
}

// renderNodes/countLeaves：生成物字面量确定性与叶子计数（LT-01 生成器内核）。
func TestRenderNodesAndCount(t *testing.T) {
	nodes := []TreeNode{
		{Zh: "组", En: "G", Children: []TreeNode{
			{Zh: "叶", En: "L", SourceModule: "demo-good", RootContainers: []string{"a", "b"}},
		}},
	}
	out := renderNodes(nodes, 0)
	for _, want := range []string{`Zh: "组"`, `SourceModule: "demo-good"`, `RootContainers: []string{"a", "b"}`, "Children: []LeftTreeNode{"} {
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

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
