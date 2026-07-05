package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
)

// YangModuleInfo represents information about a YANG module
type YangModuleInfo struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Vendor      string `json:"vendor"`
	Path        string `json:"path"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

// FieldDef represents a schema field definition for dynamic forms
type FieldDef struct {
	Path        string      `json:"path"`
	Type        string      `json:"type"`
	Label       string      `json:"label"`
	Placeholder string      `json:"placeholder,omitempty"`
	Required    bool        `json:"required,omitempty"`
	Pattern     string      `json:"pattern,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Options     []Option    `json:"options,omitempty"`
	Group       string      `json:"group,omitempty"`
	Minimum     int         `json:"minimum,omitempty"`
	Maximum     int         `json:"maximum,omitempty"`
	Readonly    bool        `json:"readonly,omitempty"`
}

// Option represents a select option
type Option struct {
	Label string      `json:"label"`
	Value interface{} `json:"value"`
}

// YangSchema represents a YANG module schema for dynamic form rendering
type YangSchema struct {
	Module   string     `json:"module"`
	Title    string     `json:"title"`
	Vendor   string     `json:"vendor"`
	Fields   []FieldDef `json:"fields"`
	ListCols []FieldDef `json:"listCols,omitempty"`
}

// YangHandler handles YANG model API requests
type YangHandler struct {
	manager manager.Manager
}

// NewYangHandler creates a new YangHandler
func NewYangHandler(manager manager.Manager) *YangHandler {
	return &YangHandler{
		manager: manager,
	}
}

// ListModules lists all supported YANG modules
//
// @Summary  列出所有支持的 YANG 模块
// @Tags     yang
// @Produce  json
// @Success  200 {object} Response{data=[]YangModuleInfo} "模块列表"
// @Router   /yang/modules [get]
func (h *YangHandler) ListModules(c *gin.Context) {
	s := h.manager.GetSchema()
	modules := make([]YangModuleInfo, 0)

	for _, mod := range s.Modules() {
		root := mod.Root()
		if root == nil {
			continue
		}

		vendor := mod.Vendor()
		if vendor == "" {
			vendor = vendorForNamespace(mod.Namespace())
		}
		info := YangModuleInfo{
			Name:        mod.Name(),
			Title:       root.Name(),
			Vendor:      vendor,
			Path:        "/" + root.Name(),
			Description: root.Description(),
			Type:        strconv.Itoa(int(root.Type())),
		}
		modules = append(modules, info)
	}

	Success(c, modules, "YANG modules retrieved successfully")
}

// GetSchema returns dynamic form schema for a specific YANG module. Modules
// present in the loaded schema tree are rendered dynamically; a hard-coded
// fallback (legacy aliases) is retained during migration (removed in task 2.5).
//
// @Summary  获取指定 YANG 模块的动态表单 schema
// @Tags     yang
// @Produce  json
// @Param    module path string true "模块名"
// @Success  200 {object} Response{data=YangSchema} "动态表单 schema"
// @Router   /yang/schema/{module} [get]
func (h *YangHandler) GetSchema(c *gin.Context) {
	module := c.Param("module")

	if mod, ok := h.manager.GetSchema().Module(module); ok {
		Success(c, buildYangSchema(mod), "Schema retrieved successfully")
		return
	}

	// Legacy hard-coded fallback for aliases not present in the schema tree.
	var schema YangSchema

	switch module {
	case "huawei-ifm", "Interfaces":
		schema = YangSchema{
			Module: module,
			Title:  "华为接口管理",
			Vendor: "huawei",
			Fields: []FieldDef{
				{Path: "ifName", Type: "string", Label: "接口名称", Placeholder: "例如: GigabitEthernet0/0/1", Required: true, Group: "基本信息"},
				{Path: "description", Type: "string", Label: "描述", Placeholder: "例如: 上行端口", Group: "基本信息"},
				{Path: "adminStatus", Type: "enum", Label: "管理状态", Default: "up", Group: "基本设置", Options: []Option{{Label: "启用", Value: "up"}, {Label: "禁用", Value: "down"}}},
				{Path: "mtu", Type: "number", Label: "MTU", Default: 1500, Minimum: 64, Maximum: 9216, Group: "高级设置"},
				{Path: "speed", Type: "enum", Label: "接口速率", Default: "auto", Group: "高级设置", Options: []Option{{Label: "自动协商", Value: "auto"}, {Label: "10M", Value: "10M"}, {Label: "100M", Value: "100M"}, {Label: "1G", Value: "1G"}}},
			},
			ListCols: []FieldDef{
				{Path: "ifName", Type: "string", Label: "接口名称"},
				{Path: "adminStatus", Type: "string", Label: "状态"},
				{Path: "mtu", Type: "number", Label: "MTU"},
			},
		}
	case "huawei-vlan", "VLANs":
		schema = YangSchema{
			Module: module,
			Title:  "华为 VLAN 配置",
			Vendor: "huawei",
			Fields: []FieldDef{
				{Path: "vlanId", Type: "number", Label: "VLAN ID", Required: true, Minimum: 1, Maximum: 4094, Group: "基本信息"},
				{Path: "vlanName", Type: "string", Label: "VLAN 名称", Placeholder: "例如: VLAN-100", Group: "基本信息"},
				{Path: "description", Type: "string", Label: "描述", Group: "基本信息"},
				{Path: "portList", Type: "string", Label: "端口列表", Placeholder: "例如: GigabitEthernet0/0/1,GigabitEthernet0/0/2", Group: "端口配置"},
			},
			ListCols: []FieldDef{
				{Path: "vlanId", Type: "number", Label: "VLAN ID"},
				{Path: "vlanName", Type: "string", Label: "VLAN 名称"},
			},
		}
	default:
		schema = YangSchema{
			Module: module,
			Title:  module,
			Vendor: "huawei",
			Fields: []FieldDef{
				{Path: "name", Type: "string", Label: "名称", Required: true, Group: "基本信息"},
				{Path: "description", Type: "string", Label: "描述", Group: "基本信息"},
			},
		}
	}

	Success(c, schema, "Schema retrieved successfully")
}
