// Command crdgen generates a Kubernetes CRD manifest from a business-intent
// YANG module (BIC-01/BIC-02): YANG is the single schema source, the CRD
// OpenAPI v3 schema is derived at build time and committed (regen-and-diff
// gate, R04 methodology).
//
// Conventions:
//   - the module MUST have exactly one top-level data container; its subtree
//     becomes the CR .spec schema (property names keep the YANG leaf names
//     verbatim so the payload stays 1:1 with RFC7951/ygot JSON);
//   - types outside the mappable set (union/leafref/identityref/decimal64/
//     binary/bits/empty) fail fast — no silent downgrade;
//   - the status subresource schema is fixed (BIC-04): observedGeneration,
//     conditions, deviceStates, claims.
package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

// Options parameterizes one CRD generation run.
type Options struct {
	Path    string // directory containing the .yang modules
	Module  string // YANG module name
	Group   string // API group, e.g. biz.usmp.io
	Version string // API version, e.g. v1alpha1
	Kind    string // CR kind, e.g. BusinessVlanService
	Plural  string // lowercase plural resource name
}

// Generate parses the YANG module and renders the CRD manifest YAML.
func Generate(opts Options) ([]byte, error) {
	root, err := loadModuleRoot(opts.Path, opts.Module)
	if err != nil {
		return nil, err
	}

	specSchema, err := mapEntry(root)
	if err != nil {
		return nil, err
	}

	crd := buildCRD(opts, root, specSchema)
	out, err := sigsyaml.Marshal(crd)
	if err != nil {
		return nil, fmt.Errorf("marshal CRD: %w", err)
	}
	return out, nil
}

// loadModuleRoot parses the module and returns its single top-level data
// container (the .spec shape).
func loadModuleRoot(dir, module string) (*yang.Entry, error) {
	ms := yang.NewModules()
	ms.AddPath(dir)
	if err := ms.Read(module); err != nil {
		return nil, fmt.Errorf("read %s: %w", module, err)
	}
	if errs := ms.Process(); len(errs) > 0 {
		return nil, fmt.Errorf("process %s: %v", module, errs[0])
	}
	m, ok := ms.Modules[module]
	if !ok {
		return nil, fmt.Errorf("module %s not found after parse", module)
	}
	entry := yang.ToEntry(m)

	var containers []*yang.Entry
	for _, child := range entry.Dir {
		if child.IsContainer() {
			containers = append(containers, child)
		}
	}
	if len(containers) != 1 {
		return nil, fmt.Errorf("module %s must have exactly one top-level container (found %d): the container subtree becomes the CR spec", module, len(containers))
	}
	return containers[0], nil
}

// mapEntry maps a YANG entry (container/list/leaf-list/leaf) to a JSON schema.
func mapEntry(e *yang.Entry) (*apiextv1.JSONSchemaProps, error) {
	switch {
	case e.IsList():
		item, err := mapObject(e)
		if err != nil {
			return nil, err
		}
		// list keys are required on each item.
		var keys []string
		if e.Key != "" {
			keys = strings.Fields(e.Key)
			sort.Strings(keys)
		}
		item.Required = mergeRequired(item.Required, keys)
		return &apiextv1.JSONSchemaProps{
			Type:  "array",
			Items: &apiextv1.JSONSchemaPropsOrArray{Schema: item},
		}, nil
	case e.IsLeafList():
		item, err := mapScalar(e)
		if err != nil {
			return nil, err
		}
		return &apiextv1.JSONSchemaProps{
			Type:  "array",
			Items: &apiextv1.JSONSchemaPropsOrArray{Schema: item},
		}, nil
	case e.IsContainer():
		return mapObject(e)
	default: // leaf
		return mapScalar(e)
	}
}

