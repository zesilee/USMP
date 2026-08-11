package yangschema

import (
	_ "embed"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// schema.ir.gz 是构建期 schemagen 产出的框架 Schema 树自有序列化（YN-03）：
// 运行期零 ygot/goyang 依赖的加载数据源。重新生成：
//
//	go generate ./internal/yangschema  （或 make gen-yang，阶段2.5 接线）
//
// 与旧链路（generated 包 gzip schema → goyang Entry → AddYgotSchema 转换）的
// 等价性由 ir_parity_test.go 逐字节对拍保证；对拍同时兼作 blob 新鲜度门禁
// （generated schema 变更而未重跑 schemagen 时测试即红）。
//
//go:generate go run ../../tools/schemagen -output=./schema.ir.gz
//go:embed schema.ir.gz
var schemaIRBlob []byte

// loadFromIR decodes the checked-in IR blob into the framework schema tree.
// 阶段1.4 切换 Load() 数据源到本函数；切换后旧链路仅存活于对拍测试。
func loadFromIR() (schema.Schema, error) {
	return schema.DecodeIR(schemaIRBlob)
}
