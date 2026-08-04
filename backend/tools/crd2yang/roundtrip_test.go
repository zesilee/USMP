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
	if testing.Short() {
		t.Skip("round-trip spawns `go run ../crdgen` (B2-style integration), skipped in -short")
	}

	in := mustReadFixture(t)
	yangOut, err := Generate(in)
	if err != nil {
		t.Fatalf("crd2yang Generate: %v", err)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "usmp-business-vlan-net.yang"), yangOut, 0o644); err != nil {
		t.Fatalf("write yang: %v", err)
	}
	outPath := filepath.Join(dir, "roundtrip.crd.yaml")
	cmd := exec.Command("go", "run", "../crdgen",
		"-path="+dir,
		"-module=usmp-business-vlan-net",
		"-group=biz.usmp.io",
		"-version=v1alpha1",
		"-kind=BusinessVlanNet",
		"-plural=businessvlannets",
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
