package api

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// listquery 实现 BR-13 的 list 行查询引擎：行提取 → 过滤 → 排序 → 切片。
// 纯函数、只读输入（快照在并发请求间共享，R09）；缓存仍存整树，切片只是
// 出口视图（design D1/D2）。

// filter 操作符（BR-13：一期仅等值与包含）。
const (
	filterOpEq       = "eq"       // == 等值（值字符串化后比较）
	filterOpContains = "contains" // ~= 包含（大小写不敏感）
)

// ListFilter 是一条过滤条件；Path 支持嵌套叶路径（如 a/b/c），多条件 AND。
type ListFilter struct {
	Path  string
	Op    string
	Value string
}

// ListQueryParams 是解析后的分页查询参数（limit 出现即进入分页模式）。
type ListQueryParams struct {
	Limit   int
	Offset  int
	Filters []ListFilter
	SortKey string
	SortDir string // "asc" | "desc"
}

// ListPage 是分页模式的响应载荷：rows 为原 list 条目对象（RFC7951，保留类型，
// 禁止 key/value 平铺——NCE 反面教材）。
type ListPage struct {
	Rows   []interface{} `json:"rows"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

// parseListQuery 解析 BR-13 查询参数。无 limit 返回 (nil, nil)——旧行为通道，
// 响应形状不变（回归锚点）。域校验：limit 1..1000、offset ≥0、filter 语法
// <leaf>==<value> 或 <leaf>~=<value>、sort_dir 仅 asc|desc。
func parseListQuery(q url.Values) (*ListQueryParams, error) {
	if q.Get("limit") == "" {
		return nil, nil
	}
	limit, err := strconv.Atoi(q.Get("limit"))
	if err != nil {
		return nil, fmt.Errorf("limit 必须为整数: %q", q.Get("limit"))
	}
	if limit < 1 || limit > 1000 {
		return nil, fmt.Errorf("limit 超出范围 [1..1000]: %d", limit)
	}
	offset := 0
	if raw := q.Get("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil || offset < 0 {
			return nil, fmt.Errorf("offset 必须为非负整数: %q", raw)
		}
	}
	p := &ListQueryParams{Limit: limit, Offset: offset}
	for _, raw := range q["filter"] {
		f, err := parseListFilter(raw)
		if err != nil {
			return nil, err
		}
		p.Filters = append(p.Filters, f)
	}
	p.SortKey = q.Get("sort")
	p.SortDir = q.Get("sort_dir")
	if p.SortDir == "" {
		p.SortDir = "asc"
	}
	if p.SortDir != "asc" && p.SortDir != "desc" {
		return nil, fmt.Errorf("sort_dir 仅支持 asc|desc: %q", p.SortDir)
	}
	return p, nil
}

// parseListFilter 解析单条 filter：取**最早出现**的操作符切分（值本身可能
// 含另一个操作符字样，先到先得才不会错切）；字段名不可为空；值允许为空串
// （等值匹配空值是合法查询）。
func parseListFilter(raw string) (ListFilter, error) {
	bestIdx, bestOp, sepLen := -1, "", 0
	for _, cand := range []struct{ sep, op string }{
		{"~=", filterOpContains},
		{"==", filterOpEq},
	} {
		if i := strings.Index(raw, cand.sep); i >= 0 && (bestIdx < 0 || i < bestIdx) {
			bestIdx, bestOp, sepLen = i, cand.op, len(cand.sep)
		}
	}
	if bestIdx < 0 {
		return ListFilter{}, fmt.Errorf("filter 缺少操作符（==/~=）: %q", raw)
	}
	if bestIdx == 0 {
		return ListFilter{}, fmt.Errorf("filter 字段名为空: %q", raw)
	}
	return ListFilter{Path: raw[:bestIdx], Op: bestOp, Value: raw[bestIdx+sepLen:]}, nil
}

// extractListRows 从整树子树中提取 list 行（BR-13）：
//  1. 祖先段含键谓词（嵌套 list，FIB 形态）→ 先按谓词索引唯一行下钻
//     （快照根 = 停剥返回的父容器子树，见 peelToPath 契约）；
//  2. 子树/下钻结果本身是数组 → 直接取行；
//  3. schema 判定目标为 list 节点（或单 list 子节点的包裹容器）→ 按 list 名取键下数组；
//  4. 兜底「子树根下唯一数组值」（与前端 normalizeRows 同规则）；
//  5. schema 明确为非 list、或无法定位数组 → 拒绝（调用方转 400）。
//
// nil 子树 / 谓词未命中 / 中途容器缺失 → 空行（设备无数据 = 合法空页，R08）。
// 运行时路径按段剥模块前缀映射 schema 路径（与 deleteGate 同规）。
func extractListRows(s schema.Schema, runtimePath string, subtree interface{}) ([]interface{}, error) {
	if subtree == nil {
		return nil, nil
	}
	// 谓词锚定下钻（design D2 补充决策）：无分页参数的读取不走本函数，
	// 停剥契约不受影响。
	if strings.Contains(runtimePath, "[") {
		descended, empty, err := descendPredicates(runtimePath, subtree)
		if err != nil {
			return nil, err
		}
		if empty {
			return nil, nil
		}
		subtree = descended
	}
	if rows, ok := subtree.([]interface{}); ok {
		return rows, nil
	}
	m, ok := subtree.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("路径 %s 的回读数据不是 list（类型 %T），分页参数仅对 YANG list 节点有效", runtimePath, subtree)
	}

	var listName string
	if node, found := lookupSchemaNode(s, runtimePath); found {
		switch t := node.(type) {
		case schema.ListNode:
			listName = t.Name()
		case schema.ContainerNode:
			// 包裹容器（group 裹单 list 的常见形态，与 deleteGate 同规）：
			// 取其唯一 list 子节点；非此形态 → 非 list 拒绝。
			var lists []schema.Node
			for _, ch := range t.Children() {
				if _, isList := ch.(schema.ListNode); isList {
					lists = append(lists, ch)
				}
			}
			if len(lists) != 1 {
				return nil, fmt.Errorf("路径 %s 在模型中不是 YANG list 节点，分页参数无效", runtimePath)
			}
			listName = lists[0].Name()
		default:
			return nil, fmt.Errorf("路径 %s 在模型中不是 YANG list 节点，分页参数无效", runtimePath)
		}
	}
	if listName != "" {
		if rows, ok := lookupArray(m, listName); ok {
			return rows, nil
		}
	}
	// 兜底：唯一数组值（schema 未覆盖 / 键名与 list 名不一致的降级通道）。
	var found []interface{}
	n := 0
	for _, v := range m {
		if arr, ok := v.([]interface{}); ok {
			found = arr
			n++
		}
	}
	switch n {
	case 1:
		return found, nil
	case 0:
		return nil, fmt.Errorf("路径 %s 的回读数据中未找到 list 行数组，分页参数仅对 YANG list 节点有效", runtimePath)
	default:
		return nil, fmt.Errorf("路径 %s 的回读数据含多个数组、无法唯一定位 list 行，请求更深一级的 list 路径", runtimePath)
	}
}

// descendPredicates 从首个谓词段起沿路径下钻：谓词段按键值索引数组唯一行，
// 普通段按局部名进容器。返回 (下钻结果, 是否空页, 错误)：
// 未命中/中途缺失 → 空页；多命中（键不完整）→ 错误。
// 已知边界：谓词值含 '/'（pathLocals 按 '/' 切分不感知引号）会产生带 '[' 无 ']'
// 的破段，此处明确报错而非静默错配。
func descendPredicates(runtimePath string, subtree interface{}) (interface{}, bool, error) {
	segs := pathLocals(runtimePath)
	first := -1
	for i, seg := range segs {
		if strings.Contains(seg, "[") {
			first = i
			break
		}
	}
	if first < 0 {
		return subtree, false, nil
	}
	cur := subtree
	for _, seg := range segs[first:] {
		m, ok := cur.(map[string]interface{})
		if !ok {
			return nil, true, nil // 中途形状意外/缺失 → 空页（R08 不崩）
		}
		if !strings.Contains(seg, "[") {
			child, ok := lookupLocal(m, seg)
			if !ok {
				return nil, true, nil
			}
			cur = child
			continue
		}
		name, preds, err := parsePredicateSeg(seg)
		if err != nil {
			return nil, false, err
		}
		arr, ok := lookupArray(m, name)
		if !ok {
			return nil, true, nil
		}
		var hit interface{}
		hits := 0
		for _, row := range arr {
			rm, ok := row.(map[string]interface{})
			if !ok {
				continue
			}
			match := true
			for k, v := range preds {
				got, ok := lookupLocal(rm, k)
				if !ok || stringifyLeaf(got) != v {
					match = false
					break
				}
			}
			if match {
				hit = row
				hits++
			}
		}
		switch {
		case hits == 0:
			return nil, true, nil // 谓词未命中 → 空页（设备无该行）
		case hits > 1:
			return nil, false, fmt.Errorf("路径 %s 谓词命中多行（键不完整），无法唯一锚定", runtimePath)
		}
		cur = hit
	}
	return cur, false, nil
}

// parsePredicateSeg 解析 list 谓词段 name[k1=v1][k2=v2]…：值可被单/双引号包裹。
func parsePredicateSeg(seg string) (string, map[string]string, error) {
	i := strings.Index(seg, "[")
	name := seg[:i]
	preds := map[string]string{}
	rest := seg[i:]
	for rest != "" {
		if !strings.HasPrefix(rest, "[") {
			return "", nil, fmt.Errorf("谓词段格式非法: %q", seg)
		}
		j := strings.Index(rest, "]")
		if j < 0 {
			return "", nil, fmt.Errorf("谓词段缺少 ']'（谓词值含 '/' 暂不支持）: %q", seg)
		}
		kv := rest[1:j]
		eq := strings.Index(kv, "=")
		if eq <= 0 {
			return "", nil, fmt.Errorf("谓词键值格式非法: %q", kv)
		}
		val := strings.Trim(kv[eq+1:], `'"`)
		preds[kv[:eq]] = val
		rest = rest[j+1:]
	}
	if len(preds) == 0 {
		return "", nil, fmt.Errorf("谓词段无键值: %q", seg)
	}
	return name, preds, nil
}

