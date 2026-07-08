// Package yangschema builds the framework YANG schema tree from the generated
// ygot models (huawei + openconfig). This is the offline fallback source for the
// hybrid schema resolution: device NETCONF capabilities narrow the usable module
// set at runtime, while attribute-level schema comes from these generated models
// (R04: schema derived from ygot-generated models, not hand-written).
package yangschema

import (
	"fmt"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/internal/generated/openconfig"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// Load builds a Schema containing the huawei and openconfig modules from their
// generated ygot schemas.
func Load() (schema.Schema, error) {
	ds := schema.NewSchema()

	hs, err := huawei.Schema()
	if err != nil {
		return nil, fmt.Errorf("load huawei schema: %w", err)
	}
	schema.AddYgotSchemaWithVendor(ds, hs, "huawei")

	os, err := openconfig.Schema()
	if err != nil {
		return nil, fmt.Errorf("load openconfig schema: %w", err)
	}
	schema.AddYgotSchemaWithVendor(ds, os, "openconfig")

	return ds, nil
}

// task-name 任务域映射（BR-01 category）：模块级扩展不存活于内嵌运行期 schema
//（全树扫描实证=0，ygot 生成丢弃模块级语句），构建期从与 ygot 生成集同源同版本的
// 模型提取，生成物提交入库（运行期零 submodule 依赖）。升级模型版本时一并重跑。
//go:generate go run ../../tools/tasknamegen -path=../../../yang-models/network-router/8.20.10/ne40e-x8x16 -modules=huawei-vlan,huawei-ifm,huawei-system -output=./taskname.gen.go -package=yangschema

// Category returns the task domain of a module (keyed by root container name,
// the same key Load exposes as the module name), "" when unmapped. Values come
// from the build-time tasknamegen map (taskname.gen.go).
func Category(module string) string {
	return TaskNames[module]
}
