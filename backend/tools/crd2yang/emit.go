// emit.go —— crd2yang 的 YANG 渲染半边：OpenAPI schema → YANG 语句（C2Y-03）。
// 映射表与 tools/crdgen mapEntry/mapScalar 逐条互逆，由往返测试闭环（C2Y-04）。
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// emitObject renders an object schema as a container: guarded, children
// sorted for determinism (mirrors crdgen sortedChildNames); required
// properties become mandatory leaves.
func emitObject(w *writer, name string, s *apiextv1.JSONSchemaProps, path string) error {
	if err := guard(s, path); err != nil {
		return err
	}
	if s.Type != "object" {
		return fmt.Errorf("%s: expected type object, got %q", path, s.Type)
	}
	w.line("container %s {", name)
	w.in()
	if s.Description != "" {
		w.line("description")
		w.line("  %s;", quote(s.Description))
	}
	required := map[string]bool{}
	for _, r := range s.Required {
		if _, ok := s.Properties[r]; !ok {
			return fmt.Errorf("%s: required property %q not found in properties — the contract would silently shrink", path, r)
		}
		required[r] = true
	}
	names := make([]string, 0, len(s.Properties))
	for n := range s.Properties {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		child := s.Properties[n]
		if err := emitNode(w, n, &child, path+"."+n, required[n]); err != nil {
			return err
		}
	}
	w.out()
	w.line("}")
	return nil
}

// emitNode dispatches one property to container/list/leaf-list/leaf.
// CRD 属性名是任意 JSON key，先过 YANG 标识符校验（含 RFC7950 xml 前缀禁令），
// 否则会静默产出非法 YANG。
func emitNode(w *writer, name string, s *apiextv1.JSONSchemaProps, path string, requiredHere bool) error {
	if err := validIdentifier(name, path); err != nil {
		return err
	}
	if err := guard(s, path); err != nil {
		return err
	}
	switch s.Type {
	case "object":
		if requiredHere {
			return fmt.Errorf("%s: required on an object property is not representable in the crdgen-inverse mapping (only leaves can be mandatory)", path)
		}
		return emitObject(w, name, s, path)
	case "array":
		if requiredHere {
			return fmt.Errorf("%s: required on an array property is not representable in the crdgen-inverse mapping (only leaves can be mandatory)", path)
		}
		return emitArray(w, name, s, path)
	case "string", "integer", "boolean":
		return emitLeaf(w, "leaf", name, s, path, requiredHere)
	case "":
		return fmt.Errorf("%s: missing type", path)
	default:
		return fmt.Errorf("%s: OpenAPI type %q is not mappable to YANG (C2Y-03)", path, s.Type)
	}
}

// emitArray maps object arrays to YANG lists (key = x-kubernetes-list-map-keys,
// mandatory) and scalar arrays to leaf-lists.
func emitArray(w *writer, name string, s *apiextv1.JSONSchemaProps, path string) error {
	if s.Items == nil || s.Items.Schema == nil {
		return fmt.Errorf("%s: array without items schema", path)
	}
	item := s.Items.Schema
	if s.XListType != nil && *s.XListType != "map" {
		return fmt.Errorf("%s: x-kubernetes-list-type %q is not mappable (only \"map\" object lists are supported, C2Y-03)", path, *s.XListType)
	}
	if item.Type == "object" {
		if len(s.XListMapKeys) == 0 {
			return fmt.Errorf("%s: object array without x-kubernetes-list-map-keys — YANG config lists require a key (C2Y-03)", path)
		}
		return emitList(w, name, s, item, path)
	}
	if len(s.XListMapKeys) > 0 || s.XListType != nil {
		return fmt.Errorf("%s: x-kubernetes-list-map-keys/list-type on a scalar array is not mappable", path)
	}
	// description 挂在数组属性上（items 无描述），下沉到 leaf-list 供前端渲染标签。
	leaf := *item
	if leaf.Description == "" {
		leaf.Description = s.Description
	}
	return emitLeaf(w, "leaf-list", name, &leaf, path, false)
}