// mapObject maps the children of a container/list entry to an object schema;
// mandatory leaves become required properties.
func mapObject(e *yang.Entry) (*apiextv1.JSONSchemaProps, error) {
	props := map[string]apiextv1.JSONSchemaProps{}
	var required []string
	for _, name := range sortedChildNames(e) {
		child := e.Dir[name]
		cs, err := mapEntry(child)
		if err != nil {
			return nil, err
		}
		props[name] = *cs
		if child.Mandatory == yang.TSTrue {
			required = append(required, name)
		}
	}
	sort.Strings(required)
	return &apiextv1.JSONSchemaProps{
		Type:       "object",
		Properties: props,
		Required:   required,
	}, nil
}

// mapScalar maps a leaf's YANG type to a scalar JSON schema. Unsupported
// kinds fail fast with the leaf location (BIC-01 负路径).
func mapScalar(e *yang.Entry) (*apiextv1.JSONSchemaProps, error) {
	t := e.Type
	if t == nil {
		return nil, fmt.Errorf("leaf %s: missing type", e.Path())
	}
	switch t.Kind {
	case yang.Yint8, yang.Yint16, yang.Yint32, yang.Yint64,
		yang.Yuint8, yang.Yuint16, yang.Yuint32, yang.Yuint64:
		s := &apiextv1.JSONSchemaProps{Type: "integer"}
		if len(t.Range) > 0 {
			min := float64(t.Range[0].Min.Value)
			max := float64(t.Range[len(t.Range)-1].Max.Value)
			if t.Range[0].Min.Negative {
				min = -min
			}
			if t.Range[len(t.Range)-1].Max.Negative {
				max = -max
			}
			s.Minimum = &min
			s.Maximum = &max
		}
		return s, nil
	case yang.Ystring:
		s := &apiextv1.JSONSchemaProps{Type: "string"}
		if len(t.Pattern) == 1 {
			s.Pattern = t.Pattern[0]
		} else if len(t.Pattern) > 1 {
			for _, p := range t.Pattern {
				s.AllOf = append(s.AllOf, apiextv1.JSONSchemaProps{Pattern: p})
			}
		}
		return s, nil
	case yang.Ybool:
		return &apiextv1.JSONSchemaProps{Type: "boolean"}, nil
	case yang.Yenum:
		s := &apiextv1.JSONSchemaProps{Type: "string"}
		for _, name := range t.Enum.Names() {
			s.Enum = append(s.Enum, apiextv1.JSON{Raw: []byte(fmt.Sprintf("%q", name))})
		}
		return s, nil
	default:
		return nil, fmt.Errorf("leaf %s: YANG type %q is not mappable to OpenAPI — restrict the intent model to the mappable set (integer/string/boolean/enumeration and their lists), see BIC-01", e.Path(), t.Kind)
	}
}

// statusSchema is the fixed status-subresource schema (BIC-04).
func statusSchema() *apiextv1.JSONSchemaProps {
	str := apiextv1.JSONSchemaProps{Type: "string"}
	return &apiextv1.JSONSchemaProps{
		Type: "object",
		Properties: map[string]apiextv1.JSONSchemaProps{
			"observedGeneration": {Type: "integer", Format: "int64"},
			"conditions": {
				Type: "array",
				Items: &apiextv1.JSONSchemaPropsOrArray{Schema: &apiextv1.JSONSchemaProps{
					Type:     "object",
					Required: []string{"status", "type"},
					Properties: map[string]apiextv1.JSONSchemaProps{
						"type":               str,
						"status":             str,
						"reason":             str,
						"message":            str,
						"lastTransitionTime": {Type: "string", Format: "date-time"},
					},
				}},
			},
			"deviceStates": {
				Type: "array",
				Items: &apiextv1.JSONSchemaPropsOrArray{Schema: &apiextv1.JSONSchemaProps{
					Type:     "object",
					Required: []string{"device", "phase"},
					Properties: map[string]apiextv1.JSONSchemaProps{
						"device":         str,
						"phase":          {Type: "string", Enum: enumJSON("pending", "synced", "failed")},
						"reason":         str,
						"lastTransition": {Type: "string", Format: "date-time"},
					},
				}},
			},
			"claims": {
				Type: "array",
				Items: &apiextv1.JSONSchemaPropsOrArray{Schema: &apiextv1.JSONSchemaProps{
					Type:     "object",
					Required: []string{"device", "module", "path"},
					Properties: map[string]apiextv1.JSONSchemaProps{
						"device": str,
						"module": str,
						"path":   str,
					},
				}},
			},
		},
	}
}

