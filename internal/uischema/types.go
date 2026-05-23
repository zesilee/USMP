package uischema

// SchemaVersion is the current version of the UI schema
const SchemaVersion = "interfaces:v1"

// WidgetType defines the type of UI widget
type WidgetType string

const (
	// WidgetTable represents a table widget
	WidgetTable WidgetType = "table"
	// WidgetForm represents a form widget
	WidgetForm WidgetType = "form"
	// WidgetCard represents a card widget
	WidgetCard WidgetType = "card"
)

// Layout defines the grid layout configuration
type Layout struct {
	Type    string `json:"type"`
	Columns int    `json:"columns"`
	Gap     string `json:"gap,omitempty"`
}

// Section defines a UI section containing widgets
type Section struct {
	ID      string   `json:"id"`
	Title   string   `json:"title,omitempty"`
	Widgets []string `json:"widgets"`
}

// Validation defines validation rules for form fields
type Validation struct {
	Required bool              `json:"required,omitempty"`
	Min      *float64          `json:"min,omitempty"`
	Max      *float64          `json:"max,omitempty"`
	Pattern  string            `json:"pattern,omitempty"`
	Custom   map[string]string `json:"custom,omitempty"`
}

// Widget defines the base widget interface
type Widget struct {
	ID       string                 `json:"id"`
	Type     WidgetType             `json:"type"`
	Title    string                 `json:"title,omitempty"`
	RowKey   string                 `json:"rowKey,omitempty"`
	Columns  []WidgetColumn         `json:"columns,omitempty"`
	Values   map[string][]interface{} `json:"values,omitempty"`
	Props    map[string]interface{} `json:"props,omitempty"`
	Validation *Validation         `json:"validation,omitempty"`
}

// WidgetColumn defines a column in a table or list widget
type WidgetColumn struct {
	Key        string `json:"key"`
	Label      string `json:"label"`
	Type       string `json:"type,omitempty"`
	Format     string `json:"format,omitempty"`
	Width      string `json:"width,omitempty"`
	Align      string `json:"align,omitempty"`
	Sortable   bool   `json:"sortable,omitempty"`
	Filterable bool   `json:"filterable,omitempty"`
}

// ApplyRequest defines a request to apply configuration
type ApplyRequest struct {
	DeviceID     string                 `json:"deviceId"`
	TargetPath   string                 `json:"targetPath"`
	Values       map[string]interface{} `json:"values"`
	SchemaVersion string                `json:"schemaVersion"`
}

// ApplyResponse defines the response from applying configuration
type ApplyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// UISchema defines the complete UI schema for a device module
type UISchema struct {
	SchemaVersion  string            `json:"schemaVersion"`
	Module         string            `json:"module"`
	TargetPath     string            `json:"targetPath"`
	CapabilitySource string         `json:"capabilitySource"`
	Layout         Layout            `json:"layout"`
	Sections       []Section         `json:"sections"`
	Widgets        []Widget          `json:"widgets"`
}