package api

import (
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// buildGateSchema：demo 模块——
//
//	locked/entry   list 带 ext:operation-exclude "create|delete"（BR-10 拒绝）
//	stats          config false 子树含 list（BR-10 拒绝）
//	open/entry     普通可删 list（放行）
func buildGateSchema(t *testing.T) schema.Schema {
	t.Helper()
	m, err := schema.ModuleFromIR(schema.IRModule{
		Name: "demo",
		Root: &schema.IRNode{Kind: "container", Name: "demo", Path: "/demo", Children: []*schema.IRNode{
			{Kind: "container", Name: "locked", Path: "/demo/locked", Children: []*schema.IRNode{
				{Kind: "list", Name: "entry", Path: "/demo/locked/entry", Keys: []string{"id"},
					OpExcludes: []string{"create", "delete"}, Children: []*schema.IRNode{
						{Kind: "leaf", Name: "id", Path: "/demo/locked/entry/id", LeafType: "string", IsKey: true},
					}},
			}},
			{Kind: "container", Name: "open", Path: "/demo/open", Children: []*schema.IRNode{
				{Kind: "list", Name: "entry", Path: "/demo/open/entry", Keys: []string{"id"}, Children: []*schema.IRNode{
					{Kind: "leaf", Name: "id", Path: "/demo/open/entry/id", LeafType: "string", IsKey: true},
				}},
			}},
			{Kind: "container", Name: "stats", Path: "/demo/stats", ReadOnly: true, Children: []*schema.IRNode{
				{Kind: "list", Name: "row", Path: "/demo/stats/row", ReadOnly: true, Keys: []string{"id"}, Children: []*schema.IRNode{
					{Kind: "leaf", Name: "id", Path: "/demo/stats/row/id", LeafType: "string", IsKey: true, ReadOnly: true},
				}},
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ds := schema.NewSchema()
	ds.AddModule(m)
	return ds
}

// BR-10：删除的模型驱动门禁——operation-exclude∋delete / readonly 拒绝，
// schema 未覆盖路径放行（降级，R08），运行时路径按段剥模块前缀映射 schema 路径。
func TestDeleteGate(t *testing.T) {
	s := buildGateSchema(t)
	cases := []struct {
		desc    string
		path    string
		wantErr bool
	}{
		{"operation-exclude 含 delete 的 list 拒绝", "/demo:demo/demo:locked/demo:entry", true},
		{"list 的包裹容器路径也拒绝（取单 list 子节点判定）", "/demo:demo/demo:locked", true},
		{"readonly 子树 list 拒绝", "/demo:demo/demo:stats/demo:row", true},
		{"readonly 容器路径拒绝", "/demo:demo/demo:stats", true},
		{"普通可删 list 放行", "/demo:demo/demo:open/demo:entry", false},
		{"schema 未覆盖路径放行（降级）", "/nowhere:x/nowhere:y", false},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			err := deleteGate(s, c.path)
			if c.wantErr && err == nil {
				t.Error("err = nil, want gate rejection")
			}
			if !c.wantErr && err != nil {
				t.Errorf("err = %v, want allow", err)
			}
		})
	}
}
