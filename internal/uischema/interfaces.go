package uischema

const (
	// InterfacesWidgetID is the ID of the interfaces table widget
	InterfacesWidgetID = "interfaces-table"
	// InterfacesTargetPath is the target path for interfaces configuration
	InterfacesTargetPath = "/ifm:ifm/ifm:interfaces"
	// InterfacesModuleName is the YANG module name for interfaces
	InterfacesModuleName = "huawei-ifm"
	// InterfacesSchemaVersion is the schema version for interfaces
	InterfacesSchemaVersion = SchemaVersion
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
func (g *InterfacesGenerator) BuildSchema(deviceID string) *UISchema {
	return &UISchema{
		SchemaVersion:  InterfacesSchemaVersion,
		Module:         InterfacesModuleName,
		TargetPath:     InterfacesTargetPath,
		CapabilitySource: InterfacesCapabilitySource,
		Layout: Layout{
			Type:    "grid",
			Columns: 12,
			Gap:     "md",
		},
		Sections: []Section{
			{
				ID:      "interfaces-section",
				Title:   "Interfaces",
				Widgets: []string{InterfacesWidgetID},
			},
		},
		Widgets: []Widget{
			{
				ID:      InterfacesWidgetID,
				Type:    WidgetTable,
				Title:   "Interfaces",
				RowKey:  "name",
				Columns: []WidgetColumn{
					{
						Key:        "name",
						Label:      "Interface Name",
						Sortable:   true,
						Filterable: true,
					},
					{
						Key:        "description",
						Label:      "Description",
						Sortable:   true,
						Filterable: true,
					},
					{
						Key:        "mtu",
						Label:      "MTU",
						Type:       "number",
						Align:      "right",
						Sortable:   true,
					},
					{
						Key:        "admin-status",
						Label:      "Admin Status",
						Type:       "badge",
						Sortable:   true,
						Filterable: true,
					},
					{
						Key:        "oper-status",
						Label:      "Operational Status",
						Type:       "badge",
						Sortable:   true,
						Filterable: true,
					},
				},
				Values: make(map[string][]interface{}),
			},
		},
	}
}