package actor

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Translator handles mapping between generic payloads and YANG struct fields.
// T is the YANG struct type (e.g., *huawei.HuaweiVlan_Vlan).
type Translator[T YANGGoStruct] interface {
	// Translate maps the payload to the target YANG struct fields.
	Translate(payload map[string]interface{}, target T) error
	// TranslateAtPath maps payload at a specific YANG schema path.
	TranslateAtPath(path string, payload map[string]interface{}, target T) error
	// ToPayload converts the YANG struct back to a generic payload map.
	ToPayload(source T) (map[string]interface{}, error)
}

// ReflectTranslator uses reflection to automatically map payload fields to YANG struct fields.
type ReflectTranslator[T YANGGoStruct] struct {
	// CustomMappings allows custom field converters for specific paths.
	CustomMappings map[string]FieldMappingFunc
	// PathPrefix is an optional prefix to strip from all source paths.
	PathPrefix string
}

// FieldMappingFunc converts a raw value to the target type.
type FieldMappingFunc func(interface{}) (interface{}, error)

// NewReflectTranslator creates a new ReflectTranslator.
func NewReflectTranslator[T YANGGoStruct]() *ReflectTranslator[T] {
	return &ReflectTranslator[T]{
		CustomMappings: make(map[string]FieldMappingFunc),
	}
}

// AddCustomMapping adds a custom field mapping for a specific path.
func (t *ReflectTranslator[T]) AddCustomMapping(path string, fn FieldMappingFunc) {
	t.CustomMappings[path] = fn
}

// Translate implements the Translator interface.
func (t *ReflectTranslator[T]) Translate(payload map[string]interface{}, target T) error {
	targetVal := reflect.ValueOf(target).Elem()
	return t.translateMap("", payload, targetVal)
}

// TranslateAtPath implements the Translator interface.
func (t *ReflectTranslator[T]) TranslateAtPath(path string, payload map[string]interface{}, target T) error {
	// Navigate to the target field at the given path.
	targetVal := reflect.ValueOf(target).Elem()
	fields := strings.Split(path, ".")
	for _, field := range fields {
		if field == "" {
			continue
		}
		targetVal = targetVal.FieldByName(field)
		if !targetVal.IsValid() {
			return fmt.Errorf("field not found at path %s", path)
		}
		if targetVal.Kind() == reflect.Ptr {
			if targetVal.IsNil() {
				targetVal.Set(reflect.New(targetVal.Type().Elem()))
			}
			targetVal = targetVal.Elem()
		}
	}
	return t.translateMap(path, payload, targetVal)
}

// ToPayload converts the YANG struct back to a generic payload map.
func (t *ReflectTranslator[T]) ToPayload(source T) (map[string]interface{}, error) {
	// TODO: Implement reverse mapping from struct to map
	result := make(map[string]interface{})
	return result, nil
}

// translateMap recursively maps payload values to struct fields.
func (t *ReflectTranslator[T]) translateMap(
	basePath string,
	payload map[string]interface{},
	target reflect.Value,
) error {
	for key, value := range payload {
		fieldPath := key
		if basePath != "" {
			fieldPath = basePath + "." + key
		}

		// Check for custom mapping
		if fn, ok := t.CustomMappings[fieldPath]; ok {
			converted, err := fn(value)
			if err != nil {
				return fmt.Errorf("custom mapping failed for %s: %w", fieldPath, err)
			}
			if err := t.setFieldValue(target, key, converted); err != nil {
				return err
			}
			continue
		}

		// Handle nested maps recursively
		if nested, ok := value.(map[string]interface{}); ok {
			fieldVal := t.findField(target, key)
			if !fieldVal.IsValid() {
				continue // Skip unknown fields silently
			}
			if fieldVal.Kind() == reflect.Ptr {
				if fieldVal.IsNil() {
					fieldVal.Set(reflect.New(fieldVal.Type().Elem()))
				}
				fieldVal = fieldVal.Elem()
			}
			if err := t.translateMap(fieldPath, nested, fieldVal); err != nil {
				return err
			}
			continue
		}

		// Handle arrays/slices
		if arr, ok := value.([]interface{}); ok {
			fieldVal := t.findField(target, key)
			if !fieldVal.IsValid() {
				continue
			}
			if err := t.setSliceField(fieldVal, arr); err != nil {
				return err
			}
			continue
		}

		// Set leaf value
		if err := t.setFieldValue(target, key, value); err != nil {
			return err
		}
	}
	return nil
}

// findField finds a struct field by name (case-insensitive).
func (t *ReflectTranslator[T]) findField(target reflect.Value, name string) reflect.Value {
	// Try exact match first
	field := target.FieldByName(name)
	if field.IsValid() {
		return field
	}

	// Try case-insensitive match
	upperName := strings.ToUpper(name)
	targetType := target.Type()
	for i := 0; i < targetType.NumField(); i++ {
		fieldName := targetType.Field(i).Name
		if strings.ToUpper(fieldName) == upperName {
			return target.Field(i)
		}
	}

	// Try kebab-case to CamelCase conversion
	camelName := kebabToCamel(name)
	return target.FieldByName(camelName)
}

// setFieldValue sets a struct field value with type conversion.
func (t *ReflectTranslator[T]) setFieldValue(target reflect.Value, fieldName string, value interface{}) error {
	field := t.findField(target, fieldName)
	if !field.IsValid() {
		return nil // Skip unknown fields
	}

	if !field.CanSet() {
		return fmt.Errorf("cannot set field %s", fieldName)
	}

	// Handle pointer fields
	if field.Kind() == reflect.Ptr {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		field = field.Elem()
	}

	return t.convertAndSet(field, value)
}

// convertAndSet converts the value to the field type and sets it.
func (t *ReflectTranslator[T]) convertAndSet(field reflect.Value, value interface{}) error {
	switch field.Kind() {
	case reflect.String:
		if s, ok := value.(string); ok {
			field.SetString(s)
		} else {
			field.SetString(fmt.Sprintf("%v", value))
		}

	case reflect.Bool:
		if b, ok := value.(bool); ok {
			field.SetBool(b)
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch v := value.(type) {
		case float64:
			field.SetInt(int64(v))
		case int:
			field.SetInt(int64(v))
		case int32:
			field.SetInt(int64(v))
		case int64:
			field.SetInt(v)
		case string:
			if i, err := strconv.ParseInt(v, 10, 64); err == nil {
				field.SetInt(i)
			}
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch v := value.(type) {
		case float64:
			field.SetUint(uint64(v))
		case int:
			field.SetUint(uint64(v))
		case uint:
			field.SetUint(uint64(v))
		case uint32:
			field.SetUint(uint64(v))
		case uint64:
			field.SetUint(v)
		case string:
			if i, err := strconv.ParseUint(v, 10, 64); err == nil {
				field.SetUint(i)
			}
		}

	case reflect.Float32, reflect.Float64:
		switch v := value.(type) {
		case float64:
			field.SetFloat(v)
		case float32:
			field.SetFloat(float64(v))
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				field.SetFloat(f)
			}
		}
	}

	return nil
}

// setSliceField sets a slice field value.
func (t *ReflectTranslator[T]) setSliceField(field reflect.Value, values []interface{}) error {
	if !field.CanSet() {
		return nil
	}

	slice := reflect.MakeSlice(field.Type(), len(values), len(values))

	for i, val := range values {
		elem := slice.Index(i)
		if err := t.convertAndSet(elem, val); err != nil {
			return err
		}
	}

	field.Set(slice)
	return nil
}

// kebabToCamel converts kebab-case to CamelCase.
func kebabToCamel(s string) string {
	parts := strings.Split(s, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}
