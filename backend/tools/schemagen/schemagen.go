// schemagen（retire-ygot-runtime 阶段1，YN-03）：构建期把框架内部 Schema 树
// 序列化为自有 IR 格式（schema.EncodeIR）入库，运行期仅 DecodeIR 加载——goyang
// Entry/ygot gzip schema 由此退出发布二进制。
//
// 一期输入源刻意复用现运行链路（generated 包 Schema() → AddYgotSchemaWithVendor，
// 与 internal/yangschema.loadUncached 同构）：构建产物与今日运行期树逐字节同构，
// 对拍（EncodeIR(旧链路树) == 入库 blob）零漂移。二期自研生成器落地后本工具切换为
// 直读 YANG 源，届时同一对拍门禁兜住行为差异。
package main

import (
	"fmt"
	"os"

	"github.com/leezesi/usmp/backend/internal/generated/business"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// BuildIR 构建与运行期 yangschema.Load() 同构的 Schema 树并编码为 IR blob。
func BuildIR() ([]byte, error) {
	ds := schema.NewSchema()

	hs, err := huawei.Schema()
	if err != nil {
		return nil, fmt.Errorf("schemagen: load huawei schema: %w", err)
	}
	schema.AddYgotSchemaWithVendor(ds, hs, "huawei")

	bs, err := business.Schema()
	if err != nil {
		return nil, fmt.Errorf("schemagen: load business schema: %w", err)
	}
	schema.AddYgotSchemaWithVendor(ds, bs, "usmp")

	if len(ds.Modules()) == 0 {
		return nil, fmt.Errorf("schemagen: built schema has no modules")
	}
	return schema.EncodeIR(ds)
}

// Run 生成 IR blob 写入 outputPath（原子性：先临时文件后 rename 不做——生成物
// 入库走 git diff 门禁，半成品会显形；写失败返回明确错误即可）。
func Run(outputPath string) error {
	blob, err := BuildIR()
	if err != nil {
		return err
	}
	if err := os.WriteFile(outputPath, blob, 0o644); err != nil {
		return fmt.Errorf("schemagen: write %s: %w", outputPath, err)
	}
	return nil
}