// lookupArray 按局部名取 map 下的数组值（键可能带模块前缀）。
func lookupArray(m map[string]interface{}, local string) ([]interface{}, bool) {
	v, ok := lookupLocal(m, local)
	if !ok {
		return nil, false
	}
	arr, ok := v.([]interface{})
	return arr, ok
}

// predicateFetchPath 分页模式的设备取数路径：截到首个谓词段之前的父容器
// （bracket 感知切分，谓词值可含 '/'）。深路径 subtree filter 选不回祖先 list
// 的键叶（RFC6241 selection 语义，sim/真机同口径），谓词下钻会匹配不到行——
// 故整列表连键取回，快照按父容器共享（不同谓词行共用一份快照）。
// 无谓词、或谓词在首段（无父可截）→ 返回原路径与 false。
func predicateFetchPath(path string) (string, bool) {
	norm := "/" + strings.Trim(strings.TrimSpace(path), "/")
	var segs []string
	var cur strings.Builder
	depth := 0
	for _, r := range strings.Trim(norm, "/") {
		switch {
		case r == '[':
			depth++
			cur.WriteRune(r)
		case r == ']':
			if depth > 0 {
				depth--
			}
			cur.WriteRune(r)
		case r == '/' && depth == 0:
			segs = append(segs, cur.String())
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		segs = append(segs, cur.String())
	}
	for i, seg := range segs {
		if strings.Contains(seg, "[") {
			if i == 0 {
				return path, false
			}
			return "/" + strings.Join(segs[:i], "/"), true
		}
	}
	return path, false
}

// lookupSchemaNode 按 deleteGate 同规把运行时路径映射到 schema 节点：
// 逐段剥模块前缀并剥除 list 谓词（/vlan:vlan/vlans[id=1] → /vlan/vlans）。
func lookupSchemaNode(s schema.Schema, runtimePath string) (schema.Node, bool) {
	if s == nil {
		return nil, false
	}
	segs := strings.Split(strings.Trim(runtimePath, "/"), "/")
	for i, seg := range segs {
		if j := strings.Index(seg, "["); j >= 0 {
			seg = seg[:j]
		}
		if j := strings.Index(seg, ":"); j >= 0 {
			seg = seg[j+1:]
		}
		segs[i] = seg
	}
	return s.Path("/" + strings.Join(segs, "/"))
}

// applyListQuery 过滤 → 排序 → 切片。输入只读：排序前复制行切片
// （快照在并发请求间共享，就地排序即数据竞态，R09）。
func applyListQuery(rows []interface{}, p ListQueryParams) ListPage {
	filtered := rows
	if len(p.Filters) > 0 {
		filtered = make([]interface{}, 0, len(rows))
		for _, r := range rows {
			if rowMatches(r, p.Filters) {
				filtered = append(filtered, r)
			}
		}
	}
	if p.SortKey != "" {
		sorted := make([]interface{}, len(filtered))
		copy(sorted, filtered)
		desc := p.SortDir == "desc"
		sort.SliceStable(sorted, func(i, j int) bool {
			vi, iok := rowValue(sorted[i], p.SortKey)
			vj, jok := rowValue(sorted[j], p.SortKey)
			// 缺失值恒排最后（与方向无关）。
			if !iok || !jok {
				return iok && !jok
			}
			if desc {
				return valueLess(vj, vi)
			}
			return valueLess(vi, vj)
		})
		filtered = sorted
	}
	total := len(filtered)
	start := p.Offset
	if start > total {
		start = total
	}
	end := start + p.Limit
	if end > total {
		end = total
	}
	// 空页也返回非 nil rows（JSON 序列化为 [] 而非 null）。
	page := make([]interface{}, end-start)
	copy(page, filtered[start:end])
	return ListPage{Rows: page, Total: total, Limit: p.Limit, Offset: p.Offset}
}

// rowMatches 多条件 AND；值先字符串化再比较（RFC7951 数值/布尔一律按
// 字符串语义，与前端搜索面板口径一致）。
func rowMatches(row interface{}, filters []ListFilter) bool {
	for _, f := range filters {
		v, ok := rowValue(row, f.Path)
		if !ok {
			return false
		}
		s := stringifyLeaf(v)
		switch f.Op {
		case filterOpEq:
			if s != f.Value {
				return false
			}
		case filterOpContains:
			if !strings.Contains(strings.ToLower(s), strings.ToLower(f.Value)) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// rowValue 按嵌套路径（a/b/c）取行内叶值。
func rowValue(row interface{}, path string) (interface{}, bool) {
	cur := row
	for _, seg := range strings.Split(path, "/") {
		m, ok := cur.(map[string]interface{})
		if !ok {
			return nil, false
		}
		cur, ok = m[seg]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

// stringifyLeaf 把叶值归一为字符串：float64 整数值不带小数点（1500 而非 1500.0）。
func stringifyLeaf(v interface{}) string {
	if f, ok := v.(float64); ok {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	return fmt.Sprintf("%v", v)
}

// valueLess 排序比较：两侧均可解析为数值时按数值，否则字符串（BR-13）。
func valueLess(a, b interface{}) bool {
	sa, sb := stringifyLeaf(a), stringifyLeaf(b)
	fa, errA := strconv.ParseFloat(sa, 64)
	fb, errB := strconv.ParseFloat(sb, 64)
	if errA == nil && errB == nil {
		return fa < fb
	}
	return sa < sb
}
