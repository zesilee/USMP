// Package yangschema builds the framework YANG schema tree from the generated
// ygot models (huawei + usmp business intent; BR-11 厂商边界). This is the offline
// fallback source for the hybrid schema resolution: device NETCONF capabilities
// narrow the usable module set at runtime, while attribute-level schema comes
// from these generated models (R04: schema derived from ygot-generated models,
// not hand-written).
package yangschema

import (
	"sync"

	"fmt"

	"github.com/leezesi/usmp/backend/internal/generated/business"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// Load builds a Schema containing the huawei and usmp business intent modules
// from their generated ygot schemas.
//
// 结果记忆化（full-yang-onboarding）：生成闭包扩到 67 根容器后，单次 gzip
// schema 解包成本显著；schema 构建是纯函数、产物只读（DefaultSchema 并发安全），
// 进程内共享一份。生产启动本就单次调用，收益主要在测试面（每包数十次 Load）。
func Load() (schema.Schema, error) {
	loadOnce.Do(func() { loadedSchema, loadErr = loadUncached() })
	return loadedSchema, loadErr
}

var (
	loadOnce     sync.Once
	loadedSchema schema.Schema
	loadErr      error
)

func loadUncached() (schema.Schema, error) {
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
//go:generate go run ../../tools/tasknamegen -path=../../../snd/ce6866p-yang -modules=huawei-vlan,huawei-ifm,huawei-system,huawei-pub-type,huawei-extension,huawei-bgp,huawei-network-instance,huawei-analysis-collector,huawei-anyflow,huawei-arp,huawei-bd,huawei-cfg,huawei-devm,huawei-driver,huawei-dsa,huawei-ecc,huawei-evpn,huawei-fib,huawei-ftpc,huawei-grpc,huawei-hwtacacs,huawei-ifm-trunk,huawei-ip,huawei-l3-multicast,huawei-lacp,huawei-license,huawei-lldp,huawei-loadbalance,huawei-m-lag,huawei-mac-flapping-detect,huawei-macsec,huawei-microsegmentation,huawei-mirror,huawei-monitor-link,huawei-mstp,huawei-multicast,huawei-mvpn,huawei-nqa,huawei-ntp,huawei-nvo3,huawei-openflow-agent,huawei-ospfv2,huawei-ospfv3,huawei-packetevent,huawei-qos,huawei-rsa,huawei-sflow,huawei-sm2,huawei-snmp,huawei-syslog,huawei-system-resources-usage,huawei-unicast-forward,huawei-vrrp,huawei-vty,huawei-vxlan-ext,huawei-vxlan-path-detect,openconfig-telemetry -output=./taskname.gen.go -package=yangschema

// SND blacklist（CN-03）：构建期 revision 匹配后生成模块名集合，运行期零 snd 文件依赖。
//go:generate go run ../../tools/blacklistgen -blacklist=../../../snd/ce6866p-yang/blacklist.xml -path=../../../snd/ce6866p-yang -output=./blacklist.gen.go -package=yangschema

// SND left-tree（LT-01）：构建期生成分组树 + 叶子根容器映射，运行期零 snd 文件依赖。
//go:generate go run ../../tools/lefttreegen -tree=../../../snd/webui/template/left-tree.json -path=../../../snd/ce6866p-yang -output=./lefttree.gen.go -package=yangschema

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

// Blacklisted reports whether the module (keyed by root container name, the
// same key Load exposes as the module name) is SND-blacklisted（CN-03 注解语义）。
func Blacklisted(module string) bool {
	return BlacklistedModules[module]
}
