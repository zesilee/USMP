package uischema

const (
	// InterfacesWidgetID is the ID of the interfaces table widget
	InterfacesWidgetID = "interfaces-table"
	// InterfacesTargetPath is the target path for interfaces configuration
	InterfacesTargetPath = "/ifm:ifm/ifm:interfaces"
)

// InterfacesGenerator generates UI schema for Huawei IFM interfaces
type InterfacesGenerator struct{}

// NewInterfacesGenerator creates a new InterfacesGenerator
func NewInterfacesGenerator() *InterfacesGenerator {
	return &InterfacesGenerator{}
}

// BuildSchema builds the complete UI schema for interfaces
func (g *InterfacesGenerator) BuildSchema(deviceIP string) GridSchema {
	maxLength80 := 80
	min1280 := 1280
	max9216 := 9216

	return GridSchema{
		SchemaVersion:    "interfaces:v1",
		Module:           "huawei-ifm",
		TargetPath:       InterfacesTargetPath,
		CapabilitySource: "module-set",
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
				ID:     InterfacesWidgetID,
				Type:   WidgetTable,
				Label:  "接口列表",
				RowKey: "name",
				Grid: WidgetGrid{
					Span: 12,
				},
				Columns: []GridColumn{
					{
						ID:       "name",
						Type:     WidgetText,
						Label:    "接口名称",
						Readonly: true,
						Validation: GridValidation{
							Required: true,
						},
					},
					{
						ID:    "description",
						Type:  WidgetText,
						Label: "描述",
						Validation: GridValidation{
							MaxLength: &maxLength80,
						},
					},
					{
						ID:    "mtu",
						Type:  WidgetNumber,
						Label: "MTU",
						Validation: GridValidation{
							Min: &min1280,
							Max: &max9216,
						},
					},
					{
						ID:    "admin-status",
						Type:  WidgetSelect,
						Label: "管理状态",
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
