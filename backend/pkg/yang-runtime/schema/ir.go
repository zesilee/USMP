// Schema IR（retire-ygot-runtime YN-03）：框架内部 Schema 模型的自有序列化格式。
// 构建期工具（tools/schemagen）把 goyang 解析出的模型树编码为本格式入库，运行期
// 仅解码本格式——发布二进制由此不再依赖 ygot gzip schema 与 goyang Entry 类型。
//
// 线格式：gzip(JSON(irEnvelope))。确定性：模块按名排序、节点序即树序（children
// 有序切片）、gzip 固定参数无时间戳——同一树两次编码字节一致（CG-01 可复现契约）。
// 版本号 irVersion 破坏性变更时递增，运行期版本不符快速失败（不半解析运行）。
package schema

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

// irVersion is the wire-format version. Bump on breaking IR changes; DecodeIR
// rejects mismatches with an explicit regenerate hint.
const irVersion = 1

type irEnvelope struct {
	Version int        `json:"version"`
	Modules []irModule `json:"modules"`
}

type irModule struct {
	Name      string  `json:"name"`
	Namespace string  `json:"namespace,omitempty"`
	Revision  string  `json:"revision,omitempty"`
	Vendor    string  `json:"vendor,omitempty"`
	Root      *irNode `json:"root"`
}

// irNode is the serialized union of all node kinds. Kind selects which fields
// are meaningful; zero values are omitted from JSON to keep the blob compact.
type irNode struct {
	Kind        string   `json:"kind"` // container|list|leaf|choice|case
	Name        string   `json:"name"`
	Description string   `json:"desc,omitempty"`
	Path        string   `json:"path"`
	ReadOnly    bool     `json:"ro,omitempty"`
	When        string   `json:"when,omitempty"`
	Must        []string `json:"must,omitempty"`
	OpExcludes  []string `json:"opEx,omitempty"`

	// container
	Presence bool `json:"presence,omitempty"`

	// container/list/case
	Children []*irNode `json:"children,omitempty"`

	// list
	Keys        []string `json:"keys,omitempty"` // key leaf names, resolved against Children on decode
	UserOrdered bool     `json:"userOrdered,omitempty"`

	// choice
	DefaultCase string    `json:"defaultCase,omitempty"`
	Cases       []*irNode `json:"cases,omitempty"`

	// leaf
	LeafType       string   `json:"leafType,omitempty"`
	IsKey          bool     `json:"isKey,omitempty"`
	Mandatory      bool     `json:"mandatory,omitempty"`
	Default        *string  `json:"default,omitempty"`
	Enums          []string `json:"enums,omitempty"`
	Units          string   `json:"units,omitempty"`
	Pattern        string   `json:"pattern,omitempty"`
	RangeMin       *int     `json:"rangeMin,omitempty"`
	RangeMax       *int     `json:"rangeMax,omitempty"`
	LengthMin      *int     `json:"lengthMin,omitempty"`
	LengthMax      *int     `json:"lengthMax,omitempty"`
	LeafList       bool     `json:"leafList,omitempty"`
	SupportFilter  bool     `json:"supportFilter,omitempty"`
	DynamicDefault bool     `json:"dynamicDefault,omitempty"`
}

// leafTypeNames maps LeafType to its stable wire name. String names (not ints)
// keep the format self-describing and reorder-safe across versions.
var leafTypeNames = map[LeafType]string{
	LeafTypeBoolean: "boolean", LeafTypeInt8: "int8", LeafTypeInt16: "int16",
	LeafTypeInt32: "int32", LeafTypeInt64: "int64", LeafTypeUint8: "uint8",
	LeafTypeUint16: "uint16", LeafTypeUint32: "uint32", LeafTypeUint64: "uint64",
	LeafTypeString: "string", LeafTypeEnum: "enum", LeafTypeEmpty: "empty",
	LeafTypeDecimal64: "decimal64", LeafTypeBits: "bits",
}

var leafTypeFromName = func() map[string]LeafType {
	m := make(map[string]LeafType, len(leafTypeNames))
	for k, v := range leafTypeNames {
		m[v] = k
	}
	return m
}()

// EncodeIR serializes a DefaultSchema to the IR wire format (deterministic).
func EncodeIR(s *DefaultSchema) ([]byte, error) {
	if s == nil {
		return nil, fmt.Errorf("schema ir: nil schema")
	}
	env := irEnvelope{Version: irVersion}
	mods := s.Modules()
	sort.Slice(mods, func(i, j int) bool { return mods[i].Name() < mods[j].Name() })
	for _, m := range mods {
		root, err := encodeNode(m.Root())
		if err != nil {
			return nil, fmt.Errorf("schema ir: module %s: %w", m.Name(), err)
		}
		env.Modules = append(env.Modules, irModule{
			Name: m.Name(), Namespace: m.Namespace(), Revision: m.Revision(),
			Vendor: m.Vendor(), Root: root,
		})
	}
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression) // 固定级别，Header 零值无时间戳
	if err != nil {
		return nil, fmt.Errorf("schema ir: gzip writer: %w", err)
	}
	enc := json.NewEncoder(zw)
	if err := enc.Encode(env); err != nil {
		return nil, fmt.Errorf("schema ir: encode: %w", err)
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("schema ir: gzip: %w", err)
	}
	return buf.Bytes(), nil
}

