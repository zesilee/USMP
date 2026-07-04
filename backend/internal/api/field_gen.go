package api

import (
	"strings"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// buildYangSchema generates a dynamic-form YangSchema from a loaded YANG module's
// schema tree, replacing the hard-coded per-module field definitions. Every
// configurable leaf becomes a FieldDef; list keys additionally become table
// columns (ListCols). This is the backend half of R05 (YANG-driven auto-render).
func buildYangSchema(mod schema.Module) YangSchema {
	vendor := mod.Vendor()
	if vendor == "" {
		vendor = vendorForNamespace(mod.Namespace())
	}
	ys := YangSchema{
		Module: mod.Name(),
		Title:  mod.Root().Name(),
		Vendor: vendor,
		Fields: []FieldDef{},
	}
	collectFields(mod.Root(), mod.Root().Name(), &ys.Fields, &ys.ListCols)
	return ys
}

// collectFields walks a schema node, appending a FieldDef for each leaf (grouped
// under the nearest container/list name) and a ListCols entry for each list key.
func collectFields(node schema.Node, group string, fields *[]FieldDef, listCols *[]FieldDef) {
	switch n := node.(type) {
	case schema.LeafNode:
		*fields = append(*fields, leafToField(n, group))
	case schema.ListNode:
		for _, k := range n.Keys() {
			*listCols = append(*listCols, leafToField(k, n.Name()))
		}
		for _, ch := range n.Children() {
			collectFields(ch, n.Name(), fields, listCols)
		}
	case schema.ContainerNode:
		for _, ch := range n.Children() {
			collectFields(ch, group, fields, listCols)
		}
	}
}

// leafToField maps a schema leaf to a form FieldDef.
func leafToField(leaf schema.LeafNode, group string) FieldDef {
	f := FieldDef{
		Path:     leaf.Path(),
		Type:     fieldType(leaf.LeafType()),
		Label:    leaf.Name(),
		Required: leaf.Mandatory(),
		Group:    group,
	}
	if leaf.LeafType() == schema.LeafTypeEnum {
		for _, v := range leaf.EnumValues() {
			f.Options = append(f.Options, Option{Label: v, Value: v})
		}
	}
	if dv := leaf.DefaultValue(); dv != nil {
		f.Default = dv
	}
	return f
}

// fieldType maps a YANG leaf type to a dynamic-form control type.
func fieldType(t schema.LeafType) string {
	switch t {
	case schema.LeafTypeBoolean:
		return "boolean"
	case schema.LeafTypeInt8, schema.LeafTypeInt16, schema.LeafTypeInt32, schema.LeafTypeInt64,
		schema.LeafTypeUint8, schema.LeafTypeUint16, schema.LeafTypeUint32, schema.LeafTypeUint64,
		schema.LeafTypeDecimal64:
		return "number"
	case schema.LeafTypeEnum:
		return "enum"
	default:
		return "string"
	}
}

// vendorForNamespace derives a vendor label from a module namespace.
func vendorForNamespace(ns string) string {
	l := strings.ToLower(ns)
	switch {
	case strings.Contains(l, "huawei"):
		return "huawei"
	case strings.Contains(l, "openconfig"):
		return "openconfig"
	default:
		return "unknown"
	}
}
