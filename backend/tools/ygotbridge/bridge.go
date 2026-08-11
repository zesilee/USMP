// Package ygotbridge 把 ygot 生成包的 schema（goyang yang.Entry 树）转换为框架
// Schema IR（retire-ygot-runtime 阶段1.5，原 pkg/yang-runtime/schema/entry.go 迁出）。
//
// 本包是**构建期/测试专用**：允许 import ygot/goyang（YN-05 守护豁免面）；发布
// 二进制的运行时闭包不得引用本包。转换语义与原 entry.go 逐字段等价，由
// internal/yangschema 的 IR blob 字节对拍兜底（bridge(generated) == 入库 blob）。
package ygotbridge

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ytypes"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// AddYgotSchema populates ds with the modules described by an ygot-generated
// *ytypes.Schema（vendor 留空）。
func AddYgotSchema(ds *schema.DefaultSchema, s *ytypes.Schema) error {
	return AddYgotSchemaWithVendor(ds, s, "")
}

// AddYgotSchemaWithVendor converts the resolved goyang yang.Entry tree into IR
// module DTOs and materializes them into ds. The ygot schema's root entry has
// each top-level YANG container as a child; each such child becomes a Module.
func AddYgotSchemaWithVendor(ds *schema.DefaultSchema, s *ytypes.Schema, vendor string) error {
	if ds == nil || s == nil {
		return nil
	}
	rootEntry := ygotRootEntry(s)
	if rootEntry == nil {
		return nil
	}
	for _, child := range sortedDir(rootEntry) {
		im := irModule(child, vendor)
		m, err := schema.ModuleFromIR(im)
		if err != nil {
			return fmt.Errorf("ygotbridge: %w", err)
		}
		ds.AddModule(m)
	}
	return nil
}

// ygotRootEntry finds the root yang.Entry of an ygot schema. The SchemaTree map
// is keyed by generated struct name; the root struct is s.Root, so its type name
// is the map key for the root entry.
func ygotRootEntry(s *ytypes.Schema) *yang.Entry {
	if s.SchemaTree == nil {
		return nil
	}
	if s.Root != nil {
		t := reflect.TypeOf(s.Root)
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if e, ok := s.SchemaTree[t.Name()]; ok {
			return e
		}
	}
	// Fallback: an entry with no parent that has children.
	for _, e := range s.SchemaTree {
		if e != nil && e.Parent == nil && e.Dir != nil {
			return e
		}
	}
	return nil
}

// sortedDir returns e's directory children ordered by name for deterministic output.
func sortedDir(e *yang.Entry) []*yang.Entry {
	if e == nil || e.Dir == nil {
		return nil
	}
	names := make([]string, 0, len(e.Dir))
	for n := range e.Dir {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]*yang.Entry, 0, len(names))
	for _, n := range names {
		out = append(out, e.Dir[n])
	}
	return out
}

func entryNamespace(e *yang.Entry) string {
	if ns := e.Namespace(); ns != nil {
		return ns.Name
	}
	return ""
}

// irModule wraps a top-level container entry as an IR module tagged with vendor.
func irModule(e *yang.Entry, vendor string) schema.IRModule {
	return schema.IRModule{
		Name:      e.Name,
		Namespace: entryNamespace(e),
		Vendor:    vendor,
		Root:      irContainer(e, "/"+e.Name, false),
	}
}

// irNode dispatches an entry to the appropriate node kind. inheritedRO carries
// an ancestor's `config false` down the subtree (YANG config inheritance, BR-09).
func irNode(e *yang.Entry, path string, inheritedRO bool) *schema.IRNode {
	switch {
	case e.IsLeafList():
		// Modeled as a leaf carrying the element type, flagged as a leaf-list so
		// the form renders repeatable scalar values.
		leaf := irLeaf(e, path, false, inheritedRO)
		leaf.LeafList = true
		return leaf
	case e.IsLeaf():
		return irLeaf(e, path, false, inheritedRO)
	case e.IsList():
		return irList(e, path, inheritedRO)
	default:
		return irContainer(e, path, inheritedRO)
	}
}

func irContainer(e *yang.Entry, path string, inheritedRO bool) *schema.IRNode {
	ro := inheritedRO || e.Config == yang.TSFalse
	out := &schema.IRNode{
		Kind: "container", Name: e.Name, Description: e.Description, Path: path,
		ReadOnly: ro,
		// presence containers survive the ygot gzip round-trip under
		// Extra["presence"] (same shape as when/must); existence toggles a
		// feature (BR-08).
		Presence:   len(e.Extra["presence"]) > 0,
		When:       firstExtraExpr(e.Extra["when"]),
		Must:       allExtraExprs(e.Extra["must"]),
		OpExcludes: extOperationExcludes(e),
	}
	for _, child := range sortedDir(e) {
		if child.IsChoice() {
			// A choice contributes no data-path segment: its case members inherit
			// this container's `path` so their data paths stay flat.
			out.Children = append(out.Children, irChoice(child, path, ro))
			continue
		}
		out.Children = append(out.Children, irNode(child, path+"/"+child.Name, ro))
	}
	return out
}

