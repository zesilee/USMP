package yangschema

import (
	_ "embed"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// schema.ir.gz 是构建期 schemagen 产出的框架 Schema 树自有序列化（YN-03）：
// 运行期零 ygot/goyang 依赖的加载数据源。二进制产物不入库（R18），克隆后
// 由 make setup 自动生成；编译报 "no matching files found" 时运行：
//
//	make gen-schema-ir  （或 go generate ./internal/yangschema / make gen-yang）
//
// 新鲜度门禁在 tools/schemagen 的 TestSourceVsBlobCompare（blob 与直读源
// 重建逐字节比对；S4 起 schemagen 直读 YANG 源，旧 gzip 链路已退役）。
//
//go:generate go -C ../../tools run ./schemagen -repo_root=../.. -output=../internal/yangschema/schema.ir.gz
//go:embed schema.ir.gz
var schemaIRBlob []byte

// loadFromIR decodes the checked-in IR blob into the framework schema tree.
// 阶段1.4 切换 Load() 数据源到本函数；切换后旧链路仅存活于对拍测试。
func loadFromIR() (schema.Schema, error) {
	return schema.DecodeIR(schemaIRBlob)
}
