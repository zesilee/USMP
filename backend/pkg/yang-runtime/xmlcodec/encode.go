package xmlcodec

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
)

// Spec is the per-module codec data a driver descriptor carries (DR-01): the
// engine is generic, everything device-specific is data. Namespace is explicit
// because the embedded gzip SchemaTree does not resolve Entry.Namespace()
// (实测为空，见 design D3b).
type Spec struct {
	// Namespace is the module's XML namespace, declared on the root element.
	Namespace string
	// Schema returns the yang.Entry of the module's root container — either a
	// list container (e.g. "vlans", whose Dir carries the keyed list child) or
	// a plain container (e.g. "bgp", holding only scalars/sub-containers). Its
	// Name is the root element name; the engine picks list- or container-mode
	// by whether the GoStruct has a root YANG-list map field.
	Schema func() *yang.Entry
	// Namespaces optionally maps module name (the generated struct field's
	// `module:"…"` tag) to its XML namespace URI, enabling per-node namespace
	// for augment 跨模块 trees (XC-06): a node whose module resolves to a
	// namespace different from its parent's effective namespace gets an explicit
	// xmlns. Nil/empty preserves the single-root-namespace behavior byte-for-byte
	// (single-module trees never resolve a differing namespace → no new xmlns).
	Namespaces map[string]string
}

// nsResolver decides per-node namespace emission from module tags (XC-06).
// A nil/empty resolver always returns "" → no per-node xmlns (byte-identical
// legacy behavior for single-module trees).
type nsResolver map[string]string

// at returns the namespace to declare on a child element whose module tag is
// `mod`, given the parent's effective namespace `parentNS` — or "" when no new
// xmlns is needed (unknown module, unmapped, or same as parent).
func (r nsResolver) at(mod, parentNS string) string {
	if len(r) == 0 || mod == "" {
		return ""
	}
	n, ok := r[mod]
	if !ok || n == "" || n == parentNS {
		return ""
	}
	return n
}

// moduleTag returns the generated struct field's `module:"…"` tag.
func moduleTag(f reflect.StructField) string { return f.Tag.Get("module") }

// NetconfBaseNS carries the edit-config `operation` attribute (RFC 6241 §7.2).
const NetconfBaseNS = "urn:ietf:params:xml:ns:netconf:base:1.0"

var goEnumType = reflect.TypeOf((*ygot.GoEnum)(nil)).Elem()

// resolved carries the per-call view of a Spec after validation.
type resolved struct {
	ns     string
	root   string
	schema *yang.Entry // container entry; may carry list child in Dir
	list   *yang.Entry // list child entry (nil if schema lacks it)
}

func (s *Spec) resolve(listName string) (*resolved, error) {
	if s == nil || s.Schema == nil {
		return nil, fmt.Errorf("xmlcodec: nil spec or schema")
	}
	e := s.Schema()
	if e == nil {
		return nil, fmt.Errorf("xmlcodec: spec schema entry is nil")
	}
	if s.Namespace == "" {
		return nil, fmt.Errorf("xmlcodec: spec namespace is empty for %s", e.Name)
	}
	r := &resolved{ns: s.Namespace, root: e.Name, schema: e}
	if c, ok := e.Dir[listName]; ok {
		r.list = c
	}
	return r, nil
}

// wrappers returns the list container's ancestor container element names,
// outermost-first (e.g. ["ifm"] for interfaces, ["vlan"] for vlans), excluding
// the synthetic fake root. The edit-config payload must nest the list container
// inside these so it matches the device's YANG data tree（真机与模拟器种子
// DemoSeedConfig 均把 list 容器嵌套在模块顶层容器下）——扁平根会在设备树里匹配不到
// 存量条目，正是「内置接口删不掉」的根因。列表容器若直接挂在 fake root 下则返回空
// （无模块容器可包，退回扁平根 + 自带 xmlns，R08 降级）。
func (r *resolved) wrappers() []string {
	var names []string
	for p := r.schema.Parent; p != nil && p.Parent != nil; p = p.Parent {
		names = append(names, p.Name)
	}
	for i, j := 0, len(names)-1; i < j; i, j = i+1, j-1 {
		names[i], names[j] = names[j], names[i]
	}
	return names
}

// openWrappers writes the ancestor container open tags; xmlns is declared on the
// outermost wrapper only (inner containers inherit it, matching the seed shape
// <ifm xmlns=NS><interfaces>). Returns true if any wrapper was written, so the
// list container omits its own redundant xmlns.
func openWrappers(b *strings.Builder, r *resolved) bool {
	w := r.wrappers()
	for i, name := range w {
		if i == 0 {
			fmt.Fprintf(b, "<%s xmlns=%q>", name, r.ns)
		} else {
			fmt.Fprintf(b, "<%s>", name)
		}
	}
	return len(w) > 0
}

