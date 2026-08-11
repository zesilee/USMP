package main

import (
	"os/exec"
	"strings"
	"testing"
)

// 守护测试（retire-ygot-runtime YN-05/SC-07，仿 NC-01 scrapligo 禁回引）：
// 发布二进制 usmp-backend 的 import 闭包 SHALL NOT 含 openconfig/ygot 与
// openconfig/goyang 任何子包。豁免面=构建期工具（tools/*）、测试文件与
// businessdemo（北向 demo 隔离锚点，任务6.2 拍板随 demo 生命周期退役）——
// 它们不进本闭包，天然不受此断言约束。
//
// 违规修复指引：运行时代码请消费 internal/generated/native/*（自研 yanggen
// 生成，object 接口族）与 pkg/yang-runtime/{object,schema,validate}；
// YANG 解析仅限构建期工具（经 tools/ygotbridge）。
func TestReleaseBinaryFreeOfYgotGoyang(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}
	var offenders []string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "openconfig/ygot") || strings.Contains(line, "openconfig/goyang") {
			offenders = append(offenders, line)
		}
	}
	if len(offenders) > 0 {
		t.Fatalf("发布二进制闭包回引了 ygot/goyang（YN-05/SC-07 红线）：\n%s\n"+
			"运行时禁止依赖外部 YANG 运行库——见本测试头注修复指引。",
			strings.Join(offenders, "\n"))
	}
}