func emitList(w *writer, name string, arr *apiextv1.JSONSchemaProps, item *apiextv1.JSONSchemaProps, path string) error {
	if err := guard(item, path); err != nil {
		return err
	}
	keys := append([]string{}, arr.XListMapKeys...)
	for _, k := range keys {
		if _, ok := item.Properties[k]; !ok {
			return fmt.Errorf("%s: list key %q not found in item properties", path, k)
		}
	}
	// key 属性从 mandatory 剥离：crdgen 会把 key 并回 required（C2Y-03/C2Y-04）。
	keySet := map[string]bool{}
	for _, k := range keys {
		keySet[k] = true
	}

	w.line("list %s {", name)
	w.in()
	w.line("key %s;", quote(strings.Join(keys, " ")))
	if arr.Description != "" {
		w.line("description")
		w.line("  %s;", quote(arr.Description))
	}
	required := map[string]bool{}
	for _, r := range item.Required {
		if _, ok := item.Properties[r]; !ok {
			return fmt.Errorf("%s: required property %q not found in item properties — the contract would silently shrink", path, r)
		}
		if !keySet[r] {
			required[r] = true
		}
	}
	names := make([]string, 0, len(item.Properties))
	for n := range item.Properties {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		child := item.Properties[n]
		if err := emitNode(w, n, &child, path+"."+n, required[n]); err != nil {
			return err
		}
	}
	w.out()
	w.line("}")
	return nil
}

// emitLeaf renders a scalar as leaf/leaf-list: type → mandatory → description
// (statement order mirrors the hand-written intent models).
func emitLeaf(w *writer, kw, name string, s *apiextv1.JSONSchemaProps, path string, mandatory bool) error {
	if err := guard(s, path); err != nil {
		return err
	}
	w.line("%s %s {", kw, name)
	w.in()
	if err := emitScalarType(w, s, path); err != nil {
		return err
	}
	if mandatory {
		w.line("mandatory true;")
	}
	if s.Description != "" {
		w.line("description")
		w.line("  %s;", quote(s.Description))
	}
	w.out()
	w.line("}")
	return nil
}

// emitScalarType is the inverse of crdgen mapScalar: every branch there has
// exactly one branch here (C2Y-03).
func emitScalarType(w *writer, s *apiextv1.JSONSchemaProps, path string) error {
	switch s.Type {
	case "boolean":
		// enum/allOf/min/max 等跨类型约束已在 guard 按类型白名单拦截。
		w.line("type boolean;")
		return nil
	case "string":
		return emitStringType(w, s, path)
	case "integer":
		return emitIntegerType(w, s, path)
	default:
		return fmt.Errorf("%s: OpenAPI type %q is not mappable to YANG (C2Y-03)", path, s.Type)
	}
}

func emitStringType(w *writer, s *apiextv1.JSONSchemaProps, path string) error {
	if len(s.Enum) > 0 {
		if s.Pattern != "" || len(s.AllOf) > 0 {
			return fmt.Errorf("%s: enum combined with pattern is not mappable", path)
		}
		w.line("type enumeration {")
		w.in()
		for _, e := range s.Enum {
			var v string
			if err := json.Unmarshal(e.Raw, &v); err != nil {
				return fmt.Errorf("%s: enum value %s is not a string", path, e.Raw)
			}
			// 空串/首尾空白的 enum 值是非法 YANG，goyang 必解析失败，生成前拦截。
			if v == "" || strings.TrimSpace(v) != v {
				return fmt.Errorf("%s: enum value %q is not a legal YANG enum (empty or leading/trailing whitespace)", path, v)
			}
			w.line("enum %s;", quote(v))
		}
		w.out()
		w.line("}")
		return nil
	}

	// 多 pattern：crdgen 用 allOf（每项仅 pattern）承载，反向逐条还原。
	var patterns []string
	if s.Pattern != "" {
		patterns = append(patterns, s.Pattern)
	}
	for i, sub := range s.AllOf {
		if sub.Pattern == "" || !patternOnly(&sub) {
			return fmt.Errorf("%s: allOf[%d] carries more than a pattern — only crdgen-style multi-pattern allOf is mappable", path, i)
		}
		patterns = append(patterns, sub.Pattern)
	}
	if len(patterns) == 0 {
		w.line("type string;")
		return nil
	}
	w.line("type string {")
	w.in()
	for _, p := range patterns {
		w.line("pattern %s;", quote(p))
	}
	w.out()
	w.line("}")
	return nil
}

