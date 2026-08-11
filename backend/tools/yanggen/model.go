// model.go — Entry 树 → 生成模型（codegen-conventions.md §1/§2 类型映射与命名）。
// 产出与 ygot 结构约定冻结等价的中间模型，emit 层据此渲染源码。
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

// Model is the complete generation model for one package.
type Model struct {
	Package string
	Structs []*Struct // 含 Device fakeroot，按类型名字典序
	Enums   []*Enum   // 按类型名字典序
	Unions  []*Union  // 按接口名字典序

	structIdx map[string]*Struct
	enumIdx   map[string]*Enum
	unionIdx  map[string]*Union
}

// Struct is one generated container/list-member type.
type Struct struct {
	Name    string
	Path    string   // schema 路径（注释用），如 /huawei-vlan/vlan/vlans/vlan
	Fields  []*Field // 按 Go 字段名字典序
	Keys    []*Field // list 成员的 key 字段，按 YANG key 语句顺序；nil=非 list 成员
	KeyName string   // 复合键 struct 名（"<Name>_Key"），单键/非 list 为空
	// OrderedKey 非空 = ordered-by user 列表成员：宿主字段为 *<Name>_OrderedMap，
	// emit 层生成保序容器类型（keys 切片 + valueMap，冻结 ygot 字段形状）。
	OrderedKey string
}

// Field is one generated struct field.
type Field struct {
	GoName   string
	YangName string
	Module   string // module tag 值（belonging module）；_Key 字段渲染时忽略
	Type     string // 渲染后的 Go 类型表达式
	Ptr      bool   // 指针标量（ListKeyMap nil 检查用）
}

// Enum is one generated E_ type.
type Enum struct {
	Name   string // 含 E_ 前缀
	Values []EnumValue
}

// EnumValue is one enum constant.
type EnumValue struct {
	ConstName string
	RawName   string // 未净化原始值名（映射表 Name 字段）
	Value     int64  // 常量数值（YANG value+1）
	DefMod    string // identityref 才填 DefiningModule
}

// Union is one generated union interface + wrappers.
type Union struct {
	Name    string   // 接口名（..._Union）
	Members []string // 成员 Go 类型（非指针，如 "uint16"/"string"/"E_X"），按声明序
}

type builder struct {
	m       *Model
	modules map[string]*yang.Module
	// rootDir：全闭包顶层容器索引（裸名→entry），跨模块 leafref 解析用
	//（/ifm:ifm/... 指向别的模块树；顶层容器名在闭包内唯一）。
	rootDir map[string]*yang.Entry
	// enumByNode：内联枚举按 AST 节点去重（grouping 复用时 goyang 展开共享
	// 同一 AST——ygot 只生成一个枚举类型、命名取首次实例化，对拍实证）。
	enumByNode map[yang.Node]string
}

// BuildModel walks the loaded module entries into a generation model.
// moduleNames 决定顶层容器归属与遍历顺序（排序保证确定性）。
func BuildModel(pkg string, entries map[string]*yang.Entry, mods map[string]*yang.Module) (*Model, error) {
	b := &builder{
		m: &Model{
			Package:   pkg,
			structIdx: map[string]*Struct{},
			enumIdx:   map[string]*Enum{},
			unionIdx:  map[string]*Union{},
		},
		modules:    mods,
		rootDir:    map[string]*yang.Entry{},
		enumByNode: map[yang.Node]string{},
	}
	for _, mname := range sortedNames(entries) {
		for cname, c := range entries[mname].Dir {
			if !c.IsLeaf() && !c.IsLeafList() && c.RPC == nil {
				b.rootDir[cname] = c
			}
		}
	}

	device := &Struct{Name: "Device", Path: "/device"}
	b.m.structIdx[device.Name] = device
	for _, mname := range sortedNames(entries) {
		root := entries[mname]
		for _, cname := range sortedNames(root.Dir) {
			c := root.Dir[cname]
			if c.IsLeaf() || c.IsLeafList() || c.RPC != nil {
				continue // 顶层 rpc/叶不入配置树（与 ygot fakeroot 行为一致）
			}
			st, err := b.buildStruct(mname, c, []string{c.Name})
			if err != nil {
				return nil, err
			}
			device.Fields = append(device.Fields, &Field{
				GoName: FieldName(c.Name), YangName: c.Name,
				Module: belongingModule(c), Type: "*" + st.Name,
			})
		}
	}
	sortFields(device.Fields)

	for _, s := range b.m.structIdx {
		b.m.Structs = append(b.m.Structs, s)
	}
	sort.Slice(b.m.Structs, func(i, j int) bool { return b.m.Structs[i].Name < b.m.Structs[j].Name })
	for _, e := range b.m.enumIdx {
		b.m.Enums = append(b.m.Enums, e)
	}
	sort.Slice(b.m.Enums, func(i, j int) bool { return b.m.Enums[i].Name < b.m.Enums[j].Name })
	for _, u := range b.m.unionIdx {
		b.m.Unions = append(b.m.Unions, u)
	}
	sort.Slice(b.m.Unions, func(i, j int) bool { return b.m.Unions[i].Name < b.m.Unions[j].Name })
	return b.m, nil
}

