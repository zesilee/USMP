// roundtrip_test.go —— C2Y-04 往返对账：示例 CRD →crd2yang→ YANG →crdgen→ CRD′，
// 归一化后 spec schema 深度相等（零失真）。这是两张映射表互逆性的最强门禁：
// 任何一侧的失真（宽度推导、required/key 处理、pattern 转义、enum 顺序）都会在此爆红。
package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func TestRoundTripAgainstCrdgen(t *testing.T) {
	assertRoundTrip(t, mustReadFixture(t), "usmp-business-vlan-net", "BusinessVlanNet", "businessvlannets")
}

// C2Y-04 分支扩展：示例 CRD 之外的可映射构造也纳入往返——无界整型（goyang
// 基类型全量 Range 陷阱）、全量边界收敛、多 pattern allOf、多 key list、
// list 内非 key mandatory、嵌套 leaf-list。
func TestRoundTripSyntheticBranches(t *testing.T) {
	in := []byte(`apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: samples.biz.usmp.io
  annotations:
    usmp.io/yang-revision: "2026-08-04"
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
              type: object
              required:
                - mode
              properties:
                counter:
                  type: integer
                ratio:
                  type: integer
                  minimum: 0
                  maximum: 255
                code:
                  type: string
                  allOf:
                    - pattern: "[A-Z].*"
                    - pattern: ".{1,8}"
                mode:
                  type: string
                  enum:
                    - primary
                    - backup
                endpoints:
                  type: array
                  x-kubernetes-list-type: map
                  x-kubernetes-list-map-keys:
                    - ip
                    - port-name
                  items:
                    type: object
                    required:
                      - ip
                      - port-name
                      - weight
                    properties:
                      ip:
                        type: string
                      port-name:
                        type: string
                      weight:
                        type: integer
                        minimum: 1
                        maximum: 100
                      tags:
                        type: array
                        items:
                          type: string
`)
	assertRoundTrip(t, in, "usmp-sample", "Sample", "samples")
}

// assertRoundTrip 驱动 CRD→crd2yang→YANG→crdgen→CRD′ 并断言归一化零失真。
func assertRoundTrip(t *testing.T, in []byte, module, kind, plural string) {
	t.Helper()
	if testing.Short() {
		t.Skip("round-trip spawns `go run ../crdgen` (B2-style integration), skipped in -short")
	}

	yangOut, err := Generate(in)
	if err != nil {
		t.Fatalf("crd2yang Generate: %v", err)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, module+".yang"), yangOut, 0o644); err != nil {
		t.Fatalf("write yang: %v", err)
	}
	outPath := filepath.Join(dir, "roundtrip.crd.yaml")
	cmd := exec.Command("go", "run", "../crdgen",
		"-path="+dir,
		"-module="+module,
		"-group=biz.usmp.io",
		"-version=v1alpha1",
		"-kind="+kind,
		"-plural="+plural,
		"-output="+outPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("crdgen round-trip failed: %v\n%s", err, out)
	}
	roundtripped, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read crdgen output: %v", err)
	}

	specIn := specSchemaOf(t, in)
	specOut := specSchemaOf(t, roundtripped)
	normalize(specIn)
	normalize(specOut)

	jsonIn := mustJSON(t, specIn)
	jsonOut := mustJSON(t, specOut)
	if jsonIn != jsonOut {
		t.Errorf("round-trip drift (normalized spec schemas differ)\n--- input ---\n%s\n--- crd2yang→crdgen ---\n%s", jsonIn, jsonOut)
	}
}

// specSchemaOf extracts the storage version's .spec schema from a CRD manifest.
func specSchemaOf(t *testing.T, manifest []byte) *apiextv1.JSONSchemaProps {
	t.Helper()
	var crd apiextv1.CustomResourceDefinition
	if err := sigsyaml.Unmarshal(manifest, &crd); err != nil {
		t.Fatalf("parse CRD: %v", err)
	}
	for i := range crd.Spec.Versions {
		v := &crd.Spec.Versions[i]
		if !v.Storage {
			continue
		}
		if v.Schema == nil || v.Schema.OpenAPIV3Schema == nil {
			t.Fatal("storage version has no schema")
		}
		s, ok := v.Schema.OpenAPIV3Schema.Properties["spec"]
		if !ok {
			t.Fatal("no spec property")
		}
		return &s
	}
	t.Fatal("no storage version")
	return nil
}

// normalize strips the representation differences the two directions are
// allowed to have (D5)：description（crdgen 不输出）、x-kubernetes-*（crdgen
// 不输出，list key 并回 items.required——与 crdgen mergeRequired 同规则）、
// required 排序与空值归一。约束语义（type/min/max/pattern/enum/结构）必须全保持。
func normalize(s *apiextv1.JSONSchemaProps) {
	if s == nil {
		return
	}
	s.Description = ""
	// 全量基类型边界视同无界（goyang 对无 range 的整型附带基类型全量 Range，
	// crdgen 因此为「无界」整型输出 ±全量边界；见 emit.go fullRanges）。
	if s.Type == "integer" && s.Minimum != nil && s.Maximum != nil {
		for _, fr := range fullRanges {
			if *s.Minimum == fr.min && *s.Maximum == fr.max {
				s.Minimum, s.Maximum = nil, nil
				break
			}
		}
	}
	if len(s.XListMapKeys) > 0 && s.Items != nil && s.Items.Schema != nil {
		merged := append([]string{}, s.Items.Schema.Required...)
		for _, k := range s.XListMapKeys {
			seen := false
			for _, r := range merged {
				if r == k {
					seen = true
					break
				}
			}
			if !seen {
				merged = append(merged, k)
			}
		}
		s.Items.Schema.Required = merged
	}
	s.XListMapKeys = nil
	s.XListType = nil
	// enum 序：goyang Enum.Names() 按字典序返回，crdgen 因此不保 CRD 声明序。
	// RFC7951 载荷按名传输（不用 enum 数值），序差属表示差，双侧排序归一。
	sort.Slice(s.Enum, func(i, j int) bool { return string(s.Enum[i].Raw) < string(s.Enum[j].Raw) })
	sort.Strings(s.Required)
	if len(s.Required) == 0 {
		s.Required = nil
	}
	for name := range s.Properties {
		c := s.Properties[name]
		normalize(&c)
		s.Properties[name] = c
	}
	if len(s.Properties) == 0 {
		s.Properties = nil
	}
	if s.Items != nil {
		normalize(s.Items.Schema)
	}
	for i := range s.AllOf {
		normalize(&s.AllOf[i])
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}