// closeWrappers writes the ancestor container close tags in innermost-first order.
func closeWrappers(b *strings.Builder, r *resolved) {
	w := r.wrappers()
	for i := len(w) - 1; i >= 0; i-- {
		fmt.Fprintf(b, "</%s>", w[i])
	}
}

// keyNames returns the list key leaf names in YANG order, or nil if unknown.
func (r *resolved) keyNames() []string {
	if r.list == nil || r.list.Key == "" {
		return nil
	}
	return strings.Fields(r.list.Key)
}

// findContainerMap locates the container's unique YANG-list map field.
// found is false when the container has no map field at all — a plain-container
// root (e.g. /bgp:bgp holds only scalars and sub-containers, no root list),
// which Encode/Decode serve via container-mode. err is returned only when the
// container has multiple map fields (malformed).
func findContainerMap(cv reflect.Value) (mapVal reflect.Value, elemTag string, found bool, err error) {
	t := cv.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := pathTag(f)
		if tag == "" || f.Type.Kind() != reflect.Map {
			continue
		}
		if found {
			return reflect.Value{}, "", false, fmt.Errorf("xmlcodec: container %s has multiple list map fields", t.Name())
		}
		mapVal, elemTag, found = cv.Field(i), tag, true
	}
	return mapVal, elemTag, found, nil
}

// containerMap requires a YANG-list map field; callers on the list-only paths
// (delete, wrap) use it. Container-rooted modules use findContainerMap instead.
func containerMap(cv reflect.Value) (reflect.Value, string, error) {
	mapVal, elemTag, found, err := findContainerMap(cv)
	if err != nil {
		return reflect.Value{}, "", err
	}
	if !found {
		return reflect.Value{}, "", fmt.Errorf("xmlcodec: container %s has no list map field", cv.Type().Name())
	}
	return mapVal, elemTag, nil
}

func pathTag(f reflect.StructField) string {
	tag := f.Tag.Get("path")
	if i := strings.Index(tag, "|"); i >= 0 {
		tag = tag[:i]
	}
	if i := strings.LastIndex(tag, "/"); i >= 0 {
		tag = tag[i+1:]
	}
	return tag
}

// Encode serializes a ygot list-container GoStruct to NETCONF edit-config XML
// (XC-01). Skip semantics mirror the legacy hand-written builders exactly:
// nil pointer leaves, zero-valued enums (UNSET) and nil nested containers are
// omitted; an empty list yields a self-closing namespaced root. Fields are
// emitted key leaves first (DP-07 惯例), then in struct declaration order. No
// config-false filtering: populated means pushed (design D3b, 实测校准).
func Encode(spec *Spec, v ygot.GoStruct) (string, error) {
	cv, err := derefContainer(v)
	if err != nil {
		return "", err
	}
	mapVal, elemTag, found, err := findContainerMap(cv)
	if err != nil {
		return "", err
	}
	if !found {
		// Plain-container root (no root YANG list, e.g. /bgp:bgp): emit the
		// container element + its fields via the shared field machinery.
		return encodeContainer(spec, cv)
	}
	r, err := spec.resolve(elemTag)
	if err != nil {
		return "", err
	}
	res := nsResolver(spec.Namespaces)
	var b strings.Builder
	wrapped := openWrappers(&b, r)
	if mapVal.IsNil() || mapVal.Len() == 0 {
		if wrapped {
			fmt.Fprintf(&b, "<%s/>", r.root)
		} else {
			fmt.Fprintf(&b, "<%s xmlns=%q/>", r.root, r.ns)
		}
		closeWrappers(&b, r)
		return b.String(), nil
	}
	if wrapped {
		fmt.Fprintf(&b, "<%s>", r.root)
	} else {
		fmt.Fprintf(&b, "<%s xmlns=%q>", r.root, r.ns)
	}
	// Root list entries share the module of the root container; entryMod "" ⇒
	// no per-node xmlns at a list root (single-module, byte-identical).
	if err := encodeList(&b, r, mapVal, elemTag, nil, res, r.ns, ""); err != nil {
		return "", err
	}
	fmt.Fprintf(&b, "</%s>", r.root)
	closeWrappers(&b, r)
	return b.String(), nil
}

