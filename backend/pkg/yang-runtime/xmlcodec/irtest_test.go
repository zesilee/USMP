package xmlcodec

import (
	"github.com/leezesi/usmp/backend/internal/yangschema"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// irTestNode 是测试用 Spec.Schema 构造器：按数据路径从 Schema IR 取节点
// （S1 起引擎 schema 源为 IR；yangschema.Load 记忆化，测试面零成本）。
func irTestNode(path string) func() schema.Node {
	return func() schema.Node {
		s, err := yangschema.Load()
		if err != nil {
			return nil
		}
		n, _ := s.Path(path)
		return n
	}
}

// irTestNodeAt 直接取节点（测试内导航 schema 用，nil=路径不存在）。
func irTestNodeAt(path string) schema.Node {
	s, err := yangschema.Load()
	if err != nil {
		return nil
	}
	n, _ := s.Path(path)
	return n
}
