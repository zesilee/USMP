// 基于 Schema IR 的服务端校验器（retire-ygot-runtime YN-04，任务5.2）：
// pattern（锚定全匹配）/range/length/min-elements 四类叶约束，替代 ygot 生成的
// Validate()。
//
// **语义冻结 = ygot Validate 行为快照**（internal/intent/validate_snapshot_test
// 实证）：mandatory 不校验（必填防线在 CRD OpenAPI required）、min-elements 仅对
// 存在的空 list 生效（list 缺失不触发）、must/when 不做运行时求值（设备侧兜底）。
// 收紧任一项都属契约变更，须先改快照拍板。
//
// 实现为反射走查（`path` tag 对齐 schema 树），与生成结构体的接口族无关——
// ygot 与 native 结构体同样适用（S3 切换期两族通吃）。
//
// 住在 schema 包内而非独立成包，是因为走查依赖 Node/LeafNode/ListNode 等 IR 类型：
// 独立成包必然 import schema，schema.Validate 再委托回去就成循环依赖，接口方法
// 因此长期空置。实现本就是纯 IR 走查、不依赖任何其他包，此处是它本来的位置
// （change config-write-validation D1）。
package schema

import (
	"fmt"
	"reflect"
	"regexp"
	"sync"
)

var (
	patternMu    sync.RWMutex
	patternCache = map[string]*regexp.Regexp{}
)

// compilePattern 返回锚定全匹配的编译正则（YANG pattern 语义），并发安全缓存。
func compilePattern(p string) (*regexp.Regexp, error) {
	patternMu.RLock()
	re, ok := patternCache[p]
	patternMu.RUnlock()
	if ok {
		return re, nil
	}
	re, err := regexp.Compile("^(?:" + p + ")$")
	if err != nil {
		return nil, err
	}
	patternMu.Lock()
	patternCache[p] = re
	patternMu.Unlock()
	return re, nil
}

// ValidateObject validates a generated YANG object subtree against its schema node.
// v 为容器指针（如 *BusinessVlanService），n 为对应 schema 节点。
func ValidateObject(n Node, v interface{}) error {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || rv.Kind() != reflect.Ptr || rv.IsNil() {
		return nil // 空子树无可校验（与 ygot 对 nil 容器行为一致）
	}
	if rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("validate: %T is not a container struct", v)
	}
	return walkStruct(n, rv.Elem(), "")
}

func walkStruct(n Node, sv reflect.Value, path string) error {
	st := sv.Type()
	for i := 0; i < st.NumField(); i++ {
		tag := pathTag(st.Field(i))
		if tag == "" {
			continue
		}
		child := childNode(n, tag)
		fv := sv.Field(i)
		fpath := path + "/" + tag
		switch fv.Kind() {
		case reflect.Ptr:
			if fv.IsNil() {
				continue
			}
			if fv.Type().Elem().Kind() == reflect.Struct {
				if err := walkStruct(child, fv.Elem(), fpath); err != nil {
					return err
				}
				continue
			}
			if err := checkLeaf(child, fv.Elem(), fpath); err != nil {
				return err
			}
		case reflect.Slice:
			if fv.Type().Name() != "" { // object.Binary 等具名切片
				continue
			}
			for j := 0; j < fv.Len(); j++ {
				if err := checkLeaf(child, fv.Index(j), fpath); err != nil {
					return err
				}
			}
		case reflect.Map:
			if ln, ok := child.(ListNode); ok && fv.Len() == 0 && ln.MinElements() > 0 {
				// 冻结语义：仅存在的空 list 触发（nil map 的 Len 也为 0——
				// 但 ygot 对 nil map 不拒。区分 nil 与空。）
				if !fv.IsNil() {
					return fmt.Errorf("validate: %s: 至少需要 %d 个条目（min-elements）", fpath, ln.MinElements())
				}
			}
			it := fv.MapRange()
			for it.Next() {
				ev := it.Value()
				if ev.Kind() == reflect.Ptr && !ev.IsNil() {
					if err := walkStruct(child, ev.Elem(), fpath); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// checkLeaf applies pattern/length/range to a scalar leaf value.
func checkLeaf(n Node, v reflect.Value, path string) error {
	leaf, ok := n.(LeafNode)
	if !ok {
		return nil // schema 未知的字段不拦（宽容对齐 ygot：结构面已由生成保证）
	}
	switch v.Kind() {
	case reflect.String:
		s := v.String()
		if p := leaf.Pattern(); p != "" {
			re, err := compilePattern(p)
			if err != nil {
				return nil // 模型内非法正则：不因校验器自身失败拦业务（R08）
			}
			if !re.MatchString(s) {
				return fmt.Errorf("validate: %s: 值 %q 不匹配 pattern %q", path, s, p)
			}
		}
		if lm, hasLen := leafLengths(leaf); hasLen {
			if n := len(s); n < lm[0] || n > lm[1] {
				return fmt.Errorf("validate: %s: 长度 %d 超出 length [%d,%d]", path, n, lm[0], lm[1])
			}
		}
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if err := checkRange(leaf, v.Int(), path); err != nil {
			return err
		}
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if v.Uint() > uint64(1)<<62 {
			return nil // 超 int 表示域的极值不硬判（IR 界为 int）
		}
		if err := checkRange(leaf, int64(v.Uint()), path); err != nil {
			return err
		}
	}
	return nil
}

func checkRange(leaf LeafNode, val int64, path string) error {
	if min, ok := leaf.RangeMin(); ok && val < int64(min) {
		return fmt.Errorf("validate: %s: 值 %d 低于 range 下界 %d", path, val, min)
	}
	if max, ok := leaf.RangeMax(); ok && val > int64(max) {
		return fmt.Errorf("validate: %s: 值 %d 超出 range 上界 %d", path, val, max)
	}
	return nil
}

// leafLengths 返回 [min,max] 与是否存在（单侧界用极值补齐）。
func leafLengths(leaf LeafNode) ([2]int, bool) {
	type lengther interface {
		LengthMin() (int, bool)
		LengthMax() (int, bool)
	}
	l, ok := leaf.(lengther)
	if !ok {
		return [2]int{}, false
	}
	min, hasMin := l.LengthMin()
	max, hasMax := l.LengthMax()
	if !hasMin && !hasMax {
		return [2]int{}, false
	}
	if !hasMin {
		min = 0
	}
	if !hasMax {
		max = int(^uint(0) >> 1)
	}
	return [2]int{min, max}, true
}

func pathTag(f reflect.StructField) string {
	tag := f.Tag.Get("path")
	if i := indexByte(tag, '|'); i >= 0 {
		tag = tag[:i]
	}
	return tag
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

// childNode 按名取子节点，穿透 choice/case（约束叶可能在 choice 内）。
func childNode(n Node, name string) Node {
	if n == nil {
		return nil
	}
	var kids []Node
	switch t := n.(type) {
	case ChoiceNode:
		for _, cs := range t.Cases() {
			if got := childNode(cs, name); got != nil {
				return got
			}
		}
		return nil
	case ListNode:
		if c, ok := t.Child(name); ok {
			return c
		}
		kids = t.Children()
	case ContainerNode:
		if c, ok := t.Child(name); ok {
			return c
		}
		kids = t.Children()
	case CaseNode:
		if c, ok := t.Child(name); ok {
			return c
		}
		kids = t.Children()
	}
	for _, c := range kids {
		if c.Type() == ChoiceNodeType || c.Type() == CaseNodeType {
			if got := childNode(c, name); got != nil {
				return got
			}
		}
	}
	return nil
}
