// negative_test.go —— C2Y-01/02/03 负路径与并发安全：fail-fast 报字段 JSON 路径、
// 禁静默降级/丢弃（含独立代码检视揪出的四类修复的回归用例）。
package main

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// C2Y-01/02/03 负路径：fail-fast 且错误信息可定位。
func TestGenerateNegativePaths(t *testing.T) {
	cases := []struct {
		name    string
		in      []byte
		wantErr []string
	}{
		{
			name: "missing revision annotation",
			in: crdYAML(`    usmp.io/yang-module: usmp-sample`, `              type: object
              properties:
                enabled:
                  type: boolean`),
			wantErr: []string{"usmp.io/yang-revision"},
		},
		{
			name: "object array without list-map-keys",
			in: crdYAML(revOnly, `              type: object
              properties:
                rules:
                  type: array
                  items:
                    type: object
                    properties:
                      action:
                        type: string`),
			wantErr: []string{"rules", "x-kubernetes-list-map-keys"},
		},
		{
			name: "oneOf fail-fast",
			in: crdYAML(revOnly, `              type: object
              properties:
                value:
                  oneOf:
                    - type: string
                    - type: integer`),
			wantErr: []string{"value", "oneOf"},
		},
		{
			name: "format fail-fast",
			in: crdYAML(revOnly, `              type: object
              properties:
                ts:
                  type: string
                  format: date-time`),
			wantErr: []string{"ts", "format"},
		},
		{
			name: "enum on integer fail-fast",
			in: crdYAML(revOnly, `              type: object
              properties:
                level:
                  type: integer
                  enum:
                    - 1
                    - 2`),
			wantErr: []string{"level"},
		},
		{
			name: "single-bounded integer fail-fast",
			in: crdYAML(revOnly, `              type: object
              properties:
                partial:
                  type: integer
                  minimum: 1`),
			wantErr: []string{"partial"},
		},
		{
			name: "non-integral bound fail-fast",
			in: crdYAML(revOnly, `              type: object
              properties:
                ratio:
                  type: integer
                  minimum: 0.5
                  maximum: 2`),
			wantErr: []string{"ratio"},
		},
		{
			name: "missing spec property",
			in: []byte(`apiVersion: apiextensions.k8s.io/v1
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
`),
			wantErr: []string{"spec"},
		},
		{
			name: "not a CRD",
			in: []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: x`),
			wantErr: []string{"CustomResourceDefinition"},
		},
		{
			// float64 无法精确表示的边界：静默回绕会腐蚀 range（检视必修 1）。
			name: "integer bound beyond float64 exact range fail-fast",
			in: crdYAML(revOnly, `              type: object
              properties:
                big:
                  type: integer
                  minimum: 0
                  maximum: 9223372036854775807`),
			wantErr: []string{"big"},
		},
		{
			// CRD 属性名是任意 JSON key，非法 YANG 标识符必须拦（检视必修 2）。
			name: "property name not a YANG identifier",
			in: crdYAML(revOnly, `              type: object
              properties:
                "3com port":
                  type: string`),
			wantErr: []string{"3com port", "identifier"},
		},
		{
			// RFC7950：标识符不得以 xml（任意大小写）开头。
			name: "property name with xml prefix",
			in: crdYAML(revOnly, `              type: object
              properties:
                xmlData:
                  type: string`),
			wantErr: []string{"xmlData", "identifier"},
		},
		{
			// 跨类型约束静默丢弃（检视必修 3）：integer 上的 pattern。
			name: "pattern on integer fail-fast",
			in: crdYAML(revOnly, `              type: object
              properties:
                level:
                  type: integer
                  pattern: "[0-9]+"`),
			wantErr: []string{"level", "pattern"},
		},
		{
			// 跨类型约束静默丢弃（检视必修 3）：object 上的 enum。
			name: "enum on object fail-fast",
			in: crdYAML(revOnly, `              type: object
              properties:
                cfg:
                  type: object
                  enum:
                    - a`),
			wantErr: []string{"cfg", "enum"},
		},
		{
			// 跨类型约束静默丢弃（检视必修 3）：string 上的 minimum。
			name: "minimum on string fail-fast",
			in: crdYAML(revOnly, `              type: object
              properties:
                label:
                  type: string
                  minimum: 1
                  maximum: 3`),
			wantErr: []string{"label", "minimum"},
		},
		{
			// guard 缺口（检视必修 4）：patternProperties 携带约束不得静默丢弃。
			name: "patternProperties fail-fast",
			in: crdYAML(revOnly, `              type: object
              properties:
                extra:
                  type: object
                  patternProperties:
                    "^x-":
                      type: string`),
			wantErr: []string{"extra", "patternProperties"},
		},
		{
			// required 引用不存在的属性：静默忽略会让契约悄悄缩水。
			name: "required references missing property",
			in: crdYAML(revOnly, `              type: object
              required:
                - ghost
              properties:
                enabled:
                  type: boolean`),
			wantErr: []string{"ghost"},
		},
		{
			// 空串枚举值是非法 YANG，goyang 必解析失败，须在生成前拦截。
			name: "empty enum value fail-fast",
			in: crdYAML(revOnly, `              type: object
              properties:
                mode:
                  type: string
                  enum:
                    - ""`),
			wantErr: []string{"mode", "enum"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Generate(tc.in)
			if err == nil {
				t.Fatal("Generate should fail")
			}
			for _, w := range tc.wantErr {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error should mention %q, got: %v", w, err)
				}
			}
		})
	}
}

// 多 served 版本且无唯一 storage：fail-fast（C2Y-01 场景）。
func TestGenerateAmbiguousVersionFailFast(t *testing.T) {
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
      storage: false
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
    - name: v1alpha2
      served: true
      storage: false
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
`)
	_, err := Generate(in)
	if err == nil {
		t.Fatal("Generate should fail without a unique storage version")
	}
	if !strings.Contains(err.Error(), "storage") {
		t.Errorf("error should explain storage-version ambiguity, got: %v", err)
	}
}

// R09：纯函数并发安全（-race 下多协程生成结果一致）。
func TestGenerateConcurrent(t *testing.T) {
	in := mustReadFixture(t)
	base, err := Generate(in)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := Generate(in)
			if err != nil {
				errs <- err
				return
			}
			if string(got) != string(base) {
				errs <- fmt.Errorf("concurrent result drifts")
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}