// buildStruct converts a container/list entry into a Struct (recursing into
// children). segs 是数据路径段（不含 choice/case，augment 用宿主路径）。
func (b *builder) buildStruct(rootModule string, e *yang.Entry, segs []string) (*Struct, error) {
	name := StructName(rootModule, segs)
	if st, ok := b.m.structIdx[name]; ok {
		return st, nil // augment 并入宿主后可能被两侧遍历到；同名即同型
	}
	st := &Struct{Name: name, Path: "/" + rootModule + "/" + strings.Join(segs, "/")}
	b.m.structIdx[name] = st

	keyNames := strings.Fields(e.Key)
	keySet := map[string]bool{}
	for _, k := range keyNames {
		keySet[k] = true
	}

	var walk func(children map[string]*yang.Entry) error
	walk = func(children map[string]*yang.Entry) error {
		for _, cname := range sortedNames(children) {
			c := children[cname]
			switch {
			case c.IsChoice():
				// choice/case 不占数据路径段也不生成类型：成员拍平进宿主。
				for _, caseName := range sortedNames(c.Dir) {
					cs := c.Dir[caseName]
					if cs.IsCase() {
						if err := walk(cs.Dir); err != nil {
							return err
						}
					} else if err := walk(map[string]*yang.Entry{cs.Name: cs}); err != nil {
						return err // shorthand case：裸成员
					}
				}
			case c.RPC != nil:
				continue
			case c.IsLeafList():
				elem, _, err := b.scalarType(st.Name, c)
				if err != nil {
					return err
				}
				st.Fields = append(st.Fields, &Field{
					GoName: FieldName(c.Name), YangName: c.Name,
					Module: belongingModule(c), Type: "[]" + elem,
				})
			case c.IsLeaf():
				elem, ptr, err := b.scalarType(st.Name, c)
				if err != nil {
					return err
				}
				typ := elem
				if ptr {
					typ = "*" + elem
				}
				f := &Field{
					GoName: FieldName(c.Name), YangName: c.Name,
					Module: belongingModule(c), Type: typ, Ptr: ptr,
				}
				st.Fields = append(st.Fields, f)
				if keySet[c.Name] {
					// Keys 按 YANG key 语句顺序回填（walk 结束后重排）。
					st.Keys = append(st.Keys, f)
				}
			case c.IsList():
				child, err := b.buildStruct(rootModule, c, append(append([]string{}, segs...), c.Name))
				if err != nil {
					return err
				}
				keyType, err := b.listKeyType(child, c)
				if err != nil {
					return err
				}
				typ := "map[" + keyType + "]*" + child.Name
				if len(strings.Fields(c.Key)) == 0 {
					typ = "[]*" + child.Name // 无 key list（当前闭包 0 处）
				}
				if orderedByUser(c) {
					if len(strings.Fields(c.Key)) != 1 {
						return fmt.Errorf("yanggen: %s: ordered-by user 仅支持单键列表（约定冻结）", child.Name)
					}
					child.OrderedKey = keyType
					typ = "*" + child.Name + "_OrderedMap"
				}
				st.Fields = append(st.Fields, &Field{
					GoName: FieldName(c.Name), YangName: c.Name,
					Module: belongingModule(c), Type: typ,
				})
			default: // container
				child, err := b.buildStruct(rootModule, c, append(append([]string{}, segs...), c.Name))
				if err != nil {
					return err
				}
				st.Fields = append(st.Fields, &Field{
					GoName: FieldName(c.Name), YangName: c.Name,
					Module: belongingModule(c), Type: "*" + child.Name,
				})
			}
		}
		return nil
	}
	if err := walk(e.Dir); err != nil {
		return nil, err
	}
	sortFields(st.Fields)

	// Keys 重排为 YANG key 语句顺序。
	if len(keyNames) > 0 {
		byYang := map[string]*Field{}
		for _, f := range st.Keys {
			byYang[f.YangName] = f
		}
		st.Keys = st.Keys[:0]
		for _, k := range keyNames {
			f, ok := byYang[k]
			if !ok {
				return nil, fmt.Errorf("yanggen: %s: key 叶 %q 不在子节点中", st.Path, k)
			}
			st.Keys = append(st.Keys, f)
		}
		if len(keyNames) > 1 {
			st.KeyName = st.Name + "_Key"
		}
	}
	return st, nil
}