// encodeContainer serializes a plain-container root GoStruct (no YANG list at
// the root, e.g. /bgp:bgp) to edit-config XML: <root xmlns=NS>{fields}</root>,
// or a self-closing <root xmlns=NS/> when every field is empty. It reuses
// encodeFields — the same leaf / nested-container / nested-list machinery that
// serves list entries — so scalars, sub-containers and deeper nested lists all
// encode identically to the list path (no separate leaf logic to drift).
func encodeContainer(spec *Spec, cv reflect.Value) (string, error) {
	r, err := spec.resolve("") // no list child; r.root = container element name
	if err != nil {
		return "", err
	}
	var body strings.Builder
	if err := encodeFields(&body, cv, r.schema, nil, nsResolver(spec.Namespaces), r.ns); err != nil {
		return "", err
	}
	var b strings.Builder
	wrapped := openWrappers(&b, r)
	empty := body.Len() == 0
	switch {
	case empty && wrapped:
		fmt.Fprintf(&b, "<%s/>", r.root)
	case empty:
		fmt.Fprintf(&b, "<%s xmlns=%q/>", r.root, r.ns)
	case wrapped:
		fmt.Fprintf(&b, "<%s>%s</%s>", r.root, body.String(), r.root)
	default:
		fmt.Fprintf(&b, "<%s xmlns=%q>%s</%s>", r.root, r.ns, body.String(), r.root)
	}
	closeWrappers(&b, r)
	return b.String(), nil
}

// encodeList emits every entry of a YANG list map, key leaves first. schema
// may be nil (schema-less lists fall back to ΛListKeyMap for key names).
// entryMod is the list field's module tag; when it resolves (via res) to a
// namespace differing from parentNS, each entry element declares that xmlns
// and its children inherit it (XC-06). entryMod "" ⇒ no per-node xmlns.
func encodeList(b *strings.Builder, r *resolved, mapVal reflect.Value, elemTag string, schema *yang.Entry, res nsResolver, parentNS, entryMod string) error {
	if schema == nil {
		schema = r.list
	}
	entryNS := res.at(entryMod, parentNS)
	effNS := parentNS
	if entryNS != "" {
		effNS = entryNS
	}
	for _, mk := range sortedKeys(mapVal) {
		ev := mapVal.MapIndex(mk)
		if ev.Kind() == reflect.Ptr && ev.IsNil() {
			continue
		}
		openTag(b, elemTag, entryNS)
		emitted, err := encodeKeysFirst(b, ev, mk, schema, res, effNS)
		if err != nil {
			return fmt.Errorf("list %s: %w", elemTag, err)
		}
		if err := encodeFields(b, ev.Elem(), schema, emitted, res, effNS); err != nil {
			return fmt.Errorf("list %s: %w", elemTag, err)
		}
		b.WriteString("</" + elemTag + ">")
	}
	return nil
}

// encodeKeysFirst writes the entry's key leaves before any other field,
// falling back to the map key value when the key leaf is nil（legacy 语义）.
// Key names come from the schema Key statement, else from ΛListKeyMap.
func encodeKeysFirst(b *strings.Builder, ev reflect.Value, mapKey reflect.Value, schema *yang.Entry, res nsResolver, parentNS string) (map[string]bool, error) {
	var names []string
	if schema != nil && schema.Key != "" {
		names = strings.Fields(schema.Key)
	} else if kh, ok := ev.Interface().(ygot.KeyHelperGoStruct); ok {
		if km, err := kh.ΛListKeyMap(); err == nil {
			for n := range km {
				names = append(names, n)
			}
			sort.Strings(names)
		}
	}
	emitted := map[string]bool{}
	sv := ev.Elem()
	for _, kn := range names {
		emitted[kn] = true
		f, sf, ok := fieldByTag(sv, kn)
		keyNS := ""
		if ok {
			keyNS = res.at(moduleTag(sf), parentNS)
		}
		if ok && f.Kind() == reflect.Ptr && !f.IsNil() {
			if err := encodeLeaf(b, kn, f.Elem(), keyNS); err != nil {
				return nil, err
			}
			continue
		}
		// Key leaf absent on the entry: use the map key（legacy fallback）.
		if err := encodeLeaf(b, kn, mapKey, keyNS); err != nil {
			return nil, err
		}
	}
	return emitted, nil
}

// encodeFields serializes the remaining struct fields in declaration order.
// parentNS is the enclosing element's effective namespace; each field may open
// a module boundary (XC-06) resolved from its `module` tag.
func encodeFields(b *strings.Builder, sv reflect.Value, schema *yang.Entry, skip map[string]bool, res nsResolver, parentNS string) error {
	t := sv.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := pathTag(f)
		if tag == "" || (skip != nil && skip[tag]) {
			continue
		}
		fv := sv.Field(i)
		var child *yang.Entry
		if schema != nil {
			child = schema.Dir[tag]
		}
		if err := encodeField(b, tag, fv, child, res, parentNS, moduleTag(f)); err != nil {
			return fmt.Errorf("field %s: %w", tag, err)
		}
	}
	return nil
}

