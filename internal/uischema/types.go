package uischema

import "time"

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

// WidgetGrid defines grid layout positioning for widgets
type WidgetGrid struct {
	Span  int `json:"span,omitempty"`
	Offset int `json:"offset,omitempty"`
	Order int `json:"order,omitempty"`
}

// GridOption defines an option for select widgets
type GridOption struct {
	Label string `json:"label"`
	Value interface{} `json:"value"`
}

// GridValidation defines validation rules for form fields
type GridValidation struct {
	Required    *bool   `json:"required,omitempty"`
	Min         *int    `json:"min,omitempty"`
	Max         *int    `json:"max,omitempty"`
	MinLength   *int    `json:"minLength,omitempty"`
	MaxLength   *int    `json:"maxLength,omitempty"`
	Pattern     string  `json:"pattern,omitempty"`
}

// GridColumn defines a column in a table or form widget
type GridColumn struct {
	ID          string          `json:"id"`
	Type        string          `json:"type,omitempty"`
	Label       string          `json:"label"`
	Placeholder string          `json:"placeholder,omitempty"`
	Readonly    bool            `json:"readonly,omitempty"`
	Options     []GridOption    `json:"options,omitempty"`
	Validation  *GridValidation `json:"validation,omitempty"`
}

// GridWidget defines a widget in the grid layout
type GridWidget struct {
	ID             string                 `json:"id"`
	Type           WidgetType             `json:"type"`
	Label          string                 `json:"label,omitempty"`
	Help           string                 `json:"help,omitempty"`
	RowKey         string                 `json:"rowKey,omitempty"`
	Grid           *WidgetGrid            `json:"grid,omitempty"`
	Columns        []GridColumn           `json:"columns,omitempty"`
	Binding        map[string]interface{} `json:"binding,omitempty"`
	Disabled       bool                   `json:"disabled,omitempty"`
	DisabledReason string                 `json:"disabledReason,omitempty"`
}

// GridSection defines a section in the grid layout
type GridSection struct {
	ID          string        `json:"id"`
	Title       string        `json:"title,omitempty"`
	Description string        `json:"description,omitempty"`
	Widgets     []string      `json:"widgets"`
}

// GridLayout defines the grid layout configuration
type GridLayout struct {
	Type    string `json:"type"`
	Columns int    `json:"columns"`
	Gap     string `json:"gap,omitempty"`
}

// GridSchema defines the complete UI schema for a device module
type GridSchema struct {
	SchemaVersion     string            `json:"schemaVersion"`
	Module            string            `json:"module"`
	TargetPath        string            `json:"targetPath"`
	CapabilitySource  string            `json:"capabilitySource"`
	Layout            GridLayout        `json:"layout"`
	Sections          []GridSection     `json:"sections"`
	Widgets           []GridWidget      `json:"widgets"`
	Values            map[string]interface{} `json:"values,omitempty"`
}

// FieldError defines an error for a specific field
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationError defines an error for validation failures
type ValidationError struct {
	FieldErrors []FieldError `json:"fieldErrors"`
	Message     string       `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Message
}

// ApplyRequest defines a request to apply configuration
type ApplyRequest struct {
	DeviceID        string                 `json:"deviceId"`
	TargetPath      string                 `json:"targetPath"`
	Values          map[string]interface{} `json:"values"`
	SchemaVersion   string                 `json:"schemaVersion"`
	Timestamp       time.Time              `json:"timestamp"`
}

// ApplyResult defines the result from applying configuration
type ApplyResult struct {
	Success      bool              `json:"success"`
	Message      string            `json:"message,omitempty"`
	Error        string            `json:"error,omitempty"`
	Timestamp    time.Time         `json:"timestamp"`
}

