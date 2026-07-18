// Package yangschema builds the framework YANG schema tree from the generated
// ygot models (huawei + usmp business intent; BR-11 厂商边界). This is the offline
// fallback source for the hybrid schema resolution: device NETCONF capabilities
// narrow the usable module set at runtime, while attribute-level schema comes
// from these generated models (R04: schema derived from ygot-generated models,
// not hand-written).
package yangschema

import (
	"fmt"

	"github.com/leezesi/usmp/backend/internal/generated/business"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// Load builds a Schema containing the huawei and usmp business intent modules
// from their generated ygot schemas.
func Load() (schema.Schema, error) {
	ds := schema.NewSchema()

	hs, err := huawei.Schema()
	if err != nil {
		return nil, fmt.Errorf("load huawei schema: %w", err)
	}
	schema.AddYgotSchemaWithVendor(ds, hs, "huawei")

	// 业务意图模型（business-network-config）：不注册 driver registry（不下设备），
	// 但进 schema 树以驱动 /yang/modules、/yang/schema 与前端表单渲染（R05）。
	bs, err := business.Schema()
	if err != nil {
		return nil, fmt.Errorf("load business schema: %w", err)
	}
	schema.AddYgotSchemaWithVendor(ds, bs, "usmp")

	return ds, nil
}

// task-name 任务域映射（BR-01 category）：模块级扩展不存活于内嵌运行期 schema
//（全树扫描实证=0，ygot 生成丢弃模块级语句），构建期从与 ygot 生成集同源同版本的
// 模型提取，生成物提交入库（运行期零 submodule 依赖）。升级模型版本时一并重跑。
//go:generate go run ../../tools/tasknamegen -path=../../../yang-models/network-router/8.20.10/ne40e-x8x16 -modules=huawei-vlan,huawei-ifm,huawei-system -output=./taskname.gen.go -package=yangschema

// 业务意图模型的 task-name 与厂商模型不同源（backend/internal/yang/models 入库
// 模型，无 submodule 依赖），独立生成文件与变量避免冲突。
//go:generate go run ../../tools/tasknamegen -path=../yang/models -modules=usmp-business-vlan -output=./taskname_business.gen.go -package=yangschema -var=BusinessTaskNames

// Category returns the task domain of a module (keyed by root container name,
// the same key Load exposes as the module name), "" when unmapped. Values come
// from the build-time tasknamegen maps (taskname*.gen.go); vendor maps first,
// business intent map as fallback (key sets are disjoint by construction).
func Category(module string) string {
	if c, ok := TaskNames[module]; ok {
		return c
	}
	return BusinessTaskNames[module]
}
