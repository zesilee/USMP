package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// BIC-01 —— YANG→CRD codegen golden：产物与入库 golden 逐字节一致（regen-and-diff
// 门禁同款方法论）；类型映射矩阵由 fixture 覆盖；不可映射类型与违反单顶层容器
// 约定 fail-fast 报错而非静默降级。

var update = flag.Bool("update", false, "rewrite golden files")

func TestGenerateCRDGolden(t *testing.T) {
	got, err := Generate(Options{
		Path:    "testdata",
		Module:  "usmp-crdgen-fixture",
		Group:   "biz.usmp.io",
		Version: "v1alpha1",
		Kind:    "FixtureService",
		Plural:  "fixtureservices",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	goldenPath := filepath.Join("testdata", "fixtureservices.golden.yaml")
	if *update {
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatalf("update golden: %v", err)
		}
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (run `go test ./tools/crdgen -update` to create): %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("generated CRD drifts from golden\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// golden 之外的结构性断言：即使 golden 被无脑 -update，也拦住关键契约丢失。
func TestGenerateCRDStructuralContract(t *testing.T) {
	got, err := Generate(Options{
		Path:    "testdata",
		Module:  "usmp-crdgen-fixture",
		Group:   "biz.usmp.io",
		Version: "v1alpha1",
		Kind:    "FixtureService",
		Plural:  "fixtureservices",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	s := string(got)
	for _, want := range []string{
		"apiVersion: apiextensions.k8s.io/v1",
		"kind: CustomResourceDefinition",
		"name: fixtureservices.biz.usmp.io",
		"vlan-id",
		"maximum: 4094",
		"minimum: 1",
		"pattern:",
		"- access",
		"- trunk",
		"type: boolean",
		// list key ip 必填
		"- ip",
		// YANG mandatory 叶子必填
		"- vlan-id",
		// status 子资源与结构
		"status: {}",
		"observedGeneration",
		"deviceStates",
		"claims",
		"conditions",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("generated CRD missing %q", want)
		}
	}
}

// 不可映射类型（union）fail-fast，错误信息带叶子定位。
func TestGenerateCRDUnsupportedTypeFailFast(t *testing.T) {
	_, err := Generate(Options{
		Path:    "testdata",
		Module:  "usmp-crdgen-union",
		Group:   "biz.usmp.io",
		Version: "v1alpha1",
		Kind:    "UnionService",
		Plural:  "unionservices",
	})
	if err == nil {
		t.Fatal("Generate should fail on union type")
	}
	if !strings.Contains(err.Error(), "union") || !strings.Contains(err.Error(), "value") {
		t.Errorf("error should name the unsupported type and leaf, got: %v", err)
	}
}

// 单顶层容器约定：违反即报错。
func TestGenerateCRDRequiresSingleTopContainer(t *testing.T) {
	_, err := Generate(Options{
		Path:    "testdata",
		Module:  "usmp-crdgen-twotop",
		Group:   "biz.usmp.io",
		Version: "v1alpha1",
		Kind:    "TwoTop",
		Plural:  "twotops",
	})
	if err == nil {
		t.Fatal("Generate should fail when module has more than one top-level container")
	}
	if !strings.Contains(err.Error(), "top-level container") {
		t.Errorf("error should explain the single-container convention, got: %v", err)
	}
}
