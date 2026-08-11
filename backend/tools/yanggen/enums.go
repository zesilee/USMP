// enums.go — 枚举/identityref/union 的登记与命名（codegen-conventions.md §4/§5）。
package main

import (
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

// registerEnum registers an enumeration type and returns its E_ type name.
// typedef enum → E_<CamelCase(使用方叶所属模块)>_<CamelCase(typedef名)>（union
// 内追加 _Enum）；内联 enumeration → E_<宿主struct>_<CamelCase(叶名)>。
func (b *builder) registerEnum(hostStruct string, e *yang.Entry, t *yang.YangType, inUnion bool) string {
	var name string
	if t.Name != "" && t.Name != "enumeration" {
		// 冻结规则（对拍实证）：typedef 枚举按**使用方叶的所属模块**命名，非
		// typedef 定义模块——pub-type:row-status 被 acl/time-range 实例化为
		// E_HuaweiAcl_RowStatus / E_HuaweiTimeRange_RowStatus 各一份。
		name = "E_" + yang.CamelCase(belongingModule(e)) + "_" + yang.CamelCase(t.Name)
		if inUnion {
			name += "_Enum"
		}
	} else {
		// 内联枚举：grouping 复用共享同一 AST → 单枚举类型、首次实例化命名
		//（遍历序确定即命名确定）。
		if e.Node != nil {
			if existing, ok := b.enumByNode[e.Node]; ok {
				return existing
			}
		}
		name = "E_" + hostStruct + "_" + FieldName(e.Name)
		if e.Node != nil {
			b.enumByNode[e.Node] = name
		}
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

// registerUnion resolves a union leaf's Go type（e 已是定义 union 的叶：leafref
// 在 mapType 先行解析到目标）。**同型折叠**（冻结自 ygot 行为）：全部成员映射
// 到同一 Go 类型时不生成接口，直接用该类型（string|string 的 ip 地址类 typedef
// 全走此路径——huawei 闭包 265 处 union 叶折叠后仅剩 6 个真接口）；异型才生成
// 接口 <宿主struct>_<CamelCase(叶名)>_Union + 包装类型。
// 返回 (Go 类型, 是否指针标量)。
func (b *builder) registerUnion(hostStruct string, e *yang.Entry, t *yang.YangType) (string, bool) {
	var members []string
	seen := map[string]bool{}
	memberPtr := false
	for _, mt := range t.Type {
		got, ptr, err := b.mapType(hostStruct, e, mt, true)
		if err != nil || got == "interface{}" {
			continue // 不支持的成员跳过（-ignore_unsupported 同精神）
		}
		if !seen[got] {
			seen[got] = true
			members = append(members, got)
			memberPtr = ptr
		}
	}
	if len(members) == 0 {
		return "interface{}", false
	}
	if len(members) == 1 {
		return members[0], memberPtr // 同型折叠
	}
	name := hostStruct + "_" + FieldName(e.Name) + "_Union"
	if _, ok := b.m.unionIdx[name]; !ok {
		b.m.unionIdx[name] = &Union{Name: name, Members: members}
	}
	return name, false
}

func bare(name string) string {
	if _, b, ok := strings.Cut(name, ":"); ok {
		return b
	}
	return name
}
