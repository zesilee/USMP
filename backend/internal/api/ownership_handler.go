package api

import (
	"github.com/gin-gonic/gin"

	"github.com/leezesi/usmp/backend/internal/intent"
)

// OwnershipHandler serves the soft-ownership query surface (BIO-07)：前端
// 原生控制台徽标（FE-18）与手改提示的数据面。
type OwnershipHandler struct {
	index *intent.OwnershipIndex
}

// NewOwnershipHandler builds the handler over the process-wide index.
func NewOwnershipHandler() *OwnershipHandler {
	return &OwnershipHandler{index: intent.DefaultOwnership}
}

// OwnershipClaim is one claimed native entry on a device.
type OwnershipClaim struct {
	// Intent 认领方（意图 CR 的 namespace/name）。
	Intent string `json:"intent"`
	Module string `json:"module"`
	Path   string `json:"path"`
}

// OwnershipData is the query response payload.
type OwnershipData struct {
	Device string `json:"device"`
	// Path 为空时返回该设备全部认领。
	Path string `json:"path,omitempty"`
	// Intents 命中 path 的认领意图（path 查询模式）。
	Intents []string `json:"intents,omitempty"`
	// Claims 设备全量认领（无 path 查询模式）。
	Claims []OwnershipClaim `json:"claims,omitempty"`
}

// Query returns ownership info for a device (optionally narrowed to a path).
//
// @Summary  查询设备原生配置的业务意图归属（软归属，BIO-07）
// @Tags     ownership
// @Produce  json
// @Param    device path  string true  "设备 IP"
// @Param    path   query string false "原生 YANG 路径（前缀匹配；缺省返回设备全量认领）"
// @Success  200 {object} Response{data=OwnershipData} "归属信息（未认领时 intents/claims 为空）"
// @Failure  400 {object} Response "缺少设备参数"
// @Router   /ownership/{device} [get]
func (h *OwnershipHandler) Query(c *gin.Context) {
	device := c.Param("device")
	if device == "" {
		Error(c, 400, "missing device")
		return
	}
	path := c.Query("path")
	data := OwnershipData{Device: device, Path: path}
	if path != "" {
		data.Intents = h.index.Owners(device, path)
	} else {
		for intentKey, claims := range h.index.Claims(device) {
			for _, cl := range claims {
				data.Claims = append(data.Claims, OwnershipClaim{Intent: intentKey, Module: cl.Module, Path: cl.Path})
			}
		}
	}
	Success(c, data, "ownership resolved")
}
