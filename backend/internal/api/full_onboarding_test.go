package api

import (
	"testing"

	"github.com/gin-gonic/gin"

	yangdriver "github.com/leezesi/usmp/backend/pkg/yang-runtime/driver"
)

// TestLeftTreeFullAvailabilityBaseline（LT-04）：左树可用叶 = 全部叶 −
// 延期项 huawei-pic（CG-04）− 4 个 augment-only 叶（无自有根容器，内容并入
// 宿主模块树呈现，无独立控制台语义）。可用集合缩水（回归）或清单外新增
// 不可用叶都在此红灯。
func TestLeftTreeFullAvailabilityBaseline(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newYangHandlerWithSchema(t)
	w := getLeftTree(t, h, "")
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	var tree []LeftTreeNodeDTO
	decodeData(t, w.Body.Bytes(), &tree)

	deferred := map[string]bool{
		"huawei-pic": true, // 延期：goyang 跨模块 submodule typedef 解析死结
		// augment-only（无自有根容器，内容并入宿主树；ethernet/gre/nvo3-statistics/ip
		// 均已入生成闭包，其 augment 面在宿主模块控制台呈现）：
		"huawei-ethernet":        true,
		"huawei-ip":              true,
		"huawei-gre":             true,
		"huawei-nvo3-statistics": true,
	}
	var walk func(nodes []LeftTreeNodeDTO)
	total, available := 0, 0
	walk = func(nodes []LeftTreeNodeDTO) {
		for _, n := range nodes {
			if len(n.Children) > 0 {
				walk(n.Children)
				continue
			}
			total++
			ok := n.Available != nil && *n.Available
			if ok {
				available++
				if n.Module == "" {
					t.Errorf("可用叶 %s 缺 module（不可路由）", n.SourceModule)
				}
			}
			if deferred[n.SourceModule] && ok {
				t.Errorf("延期叶 %s 不应可用（清单漂移，需同步 spec）", n.SourceModule)
			}
			if !deferred[n.SourceModule] && !ok {
				t.Errorf("叶 %s 应可用（全量基线缩水）", n.SourceModule)
			}
		}
	}
	walk(tree)
	if available != total-len(deferred) {
		t.Fatalf("可用叶 %d/%d, want %d（全部−延期）", available, total, total-len(deferred))
	}
}

// TestConvertConfigAllModuleAnchors（B3 参数化）：每个华为 XML 模块的锚点路径
// 经 convertConfig 编包成功（空子树 → 空容器合法）；未注册路径显式报错（负路径）。
func TestConvertConfigAllModuleAnchors(t *testing.T) {
	n := 0
	for _, d := range yangdriver.All() {
		if d.Vendor != "huawei" || d.XML == nil {
			continue
		}
		n++
		if _, err := convertConfig(d.EncodeAnchor, map[string]interface{}{}); err != nil {
			t.Errorf("convertConfig(%q) 编包失败: %v", d.EncodeAnchor, err)
		}
	}
	// 57 表行 + 4 个带 XML 的手写块（vlan/ifm/bgp/network-instance；system 无 XML）
	if n < 61 {
		t.Fatalf("华为 XML 模块描述符 %d 个, want ≥61（表/手写块缩水）", n)
	}
	if _, err := convertConfig("/no-such:module", map[string]interface{}{}); err == nil {
		t.Error("未注册路径应显式报错")
	}
}
