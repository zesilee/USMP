package drivers

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/driver"
)

// 真机回归（T07）：CE6866 8.20.10 对 edit-config 按模块声明的 namespace 校验，
// 不匹配即 rpc-error unknown-namespace（"vlan has invalid namespace"）。
// 真理源 = 仓库内设备 YANG 包 snd/ce6866p-yang/ 各模块的 namespace 语句：
//   - 描述符模块名能定位到 huawei-<module>.yang 时精确对照；
//   - 否则（openconfig 系等异名文件）退回「namespace 必须在设备包声明全集中」，
//     这正是设备 unknown-namespace 校验的语义。
// 新注册模块自动纳入守护。

const deviceYangDir = "../../../snd/ce6866p-yang"

var yangNamespaceRe = regexp.MustCompile(`namespace\s+"([^"]+)"`)

var loadDeviceNamespaces = sync.OnceValues(func() (map[string]string, error) {
	files, err := filepath.Glob(filepath.Join(deviceYangDir, "*.yang"))
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("%s 下没有 YANG 文件", deviceYangDir)
	}
	// 文件名（去 .yang）→ 声明 namespace；submodule 无 namespace，值为空串。
	out := make(map[string]string, len(files))
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		name := strings.TrimSuffix(filepath.Base(f), ".yang")
		if m := yangNamespaceRe.FindSubmatch(data); m != nil {
			out[name] = string(m[1])
		} else {
			out[name] = ""
		}
	}
	return out, nil
})

func assertDeviceNamespace(t *testing.T, module, got string) {
	t.Helper()
	nsByFile, err := loadDeviceNamespaces()
	if err != nil {
		t.Fatalf("加载设备 YANG 包失败: %v", err)
	}
	if want, ok := nsByFile["huawei-"+module]; ok && want != "" {
		if got != want {
			t.Errorf("模块 %s 注册 namespace = %q，设备声明 = %q（真机将拒绝 unknown-namespace）",
				module, got, want)
		}
		return
	}
	for _, ns := range nsByFile {
		if ns == got {
			return
		}
	}
	t.Errorf("模块 %s 注册 namespace = %q 不在设备 YANG 包声明全集中（真机将拒绝 unknown-namespace）",
		module, got)
}

func TestRegisteredNamespacesMatchDeviceYang(t *testing.T) {
	checked := 0
	for _, d := range driver.All() {
		if d.XML == nil {
			continue // 无 XML 通道（system）不参与 namespace 校验
		}
		t.Run(d.Module, func(t *testing.T) {
			assertDeviceNamespace(t, d.Module, d.XML.Namespace)
			// per-node namespace 映射（XC-06 augment 子树）同样必须匹配设备声明。
			for mod, ns := range d.XML.Namespaces {
				assertDeviceNamespace(t, strings.TrimPrefix(mod, "huawei-"), ns)
			}
		})
		checked++
	}
	if checked == 0 {
		t.Fatal("注册表中没有任何带 XML 编解码的描述符——守护测试失去对象")
	}
}