// listKeyType returns the map key Go type for a keyed list.
func (b *builder) listKeyType(child *Struct, e *yang.Entry) (string, error) {
	keys := strings.Fields(e.Key)
	switch len(keys) {
	case 0:
		return "", nil
	case 1:
		for _, f := range child.Keys {
			if f.YangName == keys[0] {
				return strings.TrimPrefix(f.Type, "*"), nil
			}
		}
		return "", fmt.Errorf("yanggen: list %s key %q 未解析", child.Name, keys[0])
	default:
		return child.Name + "_Key", nil
	}
}

// scalarType maps a leaf/leaf-list entry's YANG type to (Go 元素类型, 是否指针标量)。
func (b *builder) scalarType(hostStruct string, e *yang.Entry) (string, bool, error) {
	return b.mapType(hostStruct, e, e.Type, false)
}

func (b *builder) mapType(hostStruct string, e *yang.Entry, t *yang.YangType, inUnion bool) (string, bool, error) {
	if t == nil {
		return "string", true, nil
	}
	switch t.Kind {
	case yang.Ystring:
		return "string", true, nil
	case yang.Ybool:
		return "bool", true, nil
	case yang.Yint8:
		return "int8", true, nil
	case yang.Yint16:
		return "int16", true, nil
	case yang.Yint32:
		return "int32", true, nil
	case yang.Yint64:
		return "int64", true, nil
	case yang.Yuint8:
		return "uint8", true, nil
	case yang.Yuint16:
		return "uint16", true, nil
	case yang.Yuint32:
		return "uint32", true, nil
	case yang.Yuint64:
		return "uint64", true, nil
	case yang.Ydecimal64:
		return "float64", true, nil
	case yang.Yempty:
		return "object.Empty", false, nil
	case yang.Ybinary:
		return "object.Binary", false, nil
	case yang.Yenum:
		return b.registerEnum(hostStruct, e, t, inUnion), false, nil
	case yang.Yidentityref:
		return b.registerIdentity(t), false, nil
	case yang.Yleafref:
		target := b.resolveLeafref(e, t)
		if target == nil {
			warnUnsupported(e, "unresolved leafref "+t.Path)
			return "interface{}", false, nil
		}
		return b.mapTypeAtTarget(target, inUnion)
	case yang.Yunion:
		typ, ptr := b.registerUnion(hostStruct, e, t)
		return typ, ptr, nil
	default:
		// bits/instance-identifier/anyxml 等：-ignore_unsupported 语义 → interface{}。
		warnUnsupported(e, t.Kind.String())
		return "interface{}", false, nil
	}
}

// mapTypeAtTarget maps a leafref target leaf's type（宿主取目标叶所在 struct）。
func (b *builder) mapTypeAtTarget(target *yang.Entry, inUnion bool) (string, bool, error) {
	host, err := entryStructName(target.Parent)
	if err != nil {
		warnUnsupported(target, "leafref target host: "+err.Error())
		return "interface{}", false, nil
	}
	return b.mapType(host, target, target.Type, inUnion)
}