func buildCRD(opts Options, root *yang.Entry, specSchema *apiextv1.JSONSchemaProps) *apiextv1.CustomResourceDefinition {
	return &apiextv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1",
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s.%s", opts.Plural, opts.Group),
			Annotations: map[string]string{
				// regen-and-diff 门禁锚点：标记生成来源模块，禁止手改。
				"usmp.io/generated-from": opts.Module,
			},
		},
		Spec: apiextv1.CustomResourceDefinitionSpec{
			Group: opts.Group,
			Scope: apiextv1.NamespaceScoped,
			Names: apiextv1.CustomResourceDefinitionNames{
				Kind:     opts.Kind,
				ListKind: opts.Kind + "List",
				Plural:   opts.Plural,
				Singular: strings.ToLower(opts.Kind),
			},
			Versions: []apiextv1.CustomResourceDefinitionVersion{{
				Name:    opts.Version,
				Served:  true,
				Storage: true,
				Subresources: &apiextv1.CustomResourceSubresources{
					Status: &apiextv1.CustomResourceSubresourceStatus{},
				},
				Schema: &apiextv1.CustomResourceValidation{
					OpenAPIV3Schema: &apiextv1.JSONSchemaProps{
						Type: "object",
						Properties: map[string]apiextv1.JSONSchemaProps{
							"spec":   *specSchema,
							"status": *statusSchema(),
						},
					},
				},
				AdditionalPrinterColumns: printerColumns(root),
			}},
		},
	}
}

// printerColumns exposes up to three top-level scalar leaves plus the
// Converged condition (kubectl 直写是受支持接入方式，BIC-02).
func printerColumns(root *yang.Entry) []apiextv1.CustomResourceColumnDefinition {
	var cols []apiextv1.CustomResourceColumnDefinition
	for _, name := range sortedChildNames(root) {
		if len(cols) == 3 {
			break
		}
		child := root.Dir[name]
		if !child.IsLeaf() || child.Type == nil {
			continue
		}
		colType := "string"
		switch child.Type.Kind {
		case yang.Yint8, yang.Yint16, yang.Yint32, yang.Yint64,
			yang.Yuint8, yang.Yuint16, yang.Yuint32, yang.Yuint64:
			colType = "integer"
		case yang.Ybool:
			colType = "boolean"
		}
		cols = append(cols, apiextv1.CustomResourceColumnDefinition{
			Name:     name,
			Type:     colType,
			JSONPath: ".spec." + name,
		})
	}
	cols = append(cols,
		apiextv1.CustomResourceColumnDefinition{
			Name:     "Converged",
			Type:     "string",
			JSONPath: `.status.conditions[?(@.type=="Converged")].status`,
		},
		apiextv1.CustomResourceColumnDefinition{
			Name:     "Age",
			Type:     "date",
			JSONPath: ".metadata.creationTimestamp",
		},
	)
	return cols
}

func sortedChildNames(e *yang.Entry) []string {
	names := make([]string, 0, len(e.Dir))
	for name := range e.Dir {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func mergeRequired(existing, extra []string) []string {
	seen := map[string]bool{}
	for _, r := range existing {
		seen[r] = true
	}
	out := append([]string{}, existing...)
	for _, r := range extra {
		if !seen[r] {
			out = append(out, r)
		}
	}
	sort.Strings(out)
	return out
}

func enumJSON(values ...string) []apiextv1.JSON {
	out := make([]apiextv1.JSON, 0, len(values))
	for _, v := range values {
		out = append(out, apiextv1.JSON{Raw: []byte(fmt.Sprintf("%q", v))})
	}
	return out
}
