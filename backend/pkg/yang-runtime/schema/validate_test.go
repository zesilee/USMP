package schema

import (
	"strings"
	"sync"
	"testing"
)

// 本文件自 pkg/yang-runtime/validate 迁入（change config-write-validation 任务 1.3）。
// 用例逐条保持原样——迁移的验收标准是行为等价，不是顺便改进。

// 合成 schema（IR DTO 构建面）：box{ name(pattern+length), mtu(range 64..9216),
// items(list key=id, min-elements 1){ id } }。
func vBoxNode(t *testing.T) Node {
	t.Helper()
	m, err := ModuleFromIR(IRModule{
		Name: "box",
		Root: &IRNode{Kind: "container", Name: "box", Path: "/box", Children: []*IRNode{
			{Kind: "leaf", Name: "name", Path: "/box/name", LeafType: "string",
				Pattern: "[a-z-]+", LengthMin: vIntp(2), LengthMax: vIntp(5)},
			{Kind: "leaf", Name: "mtu", Path: "/box/mtu", LeafType: "uint16",
				RangeMin: vIntp(64), RangeMax: vIntp(9216)},
			{Kind: "list", Name: "items", Path: "/box/items", Keys: []string{"id"}, MinElements: 1, Children: []*IRNode{
				{Kind: "leaf", Name: "id", Path: "/box/items/id", LeafType: "uint16", IsKey: true},
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return m.Root()
}

func vIntp(v int) *int { return &v }

type vItem struct {
	Id *uint16 `path:"id"`
}

func (*vItem) IsYangObject() {}

type vBox struct {
	Name  *string           `path:"name"`
	Mtu   *uint16           `path:"mtu"`
	Items map[uint16]*vItem `path:"items"`
}

func (*vBox) IsYangObject() {}

func vSp(s string) *string { return &s }
func vUp(v uint16) *uint16 { return &v }

func TestValidateObjectTable(t *testing.T) {
	n := vBoxNode(t)
	cases := []struct {
		name    string
		v       *vBox
		wantErr string // "" = 通过
	}{
		{"valid", &vBox{Name: vSp("ab-c"), Mtu: vUp(1500), Items: map[uint16]*vItem{1: {Id: vUp(1)}}}, ""},
		{"nil subtree passes", nil, ""},
		{"pattern miss", &vBox{Name: vSp("AB")}, "pattern"},
		{"length under", &vBox{Name: vSp("a")}, "length"},
		{"length over", &vBox{Name: vSp("abcdef")}, "length"},
		{"range under", &vBox{Mtu: vUp(1)}, "range"},
		{"range over", &vBox{Mtu: vUp(9999)}, "range"},
		{"min-elements empty present", &vBox{Items: map[uint16]*vItem{}}, "min-elements"},
		{"min-elements nil map passes", &vBox{}, ""}, // 冻结：缺失不触发
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateObject(n, tc.v)
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

// TestValidateObjectConcurrent：pattern 缓存并发安全（R09）。
func TestValidateObjectConcurrent(t *testing.T) {
	n := vBoxNode(t)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = ValidateObject(n, &vBox{Name: vSp("ab-c"), Mtu: vUp(1500)})
				_ = ValidateObject(n, &vBox{Name: vSp("BAD")})
			}
		}()
	}
	wg.Wait()
}

// TestValidateObjectBadPatternDegrades：模型内非法正则不拦业务（R08 宽容侧）。
func TestValidateObjectBadPatternDegrades(t *testing.T) {
	m, err := ModuleFromIR(IRModule{
		Name: "b",
		Root: &IRNode{Kind: "container", Name: "b", Path: "/b", Children: []*IRNode{
			{Kind: "leaf", Name: "x", Path: "/b/x", LeafType: "string", Pattern: "[未闭合"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	type b struct {
		X *string `path:"x"`
	}
	if err := ValidateObject(m.Root(), &b{X: vSp("anything")}); err != nil {
		t.Fatalf("bad pattern must degrade to pass, got %v", err)
	}
}
