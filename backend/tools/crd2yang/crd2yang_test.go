package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
)

// C2Y-01/02/03 —— CRD→YANG codegen：golden 逐字节一致（regen-and-diff 同款方法论）、
// 生成物可被 goyang 解析、元信息注解约定（revision 必填 + 缺省确定性派生）、
// 类型映射与 crdgen 互逆、可映射集之外 fail-fast 报字段 JSON 路径。

var update = flag.Bool("update", false, "rewrite golden files")

func mustReadFixture(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "businessvlannets.crd.yaml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return b
}

func TestGenerateYangGolden(t *testing.T) {
	got, err := Generate(mustReadFixture(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	goldenPath := filepath.Join("testdata", "businessvlannets.golden.yang")
	if *update {
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatalf("update golden: %v", err)
		}
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (run `go test ./tools/crd2yang -update` to create): %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("generated YANG drifts from golden\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// 生成物必须是合法 YANG：goyang 解析零错误，且单顶层容器约定闭合（crdgen 可反向消费）。
func TestGeneratedYangParses(t *testing.T) {
	got, err := Generate(mustReadFixture(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "usmp-business-vlan-net.yang"), got, 0o644); err != nil {
		t.Fatalf("write temp yang: %v", err)
	}
	ms := yang.NewModules()
	ms.AddPath(dir)
	if err := ms.Read("usmp-business-vlan-net"); err != nil {
		t.Fatalf("goyang read: %v", err)
	}
	if errs := ms.Process(); len(errs) > 0 {
		t.Fatalf("goyang process: %v", errs[0])
	}
	m := ms.Modules["usmp-business-vlan-net"]
	if m == nil {
		t.Fatal("module usmp-business-vlan-net not found after parse")
	}
	entry := yang.ToEntry(m)

	var containers []*yang.Entry
	for _, child := range entry.Dir {
		if child.IsContainer() {
			containers = append(containers, child)
		}
	}
	if len(containers) != 1 || containers[0].Name != "business-vlan-net" {
		t.Fatalf("want single top container business-vlan-net, got %d containers", len(containers))
	}

	root := containers[0]
	// 结构性契约：即使 golden 被无脑 -update 也拦住关键映射丢失。
	vlanID := root.Dir["vlan-id"]
	if vlanID == nil || vlanID.Type == nil || vlanID.Type.Kind != yang.Yuint16 {
		t.Errorf("vlan-id should map integer[1,4094] to uint16, got %+v", vlanID)
	}
	if vlanID != nil && vlanID.Mandatory != yang.TSTrue {
		t.Error("vlan-id (required) should be mandatory true")
	}
	if vlanID != nil && vlanID.Type != nil {
		if len(vlanID.Type.Range) == 0 || vlanID.Type.Range[0].Min.Value != 1 || vlanID.Type.Range[0].Max.Value != 4094 {
			t.Errorf("vlan-id range should be 1..4094, got %v", vlanID.Type.Range)
		}
	}
	devices := root.Dir["devices"]
	if devices == nil || !devices.IsList() || devices.Key != "ip" {
		t.Fatalf("devices should be a list with key ip, got %+v", devices)
	}
	if ip := devices.Dir["ip"]; ip == nil || ip.Mandatory == yang.TSTrue {
		t.Error("list key ip must exist and must NOT carry mandatory true (key implies required)")
	}
	if ap := devices.Dir["access-ports"]; ap == nil || !ap.IsLeafList() {
		t.Error("access-ports should be a leaf-list")
	}
	if vt := root.Dir["vlan-type"]; vt == nil || vt.Type == nil || vt.Type.Kind != yang.Yenum {
		t.Error("vlan-type should map enum to enumeration")
	} else if names := vt.Type.Enum.Names(); len(names) != 2 || names[0] != "common" || names[1] != "super" {
		// 注意：goyang Enum.Names() 按字典序返回，此断言校验的是**名字集合**
		// 完整（common/super 恰为字典序）；enum 序差在往返对账中双侧排序归一。
		t.Errorf("vlan-type enum names should be [common super], got %v", names)
	}
	if en := root.Dir["enabled"]; en == nil || en.Type == nil || en.Type.Kind != yang.Ybool {
		t.Error("enabled should map boolean to boolean")
	}
	if qos := root.Dir["qos"]; qos == nil || !qos.IsContainer() {
		t.Error("qos should map nested object to container")
	} else if prio := qos.Dir["priority"]; prio == nil || prio.Type == nil || prio.Type.Kind != yang.Yuint8 {
		t.Error("qos/priority integer[0,7] should map to uint8")
	}
	if root.Dir["syncState"] != nil {
		t.Error(".status subtree must be dropped entirely (BIC-04)")
	}
	if name := root.Dir["name"]; name == nil || name.Type == nil || len(name.Type.Pattern) != 1 || name.Type.Pattern[0] != "[A-Za-z0-9_-]{1,31}" {
		t.Errorf("name should keep pattern verbatim, got %+v", name)
	}
}

// crdYAML 组一个最小 CRD：annotations 与 spec schema 由用例注入。
func crdYAML(annotations, specProps string) []byte {
	return []byte(fmt.Sprintf(`apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: samples.biz.usmp.io
  annotations:
%s
spec:
  group: biz.usmp.io
  names:
    kind: Sample
    listKind: SampleList
    plural: samples
    singular: sample
  scope: Namespaced
  versions:
    - name: v1alpha1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
%s
`, annotations, specProps))
}

const revOnly = `    usmp.io/yang-revision: "2026-08-04"`

// C2Y-02 全缺省派生：仅 revision 注解时按确定性规则派生，两次运行逐字节一致。
func TestGenerateDefaultDerivation(t *testing.T) {
	in := crdYAML(revOnly, `              type: object
              properties:
                enabled:
                  type: boolean`)
	got1, err := Generate(in)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	got2, err := Generate(in)
	if err != nil {
		t.Fatalf("Generate twice: %v", err)
	}
	if string(got1) != string(got2) {
		t.Fatal("Generate is not deterministic")
	}
	s := string(got1)
	for _, want := range []string{
		"module usmp-sample {",
		`namespace "urn:usmp:yang:usmp-sample";`,
		"prefix us;",
		"revision 2026-08-04 {",
		"container sample {",
		`us:task-name "business-network";`,
		"extension task-name {",
		"generated-from: samples.biz.usmp.io",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("derived output missing %q\n--- got ---\n%s", want, s)
		}
	}
}

// C2Y-03 映射分支（golden 之外的构造）：表格驱动。
func TestGenerateMappingBranches(t *testing.T) {
	cases := []struct {
		name  string
		props string
		wants []string
	}{
		{
			name: "signed integer picks smallest int width",
			props: `              type: object
              properties:
                offset:
                  type: integer
                  minimum: -100
                  maximum: 100`,
			wants: []string{"type int8 {", `range "-100..100";`},
		},
		{
			name: "unbounded integer maps to int64",
			props: `              type: object
              properties:
                counter:
                  type: integer`,
			wants: []string{"type int64;"},
		},
		{
			name: "multi pattern allOf",
			props: `              type: object
              properties:
                code:
                  type: string
                  allOf:
                    - pattern: "[A-Z].*"
                    - pattern: ".{1,8}"`,
			wants: []string{`pattern "[A-Z].*";`, `pattern ".{1,8}";`},
		},
		{
			name: "uint32 width",
			props: `              type: object
              properties:
                index:
                  type: integer
                  minimum: 0
                  maximum: 100000`,
			wants: []string{"type uint32 {", `range "0..100000";`},
		},
		{
			// 全量基类型边界 → 裸类型：goyang 给无 range 整型附带全量 Range，
			// crdgen 会输出这类边界，反向必须收敛回裸类型（C2Y-04 闭环）。
			name: "full base-type range collapses to bare uint16",
			props: `              type: object
              properties:
                port:
                  type: integer
                  minimum: 0
                  maximum: 65535`,
			wants: []string{"type uint16;"},
		},
		{
			name: "full int64 range collapses to bare int64",
			props: `              type: object
              properties:
                counter:
                  type: integer
                  minimum: -9223372036854775808
                  maximum: 9223372036854775807`,
			wants: []string{"type int64;"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Generate(crdYAML(revOnly, tc.props))
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			for _, w := range tc.wants {
				if !strings.Contains(string(got), w) {
					t.Errorf("output missing %q\n--- got ---\n%s", w, got)
				}
			}
		})
	}
}
