// schemagen（retire-ygot-runtime，YN-03/S4）：构建期把框架内部 Schema 树
// 序列化为自有 IR 格式（schema.EncodeIR）入库，运行期仅 DecodeIR 加载。
//
// S4 起**直读 YANG 源**（gen.conf 驱动，经 ygotbridge 装载+转换），不再依赖
// generated 包 gzip schema。与旧链路的等价性已实测冻结：叶级全字段零差异；
// 仅两处刻意口径——① description 剥离（gzip 往返历史行为不含描述，冻结字节
// 稳定；描述增益另行拍板）② 模块 namespace 补全（gzip 往返实测恒空的 D3b
// 缺口修复，68 模块获得真实 namespace，采纳为正确性修复）。
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
	"github.com/leezesi/usmp/backend/tools/ygotbridge"
)

// vendorConf 描述一个生成包的 schema 来源与 vendor 标签。
type vendorConf struct {
	conf   string // gen.conf 相对 repoRoot 路径
	vendor string
}

// schemaSources 是运行期 schema 树的组成（与旧 yangschema.loadUncached 口径
// 一致：huawei 厂商包 + usmp 业务意图包；businessdemo 隔离锚点不入树）。
var schemaSources = []vendorConf{
	{conf: "backend/internal/generated/huawei/gen.conf", vendor: "huawei"},
	{conf: "backend/internal/generated/business/gen.conf", vendor: "usmp"},
}

// BuildIR 直读 YANG 源构建与运行期同构的 Schema 树并编码为 IR blob。
func BuildIR(repoRoot string) ([]byte, error) {
	ds := schema.NewSchema()
	for _, vc := range schemaSources {
		conf, err := ParseGenConfAt(filepath.Join(repoRoot, vc.conf), repoRoot)
		if err != nil {
			return nil, err
		}
		entries, err := ygotbridge.LoadModuleEntries(conf.YangPaths, conf.Modules)
		if err != nil {
			return nil, fmt.Errorf("schemagen: %s: %w", vc.vendor, err)
		}
		if err := ygotbridge.AddSourceModules(ds, entries, vc.vendor, true); err != nil {
			return nil, fmt.Errorf("schemagen: %s: %w", vc.vendor, err)
		}
	}
	if len(ds.Modules()) == 0 {
		return nil, fmt.Errorf("schemagen: built schema has no modules")
	}
	return schema.EncodeIR(ds)
}

// Run 生成 IR blob 写入 outputPath（生成物入库走 git diff 门禁，半成品会显形）。
func Run(repoRoot, outputPath string) error {
	blob, err := BuildIR(repoRoot)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outputPath, blob, 0o644); err != nil {
		return fmt.Errorf("schemagen: write %s: %w", outputPath, err)
	}
	return nil
}
