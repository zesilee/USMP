package validate

import (
	"strings"
	"sync"
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// 合成 schema（IR DTO 构建面）：box{ name(pattern+length), mtu(range 64..9216),
// items(list key=id, min-elements 1){ id } }。
func boxNode(t *testing.T) schema.Node {
	t.Helper()
	m, err := schema.ModuleFromIR(schema.IRModule{
		Name: "box",
		Root: &schema.IRNode{Kind: "container", Name: "box", Path: "/box", Children: []*schema.IRNode{
			{Kind: "leaf", Name: "name", Path: "/box/name", LeafType: "string",
				Pattern: "[a-z-]+", LengthMin: intp(2), LengthMax: intp(5)},
			{Kind: "leaf", Name: "mtu", Path: "/box/mtu", LeafType: "uint16",
				RangeMin: intp(64), RangeMax: intp(9216)},
			{Kind: "list", Name: "items", Path: "/box/items", Keys: []string{"id"}, MinElements: 1, Children: []*schema.IRNode{
				{Kind: "leaf", Name: "id", Path: "/box/items/id", LeafType: "uint16", IsKey: true},
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return m.Root()
}

func intp(v int) *int { return &v }

type item struct {
	Id *uint16 `path:"id"`
}

func (*item) IsYangObject() {}

type box struct {
	Name  *string          `path:"name"`
	Mtu   *uint16          `path:"mtu"`
	Items map[uint16]*item `path:"items"`
}

func (*box) IsYangObject() {}

func sp(s string) *string { return &s }
func up(v uint16) *uint16 { return &v }

func TestObjectTable(t *testing.T) {
	n := boxNode(t)
	cases := []struct {
		name    string
		v       *box
		wantErr string // "" = 通过
	}{
		{"valid", &box{Name: sp("ab-c"), Mtu: up(1500), Items: map[uint16]*item{1: {Id: up(1)}}}, ""},
		{"nil subtree passes", nil, ""},
		{"pattern miss", &box{Name: sp("AB")}, "pattern"},
		{"length under", &box{Name: sp("a")}, "length"},
		{"length over", &box{Name: sp("abcdef")}, "length"},
		{"range under", &box{Mtu: up(1)}, "range"},
		{"range over", &box{Mtu: up(9999)}, "range"},
		{"min-elements empty present", &box{Items: map[uint16]*item{}}, "min-elements"},
		{"min-elements nil map passes", &box{}, ""}, // 冻结：缺失不触发
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Object(n, tc.v)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("want ok, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("want error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

// TestObjectConcurrent：pattern 缓存并发安全（R09）。
func TestObjectConcurrent(t *testing.T) {
	n := boxNode(t)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = Object(n, &box{Name: sp("ab-c"), Mtu: up(1500)})
				_ = Object(n, &box{Name: sp("BAD")})
			}
		}()
	}
	wg.Wait()
}

// TestObjectBadPatternDegrades：模型内非法正则不拦业务（R08 宽容侧）。
func TestObjectBadPatternDegrades(t *testing.T) {
	m, err := schema.ModuleFromIR(schema.IRModule{
		Name: "b",
		Root: &schema.IRNode{Kind: "container", Name: "b", Path: "/b", Children: []*schema.IRNode{
			{Kind: "leaf", Name: "x", Path: "/b/x", LeafType: "string", Pattern: "[未闭合"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	type b struct {
		X *string `path:"x"`
	}
	if err := Object(m.Root(), &b{X: sp("anything")}); err != nil {
		t.Fatalf("bad pattern must degrade to pass, got %v", err)
	}
}
