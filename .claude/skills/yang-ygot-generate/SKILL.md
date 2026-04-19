---
name: yang-ygot-generate
description: 使用openconfig/ygot解析YANG文件，go generate自动生成Go强类型对象，对接NETCONF配置
---
# 技能详情（补全激活时机+核心原则+使用样例）
## 一、激活时机（何时自动触发）
1.  当用户需求包含「YANG模型」「ygot生成」「配置结构体」「XML转换」等关键词时，自动激活。
2.  开发流程中，涉及「YANG文件解析」「配置对象生成」「NETCONF配置序列化/反序列化」时，自动启用。
3.  与 DeviceActor、NETCONF 技能联动，需要将 NETCONF 配置映射为 Go 对象时，自动触发。

## 二、核心原则（底层设计逻辑）
1.  自动化原则：通过 go generate 脚本实现 YANG 到 Go 结构体的自动生成，减少手动编写成本，避免人为错误。
2.  强类型原则：生成的结构体严格对应 YANG 模型的类型约束，杜绝泛型滥用，确保配置校验安全。
3.  无损转换原则：ygot 对象与 NETCONF XML 报文双向转换，不丢失任何配置字段、属性。
4.  标准化原则：遵循 OpenConfig YANG 规范，确保与不同品牌交换机（华为/H3C/思科）的 YANG 模型兼容。

## 三、使用样例（实操指令+输出效果）
### 样例1：触发技能指令
编写 go generate 脚本，解析 openconfig-interfaces.yang 文件，生成 ygot 结构体


### 样例2：技能输出效果（符合规则，≤500行）
```go
//go:generate go run github.com/openconfig/ygot/generator -path=./yang -output_file=./yang_models/iface.go -package=yang_models openconfig-interfaces.yang

// 生成后自动生成的结构体（简化版）
package yang_models

import (
	"github.com/openconfig/ygot/ygot"
)

// 对应YANG中的 container interfaces
type Interfaces struct {
	Interface map[string]*Interface `path:"interface" ygot:"key=name"`
}

// 对应YANG中的 list interface
type Interface struct {
	Name        string `path:"name" ygot:"key"`
	Description string `path:"description"`
	Enabled     *bool  `path:"enabled"`
	Mtu         *uint16 `path:"mtu"`
}

// 序列化示例：ygot对象转NETCONF XML
func (i *Interface) ToXML() (string, error) {
	xmlBytes, err := ygot.EmitXML(i, nil)
	if err != nil {
		return "", err
	}
	return string(xmlBytes), nil
}

// 反序列化示例：NETCONF XML转ygot对象
func XMLToInterface(xmlStr string) (*Interface, error) {
	var iface Interface
	if err := ygot.Unmarshal([]byte(xmlStr), &iface); err != nil {
		return nil, err
	}
	return &iface, nil
}
```