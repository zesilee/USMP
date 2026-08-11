// enums.go — 枚举/identityref/union 的登记与命名（codegen-conventions.md §4/§5）。
package main

import (
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

// registerEnum registers an enumeration type and returns its E_ type name.
// typedef enum → E_<CamelCase(定义模块)>_<CamelCase(typedef名)>（union 内追加
// _Enum）；内联 enumeration → E_<宿主struct>_<CamelCase(叶名)>。
func (b *builder) registerEnum(hostStruct string, e *yang.Entry, t *yang.YangType, inUnion bool) string {
	var name string
	if t.Name != "" && t.Name != "enumeration" {
		mod := b.typedefModule(e, t.Name)
		name = "E_" + yang.CamelCase(mod) + "_" + yang.CamelCase(t.Name)
		if inUnion {
			name += "_Enum"
		}
	} else {
		name = "E_" + hostStruct + "_" + FieldName(e.Name)
	}
	if _, ok := b.m.enumIdx[name]; ok {
		return name
	}
	en := &Enum{Name: name}
	if t.Enum != nil {
		// NameMap 是唯一权威配对（Names()/Values() 各自独立排序、不可 zip）。
		nm := t.Enum.NameMap()
		type pair struct {
			n string
			v int64
		}
		pairs := make([]pair, 0, len(nm))
		for n, v := range nm {
			pairs = append(pairs, pair{n, v})
		}
		sort.Slice(pairs, func(i, j int) bool { return pairs[i].v < pairs[j].v })
		for _, p := range pairs {
			en.Values = append(en.Values, EnumValue{
				ConstName: EnumConstName(name, p.n),
				RawName:   p.n,
				Value:     p.v + 1, // 冻结约定：常量数值 = YANG value + 1（UNSET=0）
			})
		}
	}
	b.m.enumIdx[name] = en
	return name
}

// registerIdentity registers an identityref enum: E_<CamelCase(基identity定义
// 模块)>_<基identity名（不做 CamelCase）>；值 = 全部派生 identity，字典序编号
// 自 1（UNSET=0），每值携带其定义模块（RFC7951 §6.8 DefiningModule）。
func (b *builder) registerIdentity(t *yang.YangType) string {
	base := t.IdentityBase
	if base == nil {
		return "interface{}"
	}
	name := "E_" + yang.CamelCase(identityModule(base)) + "_" + base.Name
	if _, ok := b.m.enumIdx[name]; ok {
		return name
	}
	en := &Enum{Name: name}
	derived := append([]*yang.Identity{}, base.Values...)
	sort.Slice(derived, func(i, j int) bool { return derived[i].Name < derived[j].Name })
	for i, id := range derived {
		en.Values = append(en.Values, EnumValue{
			ConstName: EnumConstName(name, id.Name),
			RawName:   id.Name,
			Value:     int64(i) + 1,
			DefMod:    identityModule(id),
		})
	}
	b.m.enumIdx[name] = en
	return name
}

func identityModule(id *yang.Identity) string {
	root := yang.RootNode(id)
	if root == nil {
		return ""
	}
	if root.BelongsTo != nil {
		return root.BelongsTo.Name
	}
	return root.Name
}

// registerUnion registers a union interface for leaf e（e 已是定义 union 的叶：
// leafref 在 mapType 先行解析到目标）。接口名 = <宿主struct>_<CamelCase(叶名)>_Union。
func (b *builder) registerUnion(hostStruct string, e *yang.Entry, t *yang.YangType) string {
	name := hostStruct + "_" + FieldName(e.Name) + "_Union"
	if _, ok := b.m.unionIdx[name]; ok {
		return name
	}
	u := &Union{Name: name}
	b.m.unionIdx[name] = u // 先登记防递归（union 成员理论上可再嵌 union）
	seen := map[string]bool{}
	for _, mt := range t.Type {
		got, _, err := b.mapType(hostStruct, e, mt, true)
		if err != nil || got == "interface{}" {
			continue // 不支持的成员跳过（-ignore_unsupported 同精神）
		}
		if !seen[got] {
			seen[got] = true
			u.Members = append(u.Members, got)
		}
	}
	return name
}

// typedefModule 判定 typedef 的定义模块：优先看叶 AST 声明的类型名前缀
// （prefix 经 goyang 按定义方模块的 import 表解析——grouping 展开后 AST 仍属
// 定义方模块，语义正确）；无前缀即 AST 根模块。
func (b *builder) typedefModule(e *yang.Entry, typedefName string) string {
	if e.Node != nil {
		if astName := astTypeName(e.Node, typedefName); astName != "" {
			if pfx, _, ok := strings.Cut(astName, ":"); ok {
				if m := yang.FindModuleByPrefix(e.Node, pfx); m != nil {
					if m.BelongsTo != nil {
						return m.BelongsTo.Name
					}
					return m.Name
				}
			}
		}
		if root := yang.RootNode(e.Node); root != nil {
			if root.BelongsTo != nil {
				return root.BelongsTo.Name
			}
			return root.Name
		}
	}
	return ""
}

// astTypeName 从叶/leaf-list 的 AST 取声明类型名（可能带前缀）。typedefName
// 用于在 union AST 成员里匹配对应项。
func astTypeName(n yang.Node, typedefName string) string {
	var t *yang.Type
	switch l := n.(type) {
	case *yang.Leaf:
		t = l.Type
	case *yang.LeafList:
		t = l.Type
	default:
		return ""
	}
	if t == nil {
		return ""
	}
	if bare(t.Name) == typedefName {
		return t.Name
	}
	for _, mt := range t.Type { // union AST 成员
		if bare(mt.Name) == typedefName {
			return mt.Name
		}
	}
	return ""
}

func bare(name string) string {
	if _, b, ok := strings.Cut(name, ":"); ok {
		return b
	}
	return name
}
