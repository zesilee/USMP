// json_emit.go — 生成式 RFC7951 JSON 方法（阶段3 任务3.1，codegen-conventions §8）。
// 每 struct 生成 MarshalJSON/UnmarshalJSON，每 union 生成 marshal/unmarshal 包级
// helper；规则（模块限定键/64位字符串化/枚举值域名/[null]/list数组化）全部烘焙进
// 生成代码，运行期零 schema、零反射引擎（D2）。
package main

import (
	"fmt"
	"strings"
)

// jsonKey 计算字段的 RFC7951 JSON 键：跨模块边界（含 Device 顶层，其 Module
// 为空）带模块前缀。
func jsonKey(s *Struct, f *Field) string {
	if f.Module != s.Module {
		return f.Module + ":" + f.YangName
	}
	return f.YangName
}

// is64 reports 64 位整型（RFC7951 §6.1 字符串化）。
func is64(elem string) bool { return elem == "int64" || elem == "uint64" }

// emitJSONMethods renders MarshalJSON + UnmarshalJSON for one struct.
func emitJSONMethods(m *Model, s *Struct) string {
	var b strings.Builder
	emitMarshal(&b, m, s)
	emitUnmarshal(&b, m, s)
	return b.String()
}

func emitMarshal(b *strings.Builder, m *Model, s *Struct) {
	fmt.Fprintf(b, "// MarshalJSON implements RFC7951 encoding for %s.\n", s.Name)
	fmt.Fprintf(b, "func (t *%s) MarshalJSON() ([]byte, error) {\n", s.Name)
	b.WriteString("\tout := make(map[string]json.RawMessage)\n")
	for _, f := range s.Fields {
		key := jsonKey(s, f)
		g := "t." + f.GoName
		switch f.Kind {
		case KScalar:
			if is64(f.Elem) {
				fn, cast := "FormatUint", "uint64"
				if f.Elem == "int64" {
					fn, cast = "FormatInt", "int64"
				}
				fmt.Fprintf(b, "\tif %s != nil {\n\t\tout[%q] = json.RawMessage(strconv.Quote(strconv.%s(%s(*%s), 10)))\n\t}\n",
					g, key, fn, cast, g)
			} else {
				fmt.Fprintf(b, "\tif %s != nil {\n\t\tout[%q] = object.RawJSON(*%s)\n\t}\n", g, key, g)
			}
		case KEnum:
			fmt.Fprintf(b, "\tif %s != 0 {\n\t\tn, err := object.EnumName(%s)\n\t\tif err != nil {\n\t\t\treturn nil, fmt.Errorf(\"%s.%s: %%w\", err)\n\t\t}\n\t\tout[%q] = object.RawJSON(n)\n\t}\n",
				g, g, s.Name, f.GoName, key)
		case KEmpty:
			fmt.Fprintf(b, "\tif %s {\n\t\tout[%q] = object.EmptyJSON\n\t}\n", g, key)
		case KBinary:
			fmt.Fprintf(b, "\tif len(%s) > 0 {\n\t\tout[%q] = object.RawJSON([]byte(%s))\n\t}\n", g, key, g)
		case KUnsupported:
			fmt.Fprintf(b, "\tif %s != nil {\n\t\tbv, err := json.Marshal(%s)\n\t\tif err != nil {\n\t\t\treturn nil, fmt.Errorf(\"%s.%s: %%w\", err)\n\t\t}\n\t\tout[%q] = bv\n\t}\n",
				g, g, s.Name, f.GoName, key)
		case KContainer:
			fmt.Fprintf(b, "\tif %s != nil {\n\t\tbv, err := %s.MarshalJSON()\n\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n\t\tout[%q] = bv\n\t}\n", g, g, key)
		case KUnion:
			fmt.Fprintf(b, "\tif %s != nil {\n\t\tbv, err := marshal%s(%s)\n\t\tif err != nil {\n\t\t\treturn nil, fmt.Errorf(\"%s.%s: %%w\", err)\n\t\t}\n\t\tout[%q] = bv\n\t}\n",
				g, f.Elem, g, s.Name, f.GoName, key)
		case KList:
			emitMarshalList(b, s, f, key)
		case KLeafList:
			emitMarshalLeafList(b, s, f, key)
		}
	}
	b.WriteString("\treturn json.Marshal(out)\n}\n\n")
}