func encodeNode(n Node) (*irNode, error) {
	if n == nil {
		return nil, fmt.Errorf("nil node")
	}
	out := &irNode{
		Name: n.Name(), Description: n.Description(), Path: n.Path(), ReadOnly: n.ReadOnly(),
	}
	switch t := n.(type) {
	case ListNode:
		out.Kind = "list"
		out.UserOrdered = t.IsUserOrdered()
		for _, k := range t.Keys() {
			out.Keys = append(out.Keys, k.Name())
		}
		out.OpExcludes = t.OperationExcludes() // 接口可得，不依赖具体类型
		if l, ok := t.(*defaultList); ok {
			// when/must 仅存活于具体类型；出现第二个 ListNode 实现时在此显式扩展，
			// 而非静默丢字段。
			out.When, out.Must = l.whenExpr, l.mustExprs
		}
		for _, c := range t.Children() {
			cn, err := encodeNode(c)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", n.Path(), err)
			}
			out.Children = append(out.Children, cn)
		}
	case ContainerNode:
		out.Kind = "container"
		out.Presence = t.IsPresence()
		out.When = t.WhenExpr()
		out.Must = t.MustExprs()
		out.OpExcludes = t.OperationExcludes()
		for _, c := range t.Children() {
			cn, err := encodeNode(c)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", n.Path(), err)
			}
			out.Children = append(out.Children, cn)
		}
	case ChoiceNode:
		out.Kind = "choice"
		out.DefaultCase = t.DefaultCase()
		for _, cs := range t.Cases() {
			cn, err := encodeNode(cs)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", n.Path(), err)
			}
			out.Cases = append(out.Cases, cn)
		}
	case CaseNode:
		out.Kind = "case"
		for _, c := range t.Children() {
			cn, err := encodeNode(c)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", n.Path(), err)
			}
			out.Children = append(out.Children, cn)
		}
	case LeafNode:
		out.Kind = "leaf"
		name, ok := leafTypeNames[t.LeafType()]
		if !ok {
			return nil, fmt.Errorf("%s: unknown leaf type %d", n.Path(), t.LeafType())
		}
		out.LeafType = name
		out.IsKey = t.IsKey()
		out.Mandatory = t.Mandatory()
		out.Enums = t.EnumValues()
		out.Units = t.Units()
		out.Pattern = t.Pattern()
		out.When = t.WhenExpr()
		out.Must = t.MustExprs()
		out.LeafList = t.IsLeafList()
		out.SupportFilter = t.SupportFilter()
		out.DynamicDefault = t.DynamicDefault()
		out.OpExcludes = t.OperationExcludes()
		if dv := t.DefaultValue(); dv != nil {
			s, ok := dv.(string)
			if !ok {
				// 现有构建路径（entry.go）默认值恒为字符串；异形默认值显式报错（R08）。
				return nil, fmt.Errorf("%s: non-string default %T", n.Path(), dv)
			}
			out.Default = &s
		}
		if v, ok := t.RangeMin(); ok {
			out.RangeMin = intPtr(v)
		}
		if v, ok := t.RangeMax(); ok {
			out.RangeMax = intPtr(v)
		}
		if dl, ok := t.(*defaultLeaf); ok {
			if v, ok := dl.LengthMin(); ok {
				out.LengthMin = intPtr(v)
			}
			if v, ok := dl.LengthMax(); ok {
				out.LengthMax = intPtr(v)
			}
		}
	default:
		return nil, fmt.Errorf("%s: unsupported node kind %T", n.Path(), n)
	}
	return out, nil
}

func intPtr(v int) *int { return &v }