func encodeField(b *strings.Builder, tag string, fv reflect.Value, schema *yang.Entry, res nsResolver, parentNS, mod string) error {
	ns := res.at(mod, parentNS) // module boundary xmlns ("" ⇒ inherit parent)
	effNS := parentNS
	if ns != "" {
		effNS = ns
	}
	if fv.Type().Implements(goEnumType) && fv.Kind() == reflect.Int64 {
		if fv.Int() == 0 { // UNSET
			return nil
		}
		if ns == "" {
			fmt.Fprintf(b, "<%s>%d</%s>", tag, fv.Int(), tag)
		} else {
			fmt.Fprintf(b, "<%s xmlns=%q>%d</%s>", tag, ns, fv.Int(), tag)
		}
		return nil
	}
	switch fv.Kind() {
	case reflect.Ptr:
		if fv.IsNil() {
			return nil
		}
		if fv.Elem().Kind() == reflect.Struct {
			openTag(b, tag, ns)
			if err := encodeFields(b, fv.Elem(), schema, nil, res, effNS); err != nil {
				return err
			}
			b.WriteString("</" + tag + ">")
			return nil
		}
		return encodeLeaf(b, tag, fv.Elem(), ns)
	case reflect.Map: // nested YANG list
		if fv.IsNil() || fv.Len() == 0 {
			return nil
		}
		r := &resolved{list: schema}
		return encodeList(b, r, fv, tag, schema, res, parentNS, mod)
	case reflect.Slice: // leaf-list of scalars
		if fv.Type().Elem().Kind() == reflect.Uint8 {
			return fmt.Errorf("binary leaf unsupported")
		}
		for i := 0; i < fv.Len(); i++ {
			if err := encodeLeaf(b, tag, fv.Index(i), ns); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported field form %s", fv.Kind())
	}
}

// openTag writes a child element's open tag, declaring xmlns only when ns != ""
// (a module boundary, XC-06). ns == "" reproduces the legacy plain `<tag>`.
func openTag(b *strings.Builder, tag, ns string) {
	if ns == "" {
		b.WriteString("<" + tag + ">")
	} else {
		fmt.Fprintf(b, "<%s xmlns=%q>", tag, ns)
	}
}

// encodeLeaf writes a scalar leaf. ns != "" declares a per-node xmlns on the leaf
// (XC-06 module boundary for a lone augment leaf); ns == "" is the legacy form.
func encodeLeaf(b *strings.Builder, tag string, v reflect.Value, ns string) error {
	openEl, closeEl := "<"+tag+">", "</"+tag+">"
	if ns != "" {
		openEl = fmt.Sprintf("<%s xmlns=%q>", tag, ns)
	}
	switch v.Kind() {
	case reflect.String:
		b.WriteString(openEl + escape(v.String()) + closeEl)
	case reflect.Bool:
		fmt.Fprintf(b, "%s%t%s", openEl, v.Bool(), closeEl)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fmt.Fprintf(b, "%s%d%s", openEl, v.Int(), closeEl)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		fmt.Fprintf(b, "%s%d%s", openEl, v.Uint(), closeEl)
	case reflect.Float64, reflect.Float32:
		fmt.Fprintf(b, "%s%g%s", openEl, v.Float(), closeEl)
	default:
		return fmt.Errorf("unsupported leaf kind %s", v.Kind())
	}
	return nil
}

func derefContainer(v ygot.GoStruct) (reflect.Value, error) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || rv.Kind() != reflect.Ptr || rv.IsNil() {
		return reflect.Value{}, fmt.Errorf("xmlcodec: container must be a non-nil struct pointer, got %T", v)
	}
	if rv.Elem().Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("xmlcodec: container must point to a struct, got %T", v)
	}
	return rv.Elem(), nil
}

// sortedKeys returns map keys in deterministic order (调试可读；等价性由
// 规范化比较保证，与序无关).
func sortedKeys(m reflect.Value) []reflect.Value {
	keys := m.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		switch a.Kind() {
		case reflect.String:
			return a.String() < b.String()
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return a.Uint() < b.Uint()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return a.Int() < b.Int()
		default:
			return fmt.Sprint(a.Interface()) < fmt.Sprint(b.Interface())
		}
	})
	return keys
}

// escape mirrors the legacy xmlEscape (escapes the five XML special chars).
func escape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '&':
			b.WriteString("&amp;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&apos;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// fieldByTag finds a struct field by its YANG path tag, returning its value and
// the StructField (for the `module` tag, XC-06).
func fieldByTag(sv reflect.Value, tag string) (reflect.Value, reflect.StructField, bool) {
	t := sv.Type()
	for i := 0; i < t.NumField(); i++ {
		if pathTag(t.Field(i)) == tag {
			return sv.Field(i), t.Field(i), true
		}
	}
	return reflect.Value{}, reflect.StructField{}, false
}