func emitMarshalList(b *strings.Builder, s *Struct, f *Field, key string) {
	g := "t." + f.GoName
	if f.Ordered {
		fmt.Fprintf(b, "\tif %s != nil && %s.Len() > 0 {\n\t\tparts := make([]json.RawMessage, 0, %s.Len())\n\t\tfor _, e := range %s.Values() {\n\t\t\tbv, err := e.MarshalJSON()\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\t\t\tparts = append(parts, bv)\n\t\t}\n\t\tout[%q] = object.JSONArray(parts)\n\t}\n",
			g, g, g, g, key)
		return
	}
	// map：按 key 排序保确定性（标量/枚举直接 <，bool 假前真后，复合键/union
	// 键按 fmt.Sprint）。
	less := "keys[i] < keys[j]"
	switch {
	case f.KeyType == "bool":
		less = "!keys[i] && keys[j]"
	case strings.HasSuffix(f.KeyType, "_Key") || strings.Contains(f.KeyType, "_Union"):
		less = "fmt.Sprint(keys[i]) < fmt.Sprint(keys[j])"
	}
	fmt.Fprintf(b, "\tif len(%s) > 0 {\n\t\tkeys := make([]%s, 0, len(%s))\n\t\tfor k := range %s {\n\t\t\tkeys = append(keys, k)\n\t\t}\n\t\tsort.Slice(keys, func(i, j int) bool { return %s })\n\t\tparts := make([]json.RawMessage, 0, len(keys))\n\t\tfor _, k := range keys {\n\t\t\tbv, err := %s[k].MarshalJSON()\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\t\t\tparts = append(parts, bv)\n\t\t}\n\t\tout[%q] = object.JSONArray(parts)\n\t}\n",
		g, f.KeyType, g, g, less, g, key)
}

func emitMarshalLeafList(b *strings.Builder, s *Struct, f *Field, key string) {
	g := "t." + f.GoName
	fmt.Fprintf(b, "\tif len(%s) > 0 {\n\t\tparts := make([]json.RawMessage, 0, len(%s))\n\t\tfor _, v := range %s {\n", g, g, g)
	switch {
	case f.ElemKind == KEnum:
		fmt.Fprintf(b, "\t\t\tn, err := object.EnumName(v)\n\t\t\tif err != nil {\n\t\t\t\treturn nil, fmt.Errorf(\"%s.%s: %%w\", err)\n\t\t\t}\n\t\t\tparts = append(parts, object.RawJSON(n))\n", s.Name, f.GoName)
	case f.ElemKind == KUnion:
		fmt.Fprintf(b, "\t\t\tbv, err := marshal%s(v)\n\t\t\tif err != nil {\n\t\t\t\treturn nil, fmt.Errorf(\"%s.%s: %%w\", err)\n\t\t\t}\n\t\t\tparts = append(parts, bv)\n", f.Elem, s.Name, f.GoName)
	case is64(f.Elem):
		u := "Uint"
		cast := "uint64"
		if f.Elem == "int64" {
			u, cast = "Int", "int64"
		}
		fmt.Fprintf(b, "\t\t\tparts = append(parts, json.RawMessage(strconv.Quote(strconv.Format%s(%s(v), 10))))\n", u, cast)
	default:
		b.WriteString("\t\t\tparts = append(parts, object.RawJSON(v))\n")
	}
	fmt.Fprintf(b, "\t\t}\n\t\tout[%q] = object.JSONArray(parts)\n\t}\n", key)
}

func emitUnmarshal(b *strings.Builder, m *Model, s *Struct) {
	fmt.Fprintf(b, "// UnmarshalJSON implements RFC7951 decoding for %s（未知键报错）。\n", s.Name)
	fmt.Fprintf(b, "func (t *%s) UnmarshalJSON(data []byte) error {\n", s.Name)
	b.WriteString("\tvar fields map[string]json.RawMessage\n\tif err := json.Unmarshal(data, &fields); err != nil {\n\t\treturn err\n\t}\n")
	if len(s.Fields) == 0 {
		// 零字段容器（空 presence 容器等）：任何键都是未知键。
		fmt.Fprintf(b, "\tfor k := range fields {\n\t\treturn fmt.Errorf(\"%s: unknown field %%q\", k)\n\t}\n\treturn nil\n}\n\n", s.Name)
		return
	}
	b.WriteString("\tfor k, raw := range fields {\n\t\tswitch object.StripModule(k) {\n")
	for _, f := range s.Fields {
		fmt.Fprintf(b, "\t\tcase %q:\n", f.YangName)
		emitUnmarshalField(b, m, s, f)
	}
	fmt.Fprintf(b, "\t\tdefault:\n\t\t\treturn fmt.Errorf(\"%s: unknown field %%q\", k)\n\t\t}\n\t}\n\treturn nil\n}\n\n", s.Name)
}

