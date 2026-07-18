package api

import (
	"github.com/gin-gonic/gin"

	"github.com/leezesi/usmp/backend/internal/yangschema"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// LeftTreeNodeDTO 是左树节点载荷（LT-02）：分组（Children 非空）或叶子
// （SourceModule 非空）。叶子恒带 available；supported 仅在 ?device= 且协商
// 可得时出现（省略 = unknown ≠ 不支持）。
type LeftTreeNodeDTO struct {
	Zh string `json:"zh"`
	En string `json:"en"`
	// SourceModule 是叶子的 SND 源模块名（如 huawei-vlan）；分组为空。
	SourceModule string `json:"sourceModule,omitempty"`
	// Available 标记叶子模块是否已接入（根容器已加载）；仅叶子携带。
	Available *bool `json:"available,omitempty"`
	// Module 是首个已加载根容器名（前端路由 /module/<module>）；不可用叶省略。
	Module string `json:"module,omitempty"`
	// Supported 标记该设备 hello 能力是否覆盖此模块；仅 ?device= 且协商可得时携带。
	Supported *bool             `json:"supported,omitempty"`
	Children  []LeftTreeNodeDTO `json:"children,omitempty"`
}

// LeftTree serves the SND left tree with per-leaf availability (LT-02).
//
// @Summary  获取 SND 左树（原生配置导航，含可用性标注）
// @Tags     yang
// @Produce  json
// @Param    device query string false "按设备 hello 能力叠加 supported 标注"
// @Success  200 {object} Response{data=[]LeftTreeNodeDTO} "左树"
// @Failure  404 {object} Response "device 未注册（信封）"
// @Router   /yang/left-tree [get]
func (h *YangHandler) LeftTree(c *gin.Context) {
	loaded := map[string]bool{}
	for _, mod := range h.manager.GetSchema().Modules() {
		loaded[mod.Name()] = true
	}

	// ?device=：协商可得时产出该设备支持的根容器集（CN-02 同一启发匹配）。
	var supported map[string]bool
	if deviceID := c.Query("device"); deviceID != "" {
		info, ok := h.manager.GetDeviceStore().Get(deviceID)
		if !ok {
			Error(c, 404, "device not registered: "+deviceID)
			return
		}
		if caps, negotiated := h.deviceCapabilities(info); negotiated {
			supported = map[string]bool{}
			for _, mod := range schema.NarrowModulesByCapabilities(caps, h.manager.GetSchema().Modules()) {
				supported[mod.Name()] = true
			}
		}
	}

	Success(c, convertLeftTree(yangschema.LeftTree, loaded, supported), "left tree retrieved successfully")
}

// convertLeftTree maps the generated tree to DTOs, computing leaf availability
// against the loaded module set（D2：可用性运行期计算，随渐进生成自动变真）。
func convertLeftTree(nodes []yangschema.LeftTreeNode, loaded, supported map[string]bool) []LeftTreeNodeDTO {
	out := make([]LeftTreeNodeDTO, 0, len(nodes))
	for _, n := range nodes {
		dto := LeftTreeNodeDTO{Zh: n.Zh, En: n.En}
		if n.SourceModule != "" {
			dto.SourceModule = n.SourceModule
			avail := false
			for _, rc := range n.RootContainers {
				if loaded[rc] {
					avail = true
					if dto.Module == "" {
						dto.Module = rc // D4：首个已加载根容器
					}
				}
			}
			dto.Available = &avail
			if supported != nil && avail {
				sup := false
				for _, rc := range n.RootContainers {
					if supported[rc] {
						sup = true
					}
				}
				dto.Supported = &sup
			}
		}
		dto.Children = convertLeftTree(n.Children, loaded, supported)
		out = append(out, dto)
	}
	return out
}
