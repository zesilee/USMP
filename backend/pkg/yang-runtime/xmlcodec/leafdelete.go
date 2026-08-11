package xmlcodec

import (
	"fmt"
	"strings"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// EncodeLeafDelete builds a keyed edit-config fragment that deletes the given
// scalar leaves inside each list entry of v (CS-05, 字段级清除的删除语义)：
// 外层模块容器 + 条目元素（普通打开，绝不携带 operation——不误删条目）+ key 叶
// 定位 + 每个目标叶自闭合并带 nc:operation="delete"。leaves 应用于 v 中的每个
// 条目（调用方按变更集条目逐条构造）。
//
// 防线（R08，绝不发送无目标/指错目标的删除）：
//   - 空 leaves / 空条目集合 / nil 容器 → 明确错误
//   - schema 可用时目标必须存在且为叶（leaf/leaf-list）；嵌套容器与嵌套 list 拒绝
//   - list key 叶本身拒绝（删除定位键无意义且危险）
//   - 条目 key 无法确定（无 ΛListKeyMap 且 schema 无单 key）→ 明确错误
func EncodeLeafDelete(spec *Spec, v interface{}, leaves []string) (string, error) {
	if len(leaves) == 0 {
		return "", fmt.Errorf("xmlcodec leaf delete: empty leaf set")
	}
	cv, err := derefContainer(v)
	if err != nil {
		return "", err
	}
	mapVal, elemTag, err := containerMap(cv)
	if err != nil {
		return "", err
	}
	r, err := spec.resolve(elemTag)
	if err != nil {
		return "", err
	}
	if mapVal.IsNil() || mapVal.Len() == 0 {
		return "", fmt.Errorf("xmlcodec leaf delete: empty %s target", elemTag)
	}

	keySet := map[string]bool{}
	for _, k := range r.keyNames() {
		keySet[k] = true
	}
	for _, leaf := range leaves {
		if keySet[leaf] {
			return "", fmt.Errorf("xmlcodec leaf delete: %s is the list key of %s", leaf, elemTag)
		}
		if r.list != nil && nodeHasChildren(r.list) {
			child := nodeChild(r.list, leaf)
			if child == nil {
				return "", fmt.Errorf("xmlcodec leaf delete: %s not in schema of %s", leaf, elemTag)
			}
			if child.Type() != schema.LeafNodeType {
				return "", fmt.Errorf("xmlcodec leaf delete: %s is not a leaf (kind %v)", leaf, child.Type())
			}
		}
	}

	var b strings.Builder
	wrapped := openWrappers(&b, r)
	if wrapped {
		fmt.Fprintf(&b, "<%s>", r.root)
	} else {
		fmt.Fprintf(&b, "<%s xmlns=%q>", r.root, r.ns)
	}
	for _, mk := range sortedKeys(mapVal) {
		ev := mapVal.MapIndex(mk)
		keyName, keyVal, err := deleteKey(ev, mk, r)
		if err != nil {
			return "", fmt.Errorf("xmlcodec leaf delete %s: %w", elemTag, err)
		}
		fmt.Fprintf(&b, "<%s>", elemTag)
		if err := encodeLeaf(&b, keyName, keyVal, ""); err != nil {
			return "", fmt.Errorf("xmlcodec leaf delete %s: %w", elemTag, err)
		}
		for _, leaf := range leaves {
			fmt.Fprintf(&b, `<%s nc:operation="delete" xmlns:nc=%q/>`, leaf, NetconfBaseNS)
		}
		fmt.Fprintf(&b, "</%s>", elemTag)
	}
	fmt.Fprintf(&b, "</%s>", r.root)
	closeWrappers(&b, r)
	return b.String(), nil
}