func warnUnsupported(e *yang.Entry, why string) {
	fmt.Fprintf(os.Stderr, "yanggen: %s: 类型不支持（%s）→ interface{}\n", e.Path(), why)
}

func sortFields(fs []*Field) {
	sort.Slice(fs, func(i, j int) bool { return fs[i].GoName < fs[j].GoName })
}

// belongingModule 返回节点的 module tag 值：submodule 归属父模块、augment 归属
// 来源模块（goyang 的 Entry.Node 保留定义方 AST，RootNode 即定义模块）。
func belongingModule(e *yang.Entry) string {
	if e.Node == nil {
		return ""
	}
	root := yang.RootNode(e.Node)
	if root == nil {
		return ""
	}
	if root.BelongsTo != nil {
		return root.BelongsTo.Name // submodule → 父模块
	}
	return root.Name
}

// entryStructName 由任意 container/list entry 反推其生成类型名（leafref 目标
// 宿主命名用）：沿 Parent 链回到模块根，段序列即数据路径。
func entryStructName(e *yang.Entry) (string, error) {
	var segs []string
	cur := e
	for cur != nil && cur.Parent != nil {
		if !cur.IsChoice() && !cur.IsCase() {
			segs = append([]string{cur.Name}, segs...)
		}
		cur = cur.Parent
	}
	if cur == nil || len(segs) == 0 {
		return "", fmt.Errorf("无法定位模块根")
	}
	return StructName(cur.Name, segs), nil
}

// resolveLeafref resolves a leafref path to its target entry（失败返回 nil，
// 调用方降级 interface{}——与 -ignore_unsupported 同精神）。
//
// 必须按**数据树**语义解析：YANG leafref 路径不含 choice/case 层级，而 goyang
// Entry 树保留 choice/case——`../` 计数在含 choice 的子树上会错位（acl
// source-pool-name 实证），故不用 Entry.Find。
func (b *builder) resolveLeafref(e *yang.Entry, t *yang.YangType) *yang.Entry {
	if t.Path == "" {
		return nil
	}
	segs := strings.Split(strings.Trim(t.Path, "/"), "/")
	cur := e
	if strings.HasPrefix(t.Path, "/") {
		cur = nil // 绝对路径：首段直接查全闭包顶层容器索引
	}
	for _, seg := range segs {
		if seg == ".." {
			if cur == nil {
				return nil
			}
			cur = dataParent(cur)
			continue
		}
		name := bare(seg)
		if cur == nil || cur.Parent == nil {
			// 虚拟设备根（绝对路径首段，或相对路径爬出模块根后再下钻）：
			// 顶层容器跨模块索引。
			if c, ok := b.rootDir[name]; ok {
				cur = c
				continue
			}
			if cur == nil {
				return nil
			}
		}
		cur = dataChild(cur, name)
	}
	if cur == nil || !cur.IsLeaf() && !cur.IsLeafList() {
		return nil
	}
	return cur
}

// orderedByUser reports whether a list entry is `ordered-by user`.
func orderedByUser(e *yang.Entry) bool {
	return e.ListAttr != nil && e.ListAttr.OrderedBy != nil && e.ListAttr.OrderedBy.Name == "user"
}

// dataParent 返回数据树父节点（跳过 choice/case 层）。
func dataParent(e *yang.Entry) *yang.Entry {
	p := e.Parent
	for p != nil && (p.IsChoice() || p.IsCase()) {
		p = p.Parent
	}
	return p
}

// dataChild 在数据树语义下取子节点（穿透 choice/case 层查找）。
func dataChild(e *yang.Entry, name string) *yang.Entry {
	if c, ok := e.Dir[name]; ok && !c.IsChoice() && !c.IsCase() {
		return c
	}
	for _, c := range e.Dir {
		if c.IsChoice() || c.IsCase() {
			if r := dataChild(c, name); r != nil {
				return r
			}
		}
	}
	return nil
}
