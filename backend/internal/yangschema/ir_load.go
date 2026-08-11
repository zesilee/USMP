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
// 新鲜度门禁在 tools/schemagen 的 TestSourceVsBlobCompare（blob 与直读源
// 重建逐字节比对；S4 起 schemagen 直读 YANG 源，旧 gzip 链路已退役）。
//
//go:generate go run ../../tools/schemagen -repo_root=../../.. -output=./schema.ir.gz
//go:embed schema.ir.gz
var schemaIRBlob []byte

// loadFromIR decodes the checked-in IR blob into the framework schema tree.
// 阶段1.4 切换 Load() 数据源到本函数；切换后旧链路仅存活于对拍测试。
func loadFromIR() (schema.Schema, error) {
	return schema.DecodeIR(schemaIRBlob)
}