// irChoice converts a goyang choice Entry. parentPath is the enclosing
// container/list path — case members are flattened onto it (choice and case
// names never appear in data paths), keeping the NETCONF write path intact.
func irChoice(e *yang.Entry, parentPath string, inheritedRO bool) *schema.IRNode {
	ro := inheritedRO || e.Config == yang.TSFalse
	out := &schema.IRNode{
		Kind: "choice", Name: e.Name, Description: e.Description,
		Path: parentPath + "/" + e.Name, ReadOnly: ro,
	}
	// goyang resolves `default` on choices into the DefaultCase-bearing shape
	// only via statement extras; the runtime tree derived it from nothing else
	// than entry defaults — preserved as-is（原 entry.go 未填 defaultCase，行为等价）。
	for _, caseEntry := range sortedDir(e) {
		out.Cases = append(out.Cases, irCase(caseEntry, parentPath, ro))
	}
	return out
}

// irCase converts a goyang case Entry. A "shorthand" case (a bare node directly
// under the choice, not wrapped in `case`) is treated as an implicit
// single-member case. Members inherit parentPath (no case segment).
func irCase(e *yang.Entry, parentPath string, inheritedRO bool) *schema.IRNode {
	ro := inheritedRO || e.Config == yang.TSFalse
	out := &schema.IRNode{
		Kind: "case", Name: e.Name, Description: e.Description,
		Path: parentPath + "/" + e.Name, ReadOnly: ro,
	}
	if !e.IsCase() {
		// Shorthand case: the entry itself is the single member node.
		out.Children = append(out.Children, caseMember(e, parentPath, ro))
		return out
	}
	for _, child := range sortedDir(e) {
		out.Children = append(out.Children, caseMember(child, parentPath, ro))
	}
	return out
}

// caseMember builds one member of a case, flattening onto parentPath. A choice
// nested directly in a case (no intervening container) recurses as a nested choice.
func caseMember(e *yang.Entry, parentPath string, inheritedRO bool) *schema.IRNode {
	if e.IsChoice() {
		return irChoice(e, parentPath, inheritedRO)
	}
	return irNode(e, parentPath+"/"+e.Name, inheritedRO)
}

func irList(e *yang.Entry, path string, inheritedRO bool) *schema.IRNode {
	ro := inheritedRO || e.Config == yang.TSFalse
	keyNames := map[string]bool{}
	for _, k := range strings.Fields(e.Key) {
		keyNames[k] = true
	}
	out := &schema.IRNode{
		Kind: "list", Name: e.Name, Description: e.Description, Path: path,
		ReadOnly:    ro,
		MinElements: listMinElements(e),
		When:        firstExtraExpr(e.Extra["when"]),
		Must:        allExtraExprs(e.Extra["must"]),
		OpExcludes:  extOperationExcludes(e),
	}
	for _, child := range sortedDir(e) {
		if child.IsChoice() {
			out.Children = append(out.Children, irChoice(child, path, ro))
			continue
		}
		childPath := path + "/" + child.Name
		if child.IsLeaf() && keyNames[child.Name] {
			leaf := irLeaf(child, childPath, true, ro)
			out.Children = append(out.Children, leaf)
			// Keys 按 sortedDir 遇到顺序收集——与原 entry.go 的 keys 序一致。
			out.Keys = append(out.Keys, child.Name)
			continue
		}
		out.Children = append(out.Children, irNode(child, childPath, ro))
	}
	return out
}