// DecodeIR parses the IR wire format into a fully-linked DefaultSchema
// (parent pointers, list-key identity, path cache). Version mismatch and
// malformed input fail fast with explicit errors (R08 — never a half schema).
func DecodeIR(blob []byte) (*DefaultSchema, error) {
	if len(blob) == 0 {
		return nil, fmt.Errorf("schema ir: empty input")
	}
	zr, err := gzip.NewReader(bytes.NewReader(blob))
	if err != nil {
		return nil, fmt.Errorf("schema ir: not a gzip IR blob: %w", err)
	}
	defer zr.Close()
	raw, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("schema ir: gzip read: %w", err)
	}
	var env irEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("schema ir: parse: %w", err)
	}
	if env.Version != irVersion {
		return nil, fmt.Errorf("schema ir: unsupported version %d (runtime supports %d) — regenerate via make gen-yang", env.Version, irVersion)
	}
	ds := NewSchema()
	for _, im := range env.Modules {
		if im.Root == nil {
			return nil, fmt.Errorf("schema ir: module %s: missing root", im.Name)
		}
		rootNode, err := decodeNode(im.Root, nil)
		if err != nil {
			return nil, fmt.Errorf("schema ir: module %s: %w", im.Name, err)
		}
		root, ok := rootNode.(ContainerNode)
		if !ok {
			return nil, fmt.Errorf("schema ir: module %s: root is %s, want container", im.Name, im.Root.Kind)
		}
		m := NewModule(im.Name, im.Namespace, im.Revision, root).(*defaultModule)
		m.vendor = im.Vendor
		ds.AddModule(m)
	}
	return ds, nil
}

func decodeNode(in *irNode, parent Node) (Node, error) {
	if in == nil {
		return nil, fmt.Errorf("nil ir node")
	}
	switch in.Kind {
	case "container":
		c := NewContainer(in.Name, in.Description, in.Path, parent, in.Presence).(*defaultContainer)
		c.readOnly = in.ReadOnly
		c.whenExpr = in.When
		c.mustExprs = in.Must
		c.opExcludes = in.OpExcludes
		for _, ch := range in.Children {
			n, err := decodeNode(ch, c)
			if err != nil {
				return nil, err
			}
			c.AddChild(n)
		}
		return c, nil
	case "list":
		l := NewList(in.Name, in.Description, in.Path, parent, nil, in.UserOrdered).(*defaultList)
		l.readOnly = in.ReadOnly
		l.whenExpr = in.When
		l.mustExprs = in.Must
		l.opExcludes = in.OpExcludes
		for _, ch := range in.Children {
			n, err := decodeNode(ch, l)
			if err != nil {
				return nil, err
			}
			l.AddChild(n)
		}
		for _, keyName := range in.Keys {
			child, ok := l.Child(keyName)
			if !ok {
				return nil, fmt.Errorf("%s: key leaf %q not among children", in.Path, keyName)
			}
			leaf, ok := child.(LeafNode)
			if !ok {
				return nil, fmt.Errorf("%s: key %q is %T, want leaf", in.Path, keyName, child)
			}
			l.keys = append(l.keys, leaf)
		}
		return l, nil
	case "choice":
		ch := NewChoice(in.Name, in.Description, in.Path, parent).(*defaultChoice)
		ch.readOnly = in.ReadOnly
		ch.defaultCase = in.DefaultCase
		for _, cs := range in.Cases {
			n, err := decodeNode(cs, ch)
			if err != nil {
				return nil, err
			}
			cn, ok := n.(CaseNode)
			if !ok {
				return nil, fmt.Errorf("%s: choice member %s is %q, want case", in.Path, cs.Name, cs.Kind)
			}
			ch.AddCase(cn)
		}
		return ch, nil
	case "case":
		cs := NewCase(in.Name, in.Description, in.Path, parent).(*defaultCase)
		cs.readOnly = in.ReadOnly
		for _, chd := range in.Children {
			n, err := decodeNode(chd, cs)
			if err != nil {
				return nil, err
			}
			cs.AddChild(n)
		}
		return cs, nil
	case "leaf":
		lt, ok := leafTypeFromName[in.LeafType]
		if !ok {
			return nil, fmt.Errorf("%s: unknown leaf type %q", in.Path, in.LeafType)
		}
		leaf := NewLeaf(in.Name, in.Description, in.Path, parent, lt, in.IsKey, in.Mandatory).(*defaultLeaf)
		leaf.readOnly = in.ReadOnly
		leaf.whenExpr = in.When
		leaf.mustExprs = in.Must
		leaf.opExcludes = in.OpExcludes
		if len(in.Enums) > 0 {
			leaf.enumValues = in.Enums
		}
		leaf.units = in.Units
		leaf.pattern = in.Pattern
		leaf.leafList = in.LeafList
		leaf.supportFilter = in.SupportFilter
		leaf.dynamicDefault = in.DynamicDefault
		if in.Default != nil {
			leaf.defaultValue = *in.Default
		}
		if in.RangeMin != nil {
			leaf.rangeMin, leaf.hasMin = *in.RangeMin, true
		}
		if in.RangeMax != nil {
			leaf.rangeMax, leaf.hasMax = *in.RangeMax, true
		}
		if in.LengthMin != nil {
			leaf.lengthMin, leaf.hasLenMin = *in.LengthMin, true
		}
		if in.LengthMax != nil {
			leaf.lengthMax, leaf.hasLenMax = *in.LengthMax, true
		}
		return leaf, nil
	default:
		return nil, fmt.Errorf("%s: unknown node kind %q", in.Path, in.Kind)
	}
}
