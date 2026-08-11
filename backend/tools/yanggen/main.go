// yanggen — 自研 YANG→Go 生成器（retire-ygot-runtime 阶段2，CG-01 修订版）。
// 结构约定冻结自 ygot 生成物（openspec/changes/retire-ygot-runtime/
// codegen-conventions.md），生成物实现 pkg/yang-runtime/object 接口族、
// 零 ygot/goyang import。emit 层接线后由 make gen-yang 驱动。
package main

import (
	"fmt"
	"os"
)

func main() {
	// CLI 随 emit 层交付（任务2.2 后半）；当前为 model 层脚手架。
	fmt.Fprintln(os.Stderr, "yanggen: emit 层未接线（阶段2 任务2.2 进行中）")
	os.Exit(2)
}