func emitUnmarshalField(b *strings.Builder, m *Model, s *Struct, f *Field) {
	g := "t." + f.GoName
	errCtx := s.Name + "." + f.YangName
	switch f.Kind {
	case KScalar:
		if is64(f.Elem) {
			p := "ParseUint64JSON"
			cast := ""
			if f.Elem == "int64" {
				p = "ParseInt64JSON"
			}
			fmt.Fprintf(b, "\t\t\tv, err := object.%s(raw)\n\t\t\tif err != nil {\n\t\t\t\treturn fmt.Errorf(\"%s: %%w\", err)\n\t\t\t}\n\t\t\t%s = &v\n", p, errCtx, g)
			_ = cast
		} else {
			fmt.Fprintf(b, "\t\t\tvar v %s\n\t\t\tif err := json.Unmarshal(raw, &v); err != nil {\n\t\t\t\treturn fmt.Errorf(\"%s: %%w\", err)\n\t\t\t}\n\t\t\t%s = &v\n", f.Elem, errCtx, g)
		}
	case KEnum:
		fmt.Fprintf(b, "\t\t\tvar n string\n\t\t\tif err := json.Unmarshal(raw, &n); err != nil {\n\t\t\t\treturn fmt.Errorf(\"%s: %%w\", err)\n\t\t\t}\n\t\t\tev, ok := object.EnumValueByName(EnumMaps, %q, n)\n\t\t\tif !ok {\n\t\t\t\treturn fmt.Errorf(\"%s: unknown enum value %%q\", n)\n\t\t\t}\n\t\t\t%s = %s(ev)\n",
			errCtx, f.Elem, errCtx, g, f.Elem)
	case KEmpty:
		fmt.Fprintf(b, "\t\t\tif !object.IsEmptyJSON(raw) {\n\t\t\t\treturn fmt.Errorf(\"%s: empty leaf expects [null], got %%s\", raw)\n\t\t\t}\n\t\t\t%s = true\n", errCtx, g)
	case KBinary:
		fmt.Fprintf(b, "\t\t\tvar v []byte\n\t\t\tif err := json.Unmarshal(raw, &v); err != nil {\n\t\t\t\treturn fmt.Errorf(\"%s: %%w\", err)\n\t\t\t}\n\t\t\t%s = object.Binary(v)\n", errCtx, g)
	case KUnsupported:
		fmt.Fprintf(b, "\t\t\tvar v interface{}\n\t\t\tif err := json.Unmarshal(raw, &v); err != nil {\n\t\t\t\treturn fmt.Errorf(\"%s: %%w\", err)\n\t\t\t}\n\t\t\t%s = v\n", errCtx, g)
	case KContainer:
		fmt.Fprintf(b, "\t\t\t%s = &%s{}\n\t\t\tif err := %s.UnmarshalJSON(raw); err != nil {\n\t\t\t\treturn err\n\t\t\t}\n", g, f.Elem, g)
	case KUnion:
		fmt.Fprintf(b, "\t\t\tuv, err := unmarshal%s(raw)\n\t\t\tif err != nil {\n\t\t\t\treturn fmt.Errorf(\"%s: %%w\", err)\n\t\t\t}\n\t\t\t%s = uv\n", f.Elem, errCtx, g)
	case KList:
		emitUnmarshalList(b, m, s, f)
	case KLeafList:
		emitUnmarshalLeafList(b, s, f)
	}
}

