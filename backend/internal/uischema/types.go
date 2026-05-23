package uischema

// WidgetType defines the type of UI widget
type WidgetType string

const (
	// WidgetText represents a text input widget
	WidgetText WidgetType = "text"
	// WidgetNumber represents a number input widget
	WidgetNumber WidgetType = "number"
	// WidgetSelect represents a select dropdown widget
	WidgetSelect WidgetType = "select"
	// WidgetSwitch represents a switch/toggle widget
	WidgetSwitch WidgetType = "switch"
	// WidgetTextarea represents a textarea widget
	WidgetTextarea WidgetType = "textarea"
	// WidgetTable represents a table widget
	WidgetTable WidgetType = "table"
)

// GridSchema defines the complete UI schema for a device module
type GridSchema struct {
	SchemaVersion    string                 `json:"schemaVersion"`
	Module           string                 `json:"module"`
	TargetPath       string                 `json:"targetPath"`
	CapabilitySource string                 `json:"capabilitySource"`
	Layout           GridLayout             `json:"layout"`
	Sections         []GridSection          `json:"sections"`
	Widgets          []GridWidget           `json:"widgets"`
	Values           map[string]interface{} `json:"values"`
}

// GridLayout defines the grid layout configuration
type GridLayout struct {
	Type    string `json:"type"`
	Columns int    `json:"columns"`
	Gap     string `json:"gap"`
}

// GridSection defines a section in the grid layout
type GridSection struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Widgets     []string `json:"widgets"`
}

// GridWidget defines a widget in the grid layout
type GridWidget struct {
	ID             string                 `json:"id"`
	Type           WidgetType             `json:"type"`
	Label          string                 `json:"label"`
	Help           string                 `json:"help,omitempty"`
	RowKey         string                 `json:"rowKey,omitempty"`
	Grid           WidgetGrid             `json:"grid"`
	Columns        []GridColumn           `json:"columns,omitempty"`
	Binding        map[string]interface{} `json:"binding,omitempty"`
	Disabled       bool                   `json:"disabled,omitempty"`
	DisabledReason string                 `json:"disabledReason,omitempty"`
}

// WidgetGrid defines grid layout positioning for widgets
type WidgetGrid struct {
	Span   int `json:"span"`
	Offset int `json:"offset,omitempty"`
	Order  int `json:"order,omitempty"`
}

// GridColumn defines a column in a table or form widget
type GridColumn struct {
	ID          string         `json:"id"`
	Type        WidgetType     `json:"type"`
	Label       string         `json:"label"`
	Placeholder string         `json:"placeholder,omitempty"`
	Readonly    bool           `json:"readonly,omitempty"`
	Options     []GridOption   `json:"options,omitempty"`
	Validation  GridValidation `json:"validation,omitempty"`
}

// GridOption defines an option for select widgets
type GridOption struct {
	Label string      `json:"label"`
	Value interface{} `json:"value"`
}

// GridValidation defines validation rules for form fields
type GridValidation struct {
	Required  bool `json:"required,omitempty"`
	Min       *int `json:"min,omitempty"`
	Max       *int `json:"max,omitempty"`
	MinLength *int `json:"minLength,omitempty"`
	MaxLength *int `json:"maxLength,omitempty"`
}

// ApplyRequest defines a request to apply configuration
type ApplyRequest struct {
	SchemaVersion string                 `json:"schemaVersion"`
	Values        map[string]interface{} `json:"values"`
}

// ApplyResult defines the result from applying configuration
type ApplyResult struct {
	SchemaVersion string                 `json:"schemaVersion"`
	Values        map[string]interface{} `json:"values,omitempty"`
	LastSync      string                 `json:"lastSync,omitempty"`
}

// FieldError defines an error for a specific field
type FieldError struct {
	Field    string   `json:"field"`
	Messages []string `json:"messages"`
}

// ValidationError defines an error for validation failures
type ValidationError struct {
	Code        string              `json:"code"`
	Message     string              `json:"message"`
	FieldErrors map[string][]string `json:"fieldErrors,omitempty"`
}

func (e *ValidationError) Error() string {
	return e.Message
}
