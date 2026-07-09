package api

import (
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ytypes"
)

// buildGateSchema：demo 模块——
//
//	locked/entry   list 带 ext:operation-exclude "create|delete"（BR-10 拒绝）
//	stats          config false 子树含 list（BR-10 拒绝）
//	open/entry     普通可删 list（放行）
func buildGateSchema(t *testing.T) schema.Schema {
	t.Helper()
	str := func() *yang.YangType { return &yang.YangType{Kind: yang.Ystring} }
	ext := func(kw, arg string) *yang.Statement {
		return &yang.Statement{Keyword: kw, HasArgument: arg != "", Argument: arg}
	}
	lockedEntry := &yang.Entry{
		Name: "entry", Key: "id", ListAttr: &yang.ListAttr{},
		Dir:  map[string]*yang.Entry{"id": {Name: "id", Type: str()}},
		Exts: []*yang.Statement{ext("ext:operation-exclude", "create|delete")},
	}
	locked := &yang.Entry{Name: "locked", Dir: map[string]*yang.Entry{"entry": lockedEntry}}

	statsEntry := &yang.Entry{
		Name: "row", Key: "id", ListAttr: &yang.ListAttr{},
		Dir: map[string]*yang.Entry{"id": {Name: "id", Type: str()}},
	}
	stats := &yang.Entry{Name: "stats", Config: yang.TSFalse, Dir: map[string]*yang.Entry{"row": statsEntry}}

	openEntry := &yang.Entry{
		Name: "entry", Key: "id", ListAttr: &yang.ListAttr{},
		Dir: map[string]*yang.Entry{"id": {Name: "id", Type: str()}},
	}
	open := &yang.Entry{Name: "open", Dir: map[string]*yang.Entry{"entry": openEntry}}

	demo := &yang.Entry{Name: "demo", Dir: map[string]*yang.Entry{"locked": locked, "stats": stats, "open": open}}
	root := &yang.Entry{Name: "Device", Dir: map[string]*yang.Entry{"demo": demo}}
	ds := schema.NewSchema()
	schema.AddYgotSchema(ds, &ytypes.Schema{SchemaTree: map[string]*yang.Entry{"Device": root}})
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
