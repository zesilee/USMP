package uischema

const (
	// InterfacesWidgetID is the ID of the interfaces table widget
	InterfacesWidgetID = "interfaces-table"
	// InterfacesTargetPath is the target path for interfaces configuration
	InterfacesTargetPath = "/ifm:ifm/ifm:interfaces"
	// InterfacesModuleName is the YANG module name for interfaces
	InterfacesModuleName = "huawei-ifm"
	// InterfacesSchemaVersion is the schema version for interfaces
	InterfacesSchemaVersion = "interfaces:v1"
	// InterfacesCapabilitySource is the capability source for interfaces
	InterfacesCapabilitySource = "module-set"
)

// InterfacesGenerator generates UI schema for Huawei IFM interfaces
type InterfacesGenerator struct{}

// NewInterfacesGenerator creates a new InterfacesGenerator
func NewInterfacesGenerator() *InterfacesGenerator {
	return &InterfacesGenerator{}
}

// BuildSchema builds the complete UI schema for interfaces
func (g *InterfacesGenerator) BuildSchema(deviceID string) *GridSchema {
	// Create required boolean pointers for validation
	requiredTrue := true
	min1280 := 1280
	max9216 := 9216
	maxLength80 := 80

	return &GridSchema{
		SchemaVersion:    InterfacesSchemaVersion,
		Module:           InterfacesModuleName,
		TargetPath:       InterfacesTargetPath,
		CapabilitySource: InterfacesCapabilitySource,
		Layout: GridLayout{
			Type:    "grid",
			Columns: 12,
			Gap:     "md",
		},
		Sections: []GridSection{
			{
				ID:          "interfaces",
				Title:       "接口配置",
				Description: "管理设备接口基础配置",
				Widgets:     []string{InterfacesWidgetID},
			},
		},
		Widgets: []GridWidget{
			{
				ID:      InterfacesWidgetID,
				Type:    WidgetTable,
				Label:   "接口列表",
				RowKey:  "name",
				Grid: &WidgetGrid{
					Span: 12,
				},
				Columns: []GridColumn{
					{
						ID:          "name",
						Type:        "text",
						Label:       "接口名称",
						Readonly:    true,
						Validation: &GridValidation{
							Required: &requiredTrue,
						},
					},
					{
						ID:          "description",
						Type:        "text",
						Label:       "描述",
						Validation: &GridValidation{
							MaxLength: &maxLength80,
						},
					},
					{
						ID:          "mtu",
						Type:        "number",
						Label:       "MTU",
						Validation: &GridValidation{
							Min: &min1280,
							Max: &max9216,
						},
					},
					{
						ID:          "admin-status",
						Type:        "select",
						Label:       "管理状态",
						Options: []GridOption{
							{Label: "启用", Value: 1},
							{Label: "禁用", Value: 0},
						},
					},
				},
				Binding: map[string]interface{}{
					"targetPath": InterfacesTargetPath,
				},
			},
		},
		Values: map[string]interface{}{
			InterfacesWidgetID: []interface{}{},
		},
	}
}