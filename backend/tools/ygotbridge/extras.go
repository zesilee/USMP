package ygotbridge

import (
	"reflect"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

func intPtr(v int) *int { return &v }

// extLocalName returns the local name of an extension keyword, i.e. the part
// after the module-prefix colon ("ext:support-filter" → "support-filter").
// Prefixes vary with each module's import alias, so matching is prefix-agnostic.
func extLocalName(kw string) string {
	if i := strings.LastIndex(kw, ":"); i >= 0 {
		return kw[i+1:]
	}
	return kw
}

// extSupportFilter：vendor `support-filter` 扩展且实参为 "true"（大小写不敏感，
// BR-07 查询字段标记）；实参缺失/不可解析降级 false（R08）。
func extSupportFilter(e *yang.Entry) bool {
	for _, x := range e.Exts {
		if x != nil && extLocalName(x.Keyword) == "support-filter" &&
			strings.EqualFold(strings.TrimSpace(x.Argument), "true") {
			return true
		}
	}
	return false
}

// extDynamicDefault：vendor `dynamic-default` 扩展存在即真（BR-10）；仅取布尔
// 存在性，不求值表达式（R08——无解析即无解析失败）。
func extDynamicDefault(e *yang.Entry) bool {
	for _, x := range e.Exts {
		if x != nil && extLocalName(x.Keyword) == "dynamic-default" {
			return true
		}
	}
	return false
}

// extOperationExcludes：vendor `operation-exclude` 实参规范化为操作清单（按
// `|`/`,` 切分、trim、小写；真机 IFM 用 "update|delete"）；缺失/空实参为 nil（R08）。
func extOperationExcludes(e *yang.Entry) []string {
	var out []string
	for _, x := range e.Exts {
		if x == nil || extLocalName(x.Keyword) != "operation-exclude" {
			continue
		}
		for _, op := range strings.FieldsFunc(x.Argument, func(r rune) bool {
			return r == '|' || r == ','
		}) {
			if op = strings.ToLower(strings.TrimSpace(op)); op != "" {
				out = append(out, op)
			}
		}
	}
	return out
}

// leafRangeBounds extracts integer min/max from a leaf's YANG `range`. It returns
// no bounds when: there is no range, the range is merely the type's full default
// (i.e. no explicit `range` statement), or a bound is non-integer/overflows int
// (callers then omit that bound — R08, no panic).
func leafRangeBounds(yt *yang.YangType) (min int, hasMin bool, max int, hasMax bool) {
	if yt == nil || len(yt.Range) == 0 {
		return
	}
	if def := defaultRangeForKind(yt.Kind); def != nil && yt.Range.String() == def.String() {
		return // full type-default range → not an explicit constraint
	}
	if v, err := yt.Range[0].Min.Int(); err == nil {
		min, hasMin = int(v), true
	}
	if v, err := yt.Range[len(yt.Range)-1].Max.Int(); err == nil {
		max, hasMax = int(v), true
	}
	return
}

// leafLengthBounds extracts string `length` bounds. goyang only populates
// Type.Length when a length statement exists (directly or via derived type)；
// 非整数/溢出的界按 R08 省略该侧，不 panic。
func leafLengthBounds(yt *yang.YangType) (min int, hasMin bool, max int, hasMax bool) {
	if yt == nil || len(yt.Length) == 0 {
		return
	}
	if v, err := yt.Length[0].Min.Int(); err == nil {
		min, hasMin = int(v), true
	}
	if v, err := yt.Length[len(yt.Length)-1].Max.Int(); err == nil {
		max, hasMax = int(v), true
	}
	return
}

// defaultRangeForKind returns goyang's full default range for an integer kind, or
// nil for non-integer kinds. Used to distinguish explicit ranges from type bounds.
func defaultRangeForKind(k yang.TypeKind) yang.YangRange {
	switch k {
	case yang.Yint8:
		return yang.Int8Range
	case yang.Yint16:
		return yang.Int16Range
	case yang.Yint32:
		return yang.Int32Range
	case yang.Yint64:
		return yang.Int64Range
	case yang.Yuint8:
		return yang.Uint8Range
	case yang.Yuint16:
		return yang.Uint16Range
	case yang.Yuint32:
		return yang.Uint32Range
	case yang.Yuint64:
		return yang.Uint64Range
	default:
		return nil
	}
}

// firstExtraExpr returns the XPath argument of the first element of a goyang
// Entry.Extra slice (e.g. Extra["when"]/["must"]). It tolerates the two shapes
// that occur: the ygot-unzipped JSON map ({"Name": "<xpath>"}) and the
// goyang-parsed *yang.Value struct (exported Name field). Returns "" if absent
// or unrecognized — callers degrade gracefully (R08), never panic.
func firstExtraExpr(extra []interface{}) string {
	for _, v := range extra {
		if s := extraExprName(v); s != "" {
			return s
		}
	}
	return ""
}

// allExtraExprs returns the XPath argument of every element of an Entry.Extra slice
// (order-preserved, empties skipped). Used for `must` where a leaf may carry many.
func allExtraExprs(extra []interface{}) []string {
	var out []string
	for _, v := range extra {
		if s := extraExprName(v); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func extraExprName(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case map[string]interface{}:
		if s, ok := t["Name"].(string); ok {
			return s
		}
		return ""
	case interface{ NName() string }:
		return t.NName()
	}
	// Reflection fallback for structs (e.g. *yang.Value) with an exported Name.
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Struct {
		if f := rv.FieldByName("Name"); f.IsValid() && f.Kind() == reflect.String {
			return f.String()
		}
	}
	return ""
}

// mapLeafType maps a resolved goyang YANG type to the framework LeafType.
func mapLeafType(yt *yang.YangType) schema.LeafType {
	if yt == nil {
		return schema.LeafTypeString
	}
	switch yt.Kind {
	case yang.Ybool:
		return schema.LeafTypeBoolean
	case yang.Yint8:
		return schema.LeafTypeInt8
	case yang.Yint16:
		return schema.LeafTypeInt16
	case yang.Yint32:
		return schema.LeafTypeInt32
	case yang.Yint64:
		return schema.LeafTypeInt64
	case yang.Yuint8:
		return schema.LeafTypeUint8
	case yang.Yuint16:
		return schema.LeafTypeUint16
	case yang.Yuint32:
		return schema.LeafTypeUint32
	case yang.Yuint64:
		return schema.LeafTypeUint64
	case yang.Yenum, yang.Yidentityref:
		return schema.LeafTypeEnum
	case yang.Yempty:
		return schema.LeafTypeEmpty
	case yang.Ydecimal64:
		return schema.LeafTypeDecimal64
	case yang.Ybits:
		return schema.LeafTypeBits
	default:
		// string / union / leafref / binary / instance-identifier
		return schema.LeafTypeString
	}
}