func irLeaf(e *yang.Entry, path string, isKey bool, inheritedRO bool) *schema.IRNode {
	ltName, ok := schema.IRLeafTypeName(mapLeafType(e.Type))
	if !ok {
		// mapLeafType 只产合法枚举值；此分支不可达，保底 string（R08 不 panic）。
		ltName = "string"
	}
	out := &schema.IRNode{
		Kind: "leaf", Name: e.Name, Description: e.Description, Path: path,
		ReadOnly:       inheritedRO || e.Config == yang.TSFalse,
		LeafType:       ltName,
		IsKey:          isKey,
		Mandatory:      e.Mandatory.Value(),
		When:           firstExtraExpr(e.Extra["when"]),
		Must:           allExtraExprs(e.Extra["must"]),
		SupportFilter:  extSupportFilter(e),
		DynamicDefault: extDynamicDefault(e),
		OpExcludes:     extOperationExcludes(e),
	}
	if e.Type != nil {
		if e.Type.Enum != nil {
			out.Enums = append([]string(nil), e.Type.Enum.Names()...)
		}
		out.Units = e.Type.Units
		if e.Type.Default != "" {
			d := e.Type.Default
			out.Default = &d
		}
		if len(e.Type.Pattern) > 0 {
			out.Pattern = e.Type.Pattern[0]
		}
		if min, hasMin, max, hasMax := leafRangeBounds(e.Type); hasMin || hasMax {
			if hasMin {
				out.RangeMin = intPtr(min)
			}
			if hasMax {
				out.RangeMax = intPtr(max)
			}
		}
		if min, hasMin, max, hasMax := leafLengthBounds(e.Type); hasMin || hasMax {
			if hasMin {
				out.LengthMin = intPtr(min)
			}
			if hasMax {
				out.LengthMax = intPtr(max)
			}
		}
	}
	if out.Units == "" {
		// units may sit on the leaf statement itself rather than the (derived) type.
		out.Units = e.Units
	}
	return out
}

// listMinElements 提取 YANG min-elements（无 ListAttr = 0）。
func listMinElements(e *yang.Entry) uint64 {
	if e.ListAttr == nil {
		return 0
	}
	return e.ListAttr.MinElements
}

// LoadModuleEntries 构建期从 YANG 源目录装载模块闭包（AddPath 递归 + Read +
// Process + ToEntry；与 tools/yanggen 装载语义一致）。schemagen 直读源用
// （S4：不再经 generated 包 gzip schema）。
func LoadModuleEntries(paths, modules []string) (map[string]*yang.Entry, error) {
	ms := yang.NewModules()
	for _, p := range paths {
		if err := filepath.Walk(p, func(sub string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				ms.AddPath(sub)
			}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("ygotbridge: 模型目录不可用 %s: %w", p, err)
		}
	}
	for _, m := range modules {
		found := false
		for _, p := range paths {
			fn := filepath.Join(p, m+".yang")
			if _, err := os.Stat(fn); err == nil {
				if err := ms.Read(fn); err != nil {
					return nil, fmt.Errorf("ygotbridge: read %s: %w", m, err)
				}
				found = true
				break
			}
		}
		if !found {
			if err := ms.Read(m + ".yang"); err != nil {
				return nil, fmt.Errorf("ygotbridge: 模块 %s 未找到: %w", m, err)
			}
		}
	}
	if errs := ms.Process(); len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "ygotbridge: yang process warning: %v\n", e)
		}
	}
	entries := make(map[string]*yang.Entry)
	for name, mod := range ms.Modules {
		if strings.Contains(name, "@") || mod == nil {
			continue
		}
		if e := yang.ToEntry(mod); e != nil {
			entries[name] = e
		}
	}
	return entries, nil
}

// AddSourceModules 把直读源的模块闭包顶层容器转换入 ds（vendor 统一打标；
// 模块名排序 + 容器 sortedDir 口径保确定性——与 AddYgotSchemaWithVendor 一致）。
//
// stripDescriptions=true 剥离全部 description：gzip 往返历史行为不含描述
// （实测 5/5 抽样差异均为 desc、结构零差异），切换直读源时先冻结字节稳定；
// 描述增益作为独立增强另行拍板（GD-01 联动前端黄金）。
func AddSourceModules(ds *schema.DefaultSchema, entries map[string]*yang.Entry, vendor string, stripDescriptions bool) error {
	names := make([]string, 0, len(entries))
	for n := range entries {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, mname := range names {
		root := entries[mname]
		for _, c := range sortedDir(root) {
			if c.IsLeaf() || c.IsLeafList() || c.RPC != nil {
				continue
			}
			im := irModule(c, vendor)
			if stripDescriptions {
				stripDesc(im.Root)
			}
			m, err := schema.ModuleFromIR(im)
			if err != nil {
				return fmt.Errorf("ygotbridge: %s: %w", mname, err)
			}
			ds.AddModule(m)
		}
	}
	return nil
}

// stripDesc 递归清空 IR 节点描述（含 choice cases）。
func stripDesc(n *schema.IRNode) {
	if n == nil {
		return
	}
	n.Description = ""
	for _, c := range n.Children {
		stripDesc(c)
	}
	for _, c := range n.Cases {
		stripDesc(c)
	}
}
