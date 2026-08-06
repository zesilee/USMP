package api

import (
	"fmt"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ytypes"
)

// buildListQuerySchema：demo 模块——open/entry 为普通 list（BR-13 schema 判定用），
// global 为 container（非 list 负路径用）。
func buildListQuerySchema(t *testing.T) schema.Schema {
	t.Helper()
	str := func() *yang.YangType { return &yang.YangType{Kind: yang.Ystring} }
	openEntry := &yang.Entry{
		Name: "entry", Key: "id", ListAttr: &yang.ListAttr{},
		Dir: map[string]*yang.Entry{"id": {Name: "id", Type: str()}},
	}
	open := &yang.Entry{Name: "open", Dir: map[string]*yang.Entry{"entry": openEntry}}
	global := &yang.Entry{Name: "global", Dir: map[string]*yang.Entry{"x": {Name: "x", Type: str()}}}
	demo := &yang.Entry{Name: "demo", Dir: map[string]*yang.Entry{"open": open, "global": global}}
	root := &yang.Entry{Name: "Device", Dir: map[string]*yang.Entry{"demo": demo}}
	ds := schema.NewSchema()
	schema.AddYgotSchema(ds, &ytypes.Schema{SchemaTree: map[string]*yang.Entry{"Device": root}})
	return ds
}

// BR-13 参数解析：limit 出现即分页模式，域校验 1..1000，filter 语法 <leaf><op><value>。
func TestParseListQuery(t *testing.T) {
	cases := []struct {
		desc    string
		query   string
		wantNil bool
		wantErr bool
		check   func(t *testing.T, p *ListQueryParams)
	}{
		{"无 limit 返回 nil（旧行为通道）", "force_refresh=true", true, false, nil},
		{"limit+offset 正常解析", "limit=10&offset=20", false, false, func(t *testing.T, p *ListQueryParams) {
			if p.Limit != 10 || p.Offset != 20 {
				t.Errorf("got limit=%d offset=%d, want 10/20", p.Limit, p.Offset)
			}
		}},
		{"offset 缺省 0", "limit=10", false, false, func(t *testing.T, p *ListQueryParams) {
			if p.Offset != 0 {
				t.Errorf("offset = %d, want 0", p.Offset)
			}
		}},
		{"limit=0 越下界拒绝", "limit=0", false, true, nil},
		{"limit=1000 上界放行", "limit=1000", false, false, nil},
		{"limit=1001 越上界拒绝", "limit=1001", false, true, nil},
		{"limit 非数字拒绝", "limit=abc", false, true, nil},
		{"offset 负数拒绝", "limit=10&offset=-1", false, true, nil},
		{"filter 等值+包含解析", "limit=10&filter=admin-status%3D%3Dup&filter=name~%3DGE", false, false, func(t *testing.T, p *ListQueryParams) {
			if len(p.Filters) != 2 {
				t.Fatalf("filters = %d, want 2", len(p.Filters))
			}
			if p.Filters[0].Path != "admin-status" || p.Filters[0].Op != filterOpEq || p.Filters[0].Value != "up" {
				t.Errorf("filter[0] = %+v", p.Filters[0])
			}
			if p.Filters[1].Path != "name" || p.Filters[1].Op != filterOpContains || p.Filters[1].Value != "GE" {
				t.Errorf("filter[1] = %+v", p.Filters[1])
			}
		}},
		{"filter 嵌套路径", "limit=10&filter=bandwidth-type/bandwidth-mbps/bandwidth%3D%3D100", false, false, func(t *testing.T, p *ListQueryParams) {
			if p.Filters[0].Path != "bandwidth-type/bandwidth-mbps/bandwidth" {
				t.Errorf("path = %q", p.Filters[0].Path)
			}
		}},
		{"filter 缺操作符拒绝", "limit=10&filter=name", false, true, nil},
		{"filter 值含另一操作符字样时按最早操作符切分", "limit=10&filter=description%3D%3Da~%3Db", false, false, func(t *testing.T, p *ListQueryParams) {
			f := p.Filters[0]
			if f.Path != "description" || f.Op != filterOpEq || f.Value != "a~=b" {
				t.Errorf("filter = %+v, want description == a~=b", f)
			}
		}},
		{"filter 空字段名拒绝", "limit=10&filter=%3D%3Dup", false, true, nil},
		{"sort+sort_dir 解析", "limit=10&sort=mtu&sort_dir=desc", false, false, func(t *testing.T, p *ListQueryParams) {
			if p.SortKey != "mtu" || p.SortDir != "desc" {
				t.Errorf("sort = %q/%q", p.SortKey, p.SortDir)
			}
		}},
		{"sort_dir 缺省 asc", "limit=10&sort=mtu", false, false, func(t *testing.T, p *ListQueryParams) {
			if p.SortDir != "asc" {
				t.Errorf("sort_dir = %q, want asc", p.SortDir)
			}
		}},
		{"sort_dir 非法拒绝", "limit=10&sort=mtu&sort_dir=sideways", false, true, nil},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			v, err := url.ParseQuery(c.query)
			if err != nil {
				t.Fatalf("bad test query: %v", err)
			}
			p, perr := parseListQuery(v)
			if c.wantErr {
				if perr == nil {
					t.Fatal("err = nil, want rejection")
				}
				return
			}
			if perr != nil {
				t.Fatalf("err = %v, want ok", perr)
			}
			if c.wantNil {
				if p != nil {
					t.Fatalf("params = %+v, want nil (no pagination)", p)
				}
				return
			}
			if p == nil {
				t.Fatal("params = nil, want parsed")
			}
			if c.check != nil {
				c.check(t, p)
			}
		})
	}
}

