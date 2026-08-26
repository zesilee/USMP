package schema

import (
	"strings"
	"testing"
)

// Schema.Validate 曾是空壳：只查路径存在性就 return nil，且全仓零调用方——
// 长得像一道防线、实际什么都不做（code-todo-backlog A2）。成因是包依赖方向：
// 校验实现原先独立成包并 import schema，schema 反过来委托即成循环依赖。
// 实现迁入本包后接口方法得以填实，本用例锁死「不再恒返回 nil」。

// validatableSchema 造一个装了 box 模块的 DefaultSchema（约束同 validate_test.go）。
func validatableSchema(t *testing.T) *DefaultSchema {
	t.Helper()
	m, err := ModuleFromIR(IRModule{
		Name: "box",
		Root: &IRNode{Kind: "container", Name: "box", Path: "/box", Children: []*IRNode{
			{Kind: "leaf", Name: "name", Path: "/box/name", LeafType: "string",
				Pattern: "[a-z-]+", LengthMin: vIntp(2), LengthMax: vIntp(5)},
			{Kind: "leaf", Name: "mtu", Path: "/box/mtu", LeafType: "uint16",
				RangeMin: vIntp(64), RangeMax: vIntp(9216)},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := NewSchema()
	s.AddModule(m)
	return s
}

func TestSchemaValidateRejectsConstraintViolation(t *testing.T) {
	s := validatableSchema(t)

	cases := []struct {
		name    string
		cfg     *vBox
		wantErr string // "" = 应通过
	}{
		{"合法配置通过", &vBox{Name: vSp("ab-c"), Mtu: vUp(1500)}, ""},
		{"pattern 不匹配被拒", &vBox{Name: vSp("AB")}, "pattern"},
		{"长度超限被拒", &vBox{Name: vSp("abcdef")}, "length"},
		{"数值越界被拒", &vBox{Mtu: vUp(9999)}, "range"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := s.Validate("/box", tc.cfg)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("合法配置不应被拒: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("违约配置必须被拒，Validate 不得恒返回 nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("错误应指明被违反的约束 %q，实得 %v", tc.wantErr, err)
			}
		})
	}
}

// 错误必须点名到叶路径，否则调用方无从定位是哪个字段出的问题。
func TestSchemaValidateErrorNamesLeafPath(t *testing.T) {
	s := validatableSchema(t)

	err := s.Validate("/box", &vBox{Name: vSp("TOOLONG")})
	if err == nil {
		t.Fatal("违约配置必须被拒")
	}
	if !strings.Contains(err.Error(), "/name") {
		t.Errorf("错误应点名叶路径 /name，实得 %v", err)
	}
}

// 路径不存在时快速失败——宁可拒绝也不放行未校验配置（沿用原行为）。
func TestSchemaValidateUnknownPathFails(t *testing.T) {
	s := validatableSchema(t)

	if err := s.Validate("/nope", &vBox{}); err == nil {
		t.Error("未知路径应报错，不应静默通过")
	}
}