func emitUnmarshalList(b *strings.Builder, m *Model, s *Struct, f *Field) {
	g := "t." + f.GoName
	errCtx := s.Name + "." + f.YangName
	child := m.structIdx[f.Elem]
	fmt.Fprintf(b, "\t\t\tvar arr []json.RawMessage\n\t\t\tif err := json.Unmarshal(raw, &arr); err != nil {\n\t\t\t\treturn fmt.Errorf(\"%s: %%w\", err)\n\t\t\t}\n", errCtx)
	if f.Ordered {
		fmt.Fprintf(b, "\t\t\t%s = &%s_OrderedMap{}\n", g, f.Elem)
	} else {
		fmt.Fprintf(b, "\t\t\t%s = make(map[%s]*%s, len(arr))\n", g, f.KeyType, f.Elem)
	}
	fmt.Fprintf(b, "\t\t\tfor _, er := range arr {\n\t\t\t\te := &%s{}\n\t\t\t\tif err := e.UnmarshalJSON(er); err != nil {\n\t\t\t\t\treturn err\n\t\t\t\t}\n", f.Elem)
	// key 表达式（缺 key 叶报错）
	if child == nil || len(child.Keys) == 0 {
		fmt.Fprintf(b, "\t\t\t\t_ = e // 无 key list（当前闭包 0 处）\n\t\t\t}\n")
		return
	}
	for _, k := range child.Keys {
		if k.Ptr {
			fmt.Fprintf(b, "\t\t\t\tif e.%s == nil {\n\t\t\t\t\treturn fmt.Errorf(\"%s: entry missing key %s\")\n\t\t\t\t}\n", k.GoName, errCtx, k.YangName)
		}
	}
	keyExpr := ""
	if len(child.Keys) == 1 {
		k := child.Keys[0]
		if k.Ptr {
			keyExpr = "*e." + k.GoName
		} else {
			keyExpr = "e." + k.GoName
		}
	} else {
		parts := make([]string, 0, len(child.Keys))
		for _, k := range child.Keys {
			v := "e." + k.GoName
			if k.Ptr {
				v = "*e." + k.GoName
			}
			parts = append(parts, k.GoName+": "+v)
		}
		keyExpr = child.KeyName + "{" + strings.Join(parts, ", ") + "}"
	}
	if f.Ordered {
		fmt.Fprintf(b, "\t\t\t\tif err := %s.Append(%s, e); err != nil {\n\t\t\t\t\treturn fmt.Errorf(\"%s: %%w\", err)\n\t\t\t\t}\n\t\t\t}\n", g, keyExpr, errCtx)
	} else {
		fmt.Fprintf(b, "\t\t\t\t%s[%s] = e\n\t\t\t}\n", g, keyExpr)
	}
}

func emitUnmarshalLeafList(b *strings.Builder, s *Struct, f *Field) {
	g := "t." + f.GoName
	errCtx := s.Name + "." + f.YangName
	switch {
	case f.ElemKind == KEnum:
		fmt.Fprintf(b, "\t\t\tvar names []string\n\t\t\tif err := json.Unmarshal(raw, &names); err != nil {\n\t\t\t\treturn fmt.Errorf(\"%s: %%w\", err)\n\t\t\t}\n\t\t\t%s = make([]%s, 0, len(names))\n\t\t\tfor _, n := range names {\n\t\t\t\tev, ok := object.EnumValueByName(EnumMaps, %q, n)\n\t\t\t\tif !ok {\n\t\t\t\t\treturn fmt.Errorf(\"%s: unknown enum value %%q\", n)\n\t\t\t\t}\n\t\t\t\t%s = append(%s, %s(ev))\n\t\t\t}\n",
			errCtx, g, f.Elem, f.Elem, errCtx, g, g, f.Elem)
	case f.ElemKind == KUnion:
		fmt.Fprintf(b, "\t\t\tvar elems []json.RawMessage\n\t\t\tif err := json.Unmarshal(raw, &elems); err != nil {\n\t\t\t\treturn fmt.Errorf(\"%s: %%w\", err)\n\t\t\t}\n\t\t\t%s = make([]%s, 0, len(elems))\n\t\t\tfor _, er := range elems {\n\t\t\t\tuv, err := unmarshal%s(er)\n\t\t\t\tif err != nil {\n\t\t\t\t\treturn fmt.Errorf(\"%s: %%w\", err)\n\t\t\t\t}\n\t\t\t\t%s = append(%s, uv)\n\t\t\t}\n",
			errCtx, g, f.Elem, f.Elem, errCtx, g, g)
	case is64(f.Elem):
		p := "ParseUint64JSON"
		if f.Elem == "int64" {
			p = "ParseInt64JSON"
		}
		fmt.Fprintf(b, "\t\t\tvar elems []json.RawMessage\n\t\t\tif err := json.Unmarshal(raw, &elems); err != nil {\n\t\t\t\treturn fmt.Errorf(\"%s: %%w\", err)\n\t\t\t}\n\t\t\t%s = make([]%s, 0, len(elems))\n\t\t\tfor _, er := range elems {\n\t\t\t\tv, err := object.%s(er)\n\t\t\t\tif err != nil {\n\t\t\t\t\treturn fmt.Errorf(\"%s: %%w\", err)\n\t\t\t\t}\n\t\t\t\t%s = append(%s, v)\n\t\t\t}\n",
			errCtx, g, f.Elem, p, errCtx, g, g)
	default:
		fmt.Fprintf(b, "\t\t\tvar v []%s\n\t\t\tif err := json.Unmarshal(raw, &v); err != nil {\n\t\t\t\treturn fmt.Errorf(\"%s: %%w\", err)\n\t\t\t}\n\t\t\t%s = v\n", f.Elem, errCtx, g)
	}
}

