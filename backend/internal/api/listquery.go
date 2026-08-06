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
//  1. 子树本身是数组 → 直接取行；
//  2. schema 判定目标为 list 节点 → 优先按 list 名取键下数组；
//  3. 兜底「子树根下唯一数组值」（与前端 normalizeRows 同规则）；
//  4. schema 明确为非 list 节点、或无法定位数组 → 拒绝（调用方转 400）。
//
// nil 子树返回空行（设备无该表数据 = 合法空页，R08 不拒绝）。
// 运行时路径按段剥模块前缀映射 schema 路径（与 deleteGate 同规）。
func extractListRows(s schema.Schema, runtimePath string, subtree interface{}) ([]interface{}, error) {
	if subtree == nil {
		return nil, nil
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
		ln, isList := node.(schema.ListNode)
		if !isList {
			return nil, fmt.Errorf("路径 %s 在模型中不是 YANG list 节点，分页参数无效", runtimePath)
		}
		listName = ln.Name()
	}
	if listName != "" {
		if rows, ok := m[listName].([]interface{}); ok {
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

// lookupSchemaNode 按 deleteGate 同规把运行时路径映射到 schema 节点：
// 逐段剥模块前缀（/vlan:vlan/vlan:vlans → /vlan/vlans）。
func lookupSchemaNode(s schema.Schema, runtimePath string) (schema.Node, bool) {
	if s == nil {
		return nil, false
	}
	segs := strings.Split(strings.Trim(runtimePath, "/"), "/")
	for i, seg := range segs {
		if j := strings.Index(seg, ":"); j >= 0 {
			segs[i] = seg[j+1:]
		}
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
