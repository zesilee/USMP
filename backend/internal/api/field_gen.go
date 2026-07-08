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

// buildYangSchemaNested generates a *nested tree* form schema: containers become
// type=group FieldDefs and lists become type=list FieldDefs, each carrying their
// child fields in .Fields (recursively). This preserves structure that the flat
// collectFields loses — required for list-in-list configs like VLAN member-ports.
// R05: the whole tree is derived from the loaded YANG model, nothing hand-written.
func buildYangSchemaNested(mod schema.Module) YangSchema {
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
	for _, ch := range mod.Root().Children() {
		ys.Fields = append(ys.Fields, nodeToNestedField(ch))
	}
	return ys
}

// nodeToNestedField maps a schema node to a nested FieldDef tree.
func nodeToNestedField(node schema.Node) FieldDef {
	// 注意 case 顺序：list 节点同时满足 ListNode 与 ContainerNode 接口，
	// 必须先判 ListNode（与扁平 collectFields 一致），否则 list 会被误判为 group。
	switch n := node.(type) {
	case schema.LeafNode:
		return leafToField(n, "")
	case schema.ChoiceNode:
		// choice → 互斥分支节点：每个 case 递归携带其子字段（扁平 path 保持不变）。
		f := FieldDef{Path: n.Path(), Type: "choice", Label: n.Name(), Readonly: n.ReadOnly()}
		for _, cs := range n.Cases() {
			cd := CaseDef{Name: cs.Name(), Label: cs.Name()}
			for _, ch := range cs.Children() {
				cd.Fields = append(cd.Fields, nodeToNestedField(ch))
			}
			f.Cases = append(f.Cases, cd)
		}
		return f
	case schema.ListNode:
		f := FieldDef{Path: n.Path(), Type: "list", Label: n.Name(), OperationExclude: n.OperationExcludes(), Readonly: n.ReadOnly()}
		for _, ch := range n.Children() {
			f.Fields = append(f.Fields, nodeToNestedField(ch))
		}
		return f
	case schema.ContainerNode:
		// presence 容器 + 容器级 when/must 一并透出（BR-08）：presence=开关语义，
		// must 门禁该节点能否存在（如 IFM ipv4-conflict-enable）。
		f := FieldDef{
			Path: n.Path(), Type: "group", Label: n.Name(),
			Presence: n.IsPresence(), When: n.WhenExpr(), OperationExclude: n.OperationExcludes(),
			// config false 子树降级只读视图（BR-09）。
			Readonly: n.ReadOnly(),
		}
		for _, expr := range n.MustExprs() {
			f.Must = append(f.Must, MustRule{Expr: expr, Message: n.Description()})
		}
		for _, ch := range n.Children() {
			f.Fields = append(f.Fields, nodeToNestedField(ch))
		}
		return f
	}
	return FieldDef{Path: node.Path(), Label: node.Name(), Type: "string"}
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
	case schema.ChoiceNode:
		// 扁平形态下 choice 透明：递归展开各 case 的成员叶（保留其扁平 path）。
		for _, cs := range n.Cases() {
			for _, ch := range cs.Children() {
				collectFields(ch, group, fields, listCols)
			}
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
		When:     leaf.WhenExpr(),
		// 厂商呈现扩展（BR-07）：supportFilter 驱动高级搜索，operationExclude 驱动
		// 编辑态字段禁用/操作门禁；isKey 供通用控制台派生 keyField。
		SupportFilter:    leaf.SupportFilter(),
		OperationExclude: leaf.OperationExcludes(),
		IsKey:            leaf.IsKey(),
		// config false 叶（含继承）编辑态禁用且不入 payload（BR-09）。
		Readonly: leaf.ReadOnly(),
		// 系统动态缺省（BR-10）+ 单位后缀（BR-09）。
		DynamicDefault: leaf.DynamicDefault(),
		Units:          leaf.Units(),
	}
	if leaf.LeafType() == schema.LeafTypeEnum {
		for _, v := range leaf.EnumValues() {
			f.Options = append(f.Options, Option{Label: v, Value: v})
		}
	}
	if dv := leaf.DefaultValue(); dv != nil {
		f.Default = dv
	}
	// must 约束：message 取叶 description 兜底（YANG 无 error-message），空则前端生成。
	for _, expr := range leaf.MustExprs() {
		f.Must = append(f.Must, MustRule{Expr: expr, Message: leaf.Description()})
	}
	if p := leaf.Pattern(); p != "" {
		f.Pattern = p
	}
	if mn, ok := leaf.RangeMin(); ok {
		f.Minimum = mn
	}
	if mx, ok := leaf.RangeMax(); ok {
		f.Maximum = mx
	}
	// leaf-list → 前端渲染为可增删的多值输入（元素类型/枚举选项仍随 f 携带）。
	if leaf.IsLeafList() {
		f.Type = "leaf-list"
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