// fullRanges 把 YANG 整型基类型映射到其全量取值范围的 float64 精确表示。
// goyang 对**无** range 语句的整型也会附带基类型全量 Range（实测 v1.6.0），
// 因此 crdgen 会对「无界」整型输出 ±全量边界的 minimum/maximum；反向把这类
// 边界规范化回裸类型（不带 range 语句），闭合 C2Y-04 往返。
// 注：int64/uint64 的边界超出 float64 精确整数域，常量转换后分别为 ±2^63 与
// 2^64——与 JSON/YAML 解析同一路径得到的值一致，比较仍是精确的。
var fullRanges = []struct {
	typ      string
	min, max float64
}{
	{"int8", -128, 127},
	{"int16", -32768, 32767},
	{"int32", -2147483648, 2147483647},
	{"int64", -9223372036854775808, 9223372036854775807},
	{"uint8", 0, 255},
	{"uint16", 0, 65535},
	{"uint32", 0, 4294967295},
	{"uint64", 0, 18446744073709551615},
}

func emitIntegerType(w *writer, s *apiextv1.JSONSchemaProps, path string) error {
	switch {
	case s.Minimum == nil && s.Maximum == nil:
		w.line("type int64;")
		return nil
	case s.Minimum == nil || s.Maximum == nil:
		return fmt.Errorf("%s: integer with only one bound — crdgen always emits both, supply minimum and maximum", path)
	}
	min, max := *s.Minimum, *s.Maximum
	if min != math.Trunc(min) || max != math.Trunc(max) {
		return fmt.Errorf("%s: non-integral bound (%v..%v)", path, min, max)
	}
	if min > max {
		return fmt.Errorf("%s: minimum %v exceeds maximum %v", path, min, max)
	}
	// 全量基类型边界 → 裸类型（见 fullRanges 注释）。
	for _, fr := range fullRanges {
		if min == fr.min && max == fr.max {
			w.line("type %s;", fr.typ)
			return nil
		}
	}
	// float64 只能精确表示 ±2^53 内的整数：越界转换会静默回绕、腐蚀 range
	//（如 MaxInt64 → MinInt64）。全量边界已在上面兜住，其余越界 fail-fast
	// 而非产出错误语义（C2Y-03）。
	const exactLimit = float64(1 << 53)
	if min < -exactLimit || max > exactLimit {
		return fmt.Errorf("%s: integer bound beyond ±2^53 cannot be represented exactly in JSON/float64 — drop the bounds or narrow the range", path)
	}
	lo, hi := int64(min), int64(max)
	w.line("type %s {", intTypeFor(lo, hi))
	w.in()
	w.line("range \"%d..%d\";", lo, hi)
	w.out()
	w.line("}")
	return nil
}

// intTypeFor picks the smallest YANG integer type containing [min,max]
// (deterministic, round-trip idempotent: uint16 1..4094 → [1,4094] → uint16).
func intTypeFor(min, max int64) string {
	if min >= 0 {
		switch {
		case max <= math.MaxUint8:
			return "uint8"
		case max <= math.MaxUint16:
			return "uint16"
		case max <= math.MaxUint32:
			return "uint32"
		default:
			return "uint64"
		}
	}
	switch {
	case min >= math.MinInt8 && max <= math.MaxInt8:
		return "int8"
	case min >= math.MinInt16 && max <= math.MaxInt16:
		return "int16"
	case min >= math.MinInt32 && max <= math.MaxInt32:
		return "int32"
	default:
		return "int64"
	}
}