// BR-13 行提取：schema 判定 list 优先，唯一数组值兜底，两者失败拒绝。
func TestExtractListRows(t *testing.T) {
	s := buildListQuerySchema(t)
	rows3 := []interface{}{
		map[string]interface{}{"id": "a"},
		map[string]interface{}{"id": "b"},
		map[string]interface{}{"id": "c"},
	}
	cases := []struct {
		desc    string
		s       schema.Schema
		path    string
		subtree interface{}
		wantN   int
		wantErr bool
	}{
		{"schema 判定 list + list 名键提取", s, "/demo:demo/demo:open/demo:entry",
			map[string]interface{}{"entry": rows3}, 3, false},
		{"schema 为 container 时拒绝", s, "/demo:demo/demo:global",
			map[string]interface{}{"x": "1"}, 0, true},
		{"nil schema 唯一数组兜底", nil, "/nowhere:x/nowhere:y",
			map[string]interface{}{"whatever": rows3}, 3, false},
		{"子树本身是数组", nil, "/a:b/c",
			rows3, 3, false},
		{"nil schema 无数组拒绝", nil, "/a:b/c",
			map[string]interface{}{"x": "1"}, 0, true},
		{"nil schema 多数组歧义拒绝", nil, "/a:b/c",
			map[string]interface{}{"l1": rows3, "l2": rows3}, 0, true},
		{"schema 判定 list 但键缺失时唯一数组兜底", s, "/demo:demo/demo:open/demo:entry",
			map[string]interface{}{"renamed": rows3}, 3, false},
		{"nil 子树返回空行（设备无数据）", s, "/demo:demo/demo:open/demo:entry",
			nil, 0, false},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			rows, err := extractListRows(c.s, c.path, c.subtree)
			if c.wantErr {
				if err == nil {
					t.Fatal("err = nil, want rejection")
				}
				return
			}
			if err != nil {
				t.Fatalf("err = %v, want ok", err)
			}
			if len(rows) != c.wantN {
				t.Errorf("rows = %d, want %d", len(rows), c.wantN)
			}
		})
	}
}

func lqRow(name, status string, mtu float64, extras ...map[string]interface{}) map[string]interface{} {
	r := map[string]interface{}{"name": name, "admin-status": status, "mtu": mtu}
	for _, e := range extras {
		for k, v := range e {
			r[k] = v
		}
	}
	return r
}

