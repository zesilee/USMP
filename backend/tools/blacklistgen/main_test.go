package main

import "testing"

// collectBlacklistedRoots：revision 匹配的模块产出其顶层数据容器名；revision
// 不匹配与文件缺失的条目跳过；RPC 不是数据容器（CN-03）。
func TestCollectBlacklistedRoots(t *testing.T) {
	got, err := collectBlacklistedRoots("testdata/blacklist.xml", "testdata")
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(got) != 1 || got[0] != "badroot" {
		t.Fatalf("got %v, want [badroot]（oldrev 版本不匹配、missing 无文件、rpc 非数据容器）", got)
	}
}

// 负路径：blacklist.xml 缺失/畸形须明确报错，不产出半成品（R08）。
func TestCollectBlacklistedRootsNegative(t *testing.T) {
	if _, err := collectBlacklistedRoots("testdata/nope.xml", "testdata"); err == nil {
		t.Error("missing blacklist should error")
	}
	if _, err := collectBlacklistedRoots("testdata/demo-bad.yang", "testdata"); err == nil {
		t.Error("malformed xml should error")
	}
}