// emitUnionJSONHelpers renders marshal<U>/unmarshal<U> for one union.
func emitUnionJSONHelpers(u *Union) string {
	var b strings.Builder
	fmt.Fprintf(&b, "// marshal%s encodes a %s union value（RFC7951）。\n", u.Name, u.Name)
	fmt.Fprintf(&b, "func marshal%s(u %s) (json.RawMessage, error) {\n\tswitch v := u.(type) {\n", u.Name, u.Name)
	for _, mem := range u.Members {
		wrap := u.Name + "_" + unionMemberSuffix(mem)
		field := unionMemberField(mem)
		switch {
		case strings.HasPrefix(mem, "E_"):
			fmt.Fprintf(&b, "\tcase *%s:\n\t\tn, err := object.EnumName(v.%s)\n\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n\t\treturn object.RawJSON(n), nil\n", wrap, field)
		case is64(mem):
			fn, cast := "FormatUint", "uint64"
			if mem == "int64" {
				fn, cast = "FormatInt", "int64"
			}
			fmt.Fprintf(&b, "\tcase *%s:\n\t\treturn json.RawMessage(strconv.Quote(strconv.%s(%s(v.%s), 10))), nil\n", wrap, fn, cast, field)
		default:
			fmt.Fprintf(&b, "\tcase *%s:\n\t\treturn object.RawJSON(v.%s), nil\n", wrap, field)
		}
	}
	fmt.Fprintf(&b, "\tdefault:\n\t\treturn nil, fmt.Errorf(\"unsupported %s value %%T\", u)\n\t}\n}\n\n", u.Name)

	fmt.Fprintf(&b, "// unmarshal%s decodes a %s union value（按成员声明序试探）。\n", u.Name, u.Name)
	fmt.Fprintf(&b, "func unmarshal%s(raw json.RawMessage) (%s, error) {\n", u.Name, u.Name)
	for _, mem := range u.Members {
		wrap := u.Name + "_" + unionMemberSuffix(mem)
		field := unionMemberField(mem)
		switch {
		case strings.HasPrefix(mem, "E_"):
			fmt.Fprintf(&b, "\t{\n\t\tvar n string\n\t\tif json.Unmarshal(raw, &n) == nil {\n\t\t\tif ev, ok := object.EnumValueByName(EnumMaps, %q, n); ok {\n\t\t\t\treturn &%s{%s: %s(ev)}, nil\n\t\t\t}\n\t\t}\n\t}\n", mem, wrap, field, mem)
		case is64(mem):
			p := "ParseUint64JSON"
			if mem == "int64" {
				p = "ParseInt64JSON"
			}
			fmt.Fprintf(&b, "\t{\n\t\tif v, err := object.%s(raw); err == nil {\n\t\t\treturn &%s{%s: v}, nil\n\t\t}\n\t}\n", p, wrap, field)
		case mem == "string":
			fmt.Fprintf(&b, "\t{\n\t\tvar v string\n\t\tif json.Unmarshal(raw, &v) == nil {\n\t\t\treturn &%s{%s: v}, nil\n\t\t}\n\t}\n", wrap, field)
		default: // 数值类
			fmt.Fprintf(&b, "\t{\n\t\tvar v %s\n\t\tif json.Unmarshal(raw, &v) == nil {\n\t\t\treturn &%s{%s: v}, nil\n\t\t}\n\t}\n", mem, wrap, field)
		}
	}
	fmt.Fprintf(&b, "\treturn nil, fmt.Errorf(\"no %s member matches %%s\", raw)\n}\n\n", u.Name)
	return b.String()
}