// BR-13 过滤/排序/切片：等值、包含（大小写不敏感）、AND、嵌套路径、
// 数值/字符串排序、无 sort 保序、offset 越界空页。
func TestApplyListQuery(t *testing.T) {
	rows := []interface{}{
		lqRow("GE1/0/1", "up", 1500),
		lqRow("GE1/0/2", "down", 9000),
		lqRow("Vlanif100", "up", 1400,
			map[string]interface{}{"nested": map[string]interface{}{"inner": map[string]interface{}{"leaf": "hit"}}}),
		lqRow("ge1/0/3", "up", 1500),
		lqRow("NULL0", "up", 1500, map[string]interface{}{"loopback": true}),
	}

	t.Run("等值过滤", func(t *testing.T) {
		p := applyListQuery(rows, ListQueryParams{Limit: 100,
			Filters: []ListFilter{{Path: "admin-status", Op: filterOpEq, Value: "up"}}})
		if p.Total != 4 || len(p.Rows) != 4 {
			t.Errorf("total=%d rows=%d, want 4/4", p.Total, len(p.Rows))
		}
	})
	t.Run("包含过滤大小写不敏感", func(t *testing.T) {
		p := applyListQuery(rows, ListQueryParams{Limit: 100,
			Filters: []ListFilter{{Path: "name", Op: filterOpContains, Value: "ge1"}}})
		if p.Total != 3 {
			t.Errorf("total = %d, want 3（GE1/0/1、GE1/0/2、ge1/0/3）", p.Total)
		}
	})
	t.Run("多条件 AND", func(t *testing.T) {
		p := applyListQuery(rows, ListQueryParams{Limit: 100, Filters: []ListFilter{
			{Path: "name", Op: filterOpContains, Value: "ge"},
			{Path: "admin-status", Op: filterOpEq, Value: "up"},
		}})
		if p.Total != 2 {
			t.Errorf("total = %d, want 2", p.Total)
		}
	})
	t.Run("嵌套路径过滤", func(t *testing.T) {
		p := applyListQuery(rows, ListQueryParams{Limit: 100,
			Filters: []ListFilter{{Path: "nested/inner/leaf", Op: filterOpEq, Value: "hit"}}})
		if p.Total != 1 {
			t.Errorf("total = %d, want 1", p.Total)
		}
	})
	t.Run("数值等值过滤（float64 字符串化）", func(t *testing.T) {
		p := applyListQuery(rows, ListQueryParams{Limit: 100,
			Filters: []ListFilter{{Path: "mtu", Op: filterOpEq, Value: "9000"}}})
		if p.Total != 1 {
			t.Errorf("total = %d, want 1", p.Total)
		}
	})
	t.Run("布尔值过滤", func(t *testing.T) {
		p := applyListQuery(rows, ListQueryParams{Limit: 100,
			Filters: []ListFilter{{Path: "loopback", Op: filterOpEq, Value: "true"}}})
		if p.Total != 1 {
			t.Errorf("total = %d, want 1", p.Total)
		}
	})
	t.Run("数值排序 desc", func(t *testing.T) {
		p := applyListQuery(rows, ListQueryParams{Limit: 100, SortKey: "mtu", SortDir: "desc"})
		first := p.Rows[0].(map[string]interface{})
		if fmt.Sprintf("%v", first["mtu"]) != "9000" {
			t.Errorf("first mtu = %v, want 9000", first["mtu"])
		}
	})
	t.Run("字符串排序 asc", func(t *testing.T) {
		p := applyListQuery(rows, ListQueryParams{Limit: 100, SortKey: "name", SortDir: "asc"})
		first := p.Rows[0].(map[string]interface{})
		if first["name"] != "GE1/0/1" {
			t.Errorf("first name = %v, want GE1/0/1", first["name"])
		}
	})
	t.Run("无 sort 保持原序", func(t *testing.T) {
		p := applyListQuery(rows, ListQueryParams{Limit: 2, Offset: 0})
		got := p.Rows[0].(map[string]interface{})["name"]
		if got != "GE1/0/1" {
			t.Errorf("first = %v, want 原序 GE1/0/1", got)
		}
	})
	t.Run("排序不改原切片（快照共享安全）", func(t *testing.T) {
		before := rows[0].(map[string]interface{})["name"]
		applyListQuery(rows, ListQueryParams{Limit: 100, SortKey: "name", SortDir: "desc"})
		after := rows[0].(map[string]interface{})["name"]
		if before != after {
			t.Errorf("原切片被就地排序：before=%v after=%v", before, after)
		}
	})
	t.Run("切片翻页", func(t *testing.T) {
		p := applyListQuery(rows, ListQueryParams{Limit: 2, Offset: 2})
		if p.Total != 5 || len(p.Rows) != 2 || p.Limit != 2 || p.Offset != 2 {
			t.Errorf("page = %+v", p)
		}
	})
	t.Run("末页截断", func(t *testing.T) {
		p := applyListQuery(rows, ListQueryParams{Limit: 2, Offset: 4})
		if len(p.Rows) != 1 || p.Total != 5 {
			t.Errorf("rows=%d total=%d, want 1/5", len(p.Rows), p.Total)
		}
	})
	t.Run("offset 越界返回空页", func(t *testing.T) {
		p := applyListQuery(rows, ListQueryParams{Limit: 10, Offset: 100})
		if len(p.Rows) != 0 || p.Total != 5 {
			t.Errorf("rows=%d total=%d, want 0/5", len(p.Rows), p.Total)
		}
	})
	t.Run("过滤后 total 为过滤后总数", func(t *testing.T) {
		p := applyListQuery(rows, ListQueryParams{Limit: 1,
			Filters: []ListFilter{{Path: "admin-status", Op: filterOpEq, Value: "up"}}})
		if p.Total != 4 || len(p.Rows) != 1 {
			t.Errorf("total=%d rows=%d, want 4/1", p.Total, len(p.Rows))
		}
	})
}