// guard fail-fasts on OpenAPI constructs outside the mappable set, naming the
// field's JSON path (C2Y-03 — no silent downgrade). allOf is special-cased in
// emitStringType (crdgen multi-pattern carrier), so it is only rejected here
// for non-string nodes.
func guard(s *apiextv1.JSONSchemaProps, path string) error {
	checks := []struct {
		bad  bool
		name string
	}{
		{s.Format != "", "format"},
		{s.Nullable, "nullable"},
		{len(s.OneOf) > 0, "oneOf"},
		{len(s.AnyOf) > 0, "anyOf"},
		{s.Not != nil, "not"},
		{s.Ref != nil, "$ref"},
		{s.Default != nil, "default"},
		{s.AdditionalProperties != nil, "additionalProperties"},
		{s.AdditionalItems != nil, "additionalItems"},
		{s.MultipleOf != nil, "multipleOf"},
		{s.MaxLength != nil, "maxLength"},
		{s.MinLength != nil, "minLength"},
		{s.MaxItems != nil, "maxItems"},
		{s.MinItems != nil, "minItems"},
		{s.UniqueItems, "uniqueItems"},
		{s.MaxProperties != nil, "maxProperties"},
		{s.MinProperties != nil, "minProperties"},
		{len(s.AllOf) > 0 && s.Type != "string", "allOf"},
		{s.XPreserveUnknownFields != nil, "x-kubernetes-preserve-unknown-fields"},
		{s.XEmbeddedResource, "x-kubernetes-embedded-resource"},
		{s.XIntOrString, "x-kubernetes-int-or-string"},
		{len(s.XValidations) > 0, "x-kubernetes-validations"},
		{s.XMapType != nil, "x-kubernetes-map-type"},
		{s.ExclusiveMinimum, "exclusiveMinimum"},
		{s.ExclusiveMaximum, "exclusiveMaximum"},
		{len(s.PatternProperties) > 0, "patternProperties"},
		{len(s.Dependencies) > 0, "dependencies"},
		{len(s.Definitions) > 0, "definitions"},
		{s.ID != "", "id"},
		{s.Schema != "", "$schema"},
		{s.Title != "", "title"},
		{s.ExternalDocs != nil, "externalDocs"},
		{s.Example != nil, "example"},
	}
	for _, c := range checks {
		if c.bad {
			return fmt.Errorf("%s: unsupported OpenAPI construct %q — restrict northbound CRDs to the mappable set or extend the contract (C2Y-03)", path, c.name)
		}
	}

	// 跨类型约束白名单：约束只允许挂在「对口类型」上，挂错类型即 fail-fast——
	// 否则会在各 emit 分支被静默丢弃（C2Y-03「SHALL NOT 静默丢弃」）。
	typed := []struct {
		bad  bool
		name string
	}{
		{s.Type != "integer" && (s.Minimum != nil || s.Maximum != nil), "minimum/maximum"},
		{s.Type != "string" && s.Pattern != "", "pattern"},
		{s.Type != "string" && len(s.Enum) > 0, "enum"},
		{s.Type != "array" && len(s.XListMapKeys) > 0, "x-kubernetes-list-map-keys"},
		{s.Type != "array" && s.XListType != nil, "x-kubernetes-list-type"},
		{s.Type != "array" && s.Items != nil, "items"},
		{s.Type != "object" && len(s.Properties) > 0, "properties"},
		{s.Type != "object" && len(s.Required) > 0, "required"},
	}
	for _, c := range typed {
		if c.bad {
			return fmt.Errorf("%s: construct %q is not applicable to type %q — it would be silently dropped (C2Y-03)", path, c.name, s.Type)
		}
	}
	return nil
}

// validIdentifier enforces YANG identifier rules (RFC7950 §6.2) on names that
// come straight from CRD JSON keys, including the no-"xml"-prefix rule.
func validIdentifier(name, path string) error {
	if !identifierRe.MatchString(name) || strings.HasPrefix(strings.ToLower(name), "xml") {
		return fmt.Errorf("%s: name %q is not a valid YANG identifier", path, name)
	}
	return nil
}

// patternOnly reports whether an allOf entry carries nothing but a pattern
// (the exact shape crdgen emits for multi-pattern strings).
func patternOnly(s *apiextv1.JSONSchemaProps) bool {
	clone := *s
	clone.Pattern = ""
	empty := apiextv1.JSONSchemaProps{}
	cloneJSON, _ := json.Marshal(clone)
	emptyJSON, _ := json.Marshal(empty)
	return string(cloneJSON) == string(emptyJSON)
}

// quote renders a YANG double-quoted string with escaping.
func quote(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\t", `\t`)
	return `"` + r.Replace(s) + `"`
}

// writer is a tiny indented-line emitter (two-space indent, YANG house style).
type writer struct {
	b     strings.Builder
	depth int
}

func (w *writer) line(format string, args ...any) {
	w.b.WriteString(strings.Repeat("  ", w.depth))
	fmt.Fprintf(&w.b, format, args...)
	w.b.WriteByte('\n')
}

func (w *writer) blank()         { w.b.WriteByte('\n') }
func (w *writer) in()            { w.depth++ }
func (w *writer) out()           { w.depth-- }
func (w *writer) String() string { return w.b.String() }
