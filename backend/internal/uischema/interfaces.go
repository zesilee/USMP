package uischema

import (
	"fmt"
	"math"
)

const (
	// InterfacesWidgetID is the ID of the interfaces table widget
	InterfacesWidgetID = "interfaces-table"
	// InterfacesTargetPath is the target path for interfaces configuration
	InterfacesTargetPath = "/ifm:ifm/ifm:interfaces"
	// InterfaceMTUMin is the minimum allowed MTU value
	InterfaceMTUMin = 1280
	// InterfaceMTUMax is the maximum allowed MTU value
	InterfaceMTUMax = 9216
	// InterfaceDescriptionMaxLength is the maximum allowed length for interface description
	InterfaceDescriptionMaxLength = 80
)

// InterfacesGenerator generates UI schema for Huawei IFM interfaces
type InterfacesGenerator struct{}

// NewInterfacesGenerator creates a new InterfacesGenerator
func NewInterfacesGenerator() *InterfacesGenerator {
	return &InterfacesGenerator{}
}

// BuildSchema builds the complete UI schema for interfaces
func (g *InterfacesGenerator) BuildSchema(deviceIP string) GridSchema {
	descMaxLen := InterfaceDescriptionMaxLength
	mtuMin := InterfaceMTUMin
	mtuMax := InterfaceMTUMax

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
							MaxLength: &descMaxLen,
						},
					},
					{
						ID:    "mtu",
						Type:  WidgetNumber,
						Label: "MTU",
						Validation: GridValidation{
							Min: &mtuMin,
							Max: &mtuMax,
						},
					},
					{
						ID:    "admin-status",
						Type:  WidgetSelect,
						Label: "管理状态",
						Options: []GridOption{
							{Label: "启用", Value: 2},
							{Label: "禁用", Value: 1},
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

// ValidateApply validates an apply request for interfaces
func (g *InterfacesGenerator) ValidateApply(req ApplyRequest) error {
	// Check schema version
	if req.SchemaVersion != "interfaces:v1" {
		return &ValidationError{
			Code:    "SCHEMA_VERSION_MISMATCH",
			Message: "Schema 已更新，请刷新后重试",
		}
	}

	fieldErrors := make(map[string][]string)

	// Get interfaces table
	tableVal, ok := req.Values[InterfacesWidgetID]
	if !ok {
		fieldErrors[InterfacesWidgetID] = []string{"接口列表是必填项"}
		return &ValidationError{
			Code:        "VALIDATION_FAILED",
			Message:     "配置校验失败",
			FieldErrors: fieldErrors,
		}
	}

	rows, ok := tableVal.([]interface{})
	if !ok {
		fieldErrors[InterfacesWidgetID] = []string{"接口列表格式错误"}
		return &ValidationError{
			Code:        "VALIDATION_FAILED",
			Message:     "配置校验失败",
			FieldErrors: fieldErrors,
		}
	}

	// Validate each row
	for _, rowVal := range rows {
		row, ok := rowVal.(map[string]interface{})
		if !ok {
			fieldErrors[InterfacesWidgetID] = append(fieldErrors[InterfacesWidgetID], "接口行格式不正确")
			continue
		}

		// Get row name
		nameVal, nameOk := row["name"]
		name, nameStrOk := nameVal.(string)
		if !nameOk || !nameStrOk || name == "" {
			fieldKey := "interfaces-table:row:unknown:name"
			fieldErrors[fieldKey] = append(fieldErrors[fieldKey], "接口名称是必填项")
			continue
		}

		// Validate MTU
		if mtuVal, ok := row["mtu"]; ok {
			mtu, ok := numberToInt(mtuVal)
			if !ok {
				fieldKey := fmt.Sprintf("interfaces-table:row:%s:mtu", name)
				fieldErrors[fieldKey] = append(fieldErrors[fieldKey], "MTU 必须是数字")
			} else if mtu < InterfaceMTUMin || mtu > InterfaceMTUMax {
				fieldKey := fmt.Sprintf("interfaces-table:row:%s:mtu", name)
				fieldErrors[fieldKey] = append(fieldErrors[fieldKey], fmt.Sprintf("MTU 必须在 %d 到 %d 之间", InterfaceMTUMin, InterfaceMTUMax))
			}
		}

		// Validate description
		if descVal, ok := row["description"]; ok {
			desc, ok := descVal.(string)
			if ok && len(desc) > InterfaceDescriptionMaxLength {
				fieldKey := fmt.Sprintf("interfaces-table:row:%s:description", name)
				fieldErrors[fieldKey] = append(fieldErrors[fieldKey], fmt.Sprintf("描述长度不能超过 %d 个字符", InterfaceDescriptionMaxLength))
			}
		}

		if statusVal, ok := row["admin-status"]; ok {
			status, ok := numberToInt(statusVal)
			if !ok {
				fieldKey := fmt.Sprintf("interfaces-table:row:%s:admin-status", name)
				fieldErrors[fieldKey] = append(fieldErrors[fieldKey], "管理状态必须是数字")
			} else if status != 1 && status != 2 {
				fieldKey := fmt.Sprintf("interfaces-table:row:%s:admin-status", name)
				fieldErrors[fieldKey] = append(fieldErrors[fieldKey], "管理状态必须是启用或禁用")
			}
		}
	}

	if len(fieldErrors) > 0 {
		return &ValidationError{
			Code:        "VALIDATION_FAILED",
			Message:     "配置校验失败",
			FieldErrors: fieldErrors,
		}
	}

	return nil
}

// numberToInt converts various numeric types to int
func numberToInt(v interface{}) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int32:
		return int(val), true
	case int64:
		return int(val), true
	case float64:
		if math.Trunc(val) != val {
			return 0, false
		}
		return int(val), true
	case float32:
		floatVal := float64(val)
		if math.Trunc(floatVal) != floatVal {
			return 0, false
		}
		return int(val), true
	default:
		return 0, false
	}
}