// R09：同一快照并发查询无竞态（applyListQuery 必须只读输入）。
func TestApplyListQueryConcurrent(t *testing.T) {
	rows := make([]interface{}, 0, 500)
	for i := 0; i < 500; i++ {
		rows = append(rows, lqRow(fmt.Sprintf("GE1/0/%d", i), "up", float64(1000+i)))
	}
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				p := applyListQuery(rows, ListQueryParams{
					Limit: 10, Offset: (g * 7) % 100,
					Filters: []ListFilter{{Path: "name", Op: filterOpContains, Value: "GE1"}},
					SortKey: "mtu", SortDir: "desc",
				})
				if p.Total != 500 {
					t.Errorf("total = %d, want 500", p.Total)
					return
				}
			}
		}(g)
	}
	wg.Wait()
}

// makeBenchRows 造 n 行接口形态数据（含嵌套叶）。
func makeBenchRows(n int) []interface{} {
	rows := make([]interface{}, 0, n)
	for i := 0; i < n; i++ {
		rows = append(rows, map[string]interface{}{
			"name":         fmt.Sprintf("GE%d/0/%d", i%8, i),
			"admin-status": []string{"up", "down"}[i%2],
			"mtu":          float64(1000 + i%9000),
			"bandwidth-type": map[string]interface{}{
				"bandwidth-mbps": map[string]interface{}{"bandwidth": float64(i % 100)},
			},
		})
	}
	return rows
}

// design D2 承诺：10k 行 filter+sort+slice 毫秒量级。500ms 为防退化护栏
// （宽松上界防 CI 抖动误报——真实耗时应 <50ms，退化成 O(N²) 才会撞线）。
func TestListQuery10kGuard(t *testing.T) {
	rows := makeBenchRows(10000)
	start := time.Now()
	p := applyListQuery(rows, ListQueryParams{
		Limit: 50, Offset: 5000,
		Filters: []ListFilter{{Path: "admin-status", Op: filterOpEq, Value: "up"}},
		SortKey: "mtu", SortDir: "desc",
	})
	elapsed := time.Since(start)
	if p.Total != 5000 || len(p.Rows) != 0 {
		// offset 5000 == 过滤后 total → 越界空页
		t.Errorf("total=%d rows=%d, want 5000/0", p.Total, len(p.Rows))
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("10k 行查询耗时 %v，超过 500ms 护栏（疑似复杂度退化）", elapsed)
	}
}

func BenchmarkApplyListQuery10k(b *testing.B) {
	rows := makeBenchRows(10000)
	q := ListQueryParams{
		Limit: 50, Offset: 100,
		Filters: []ListFilter{{Path: "name", Op: filterOpContains, Value: "ge3"}},
		SortKey: "mtu", SortDir: "asc",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		applyListQuery(rows, q)
	}
}
