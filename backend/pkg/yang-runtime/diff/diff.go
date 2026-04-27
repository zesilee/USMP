package diff

import (
	"fmt"
	"reflect"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// DiffEngine computes differences between two YANG config trees
type DiffEngine interface {
	// Diff computes the differences between desired and actual config
	// Both desired and actual must be compatible with the schema
	Diff(desired, actual interface{}, s schema.Schema) (*DiffResult, error)
	// DiffAtPath computes differences starting at a specific path
	DiffAtPath(path string, desired, actual interface{}) (*DiffResult, error)
}

// DefaultDiffEngine is the default implementation of DiffEngine
// that performs recursive tree comparison with list key matching
type DefaultDiffEngine struct {
	pruneRedundant bool // Prune redundant changes when parent is replaced
}

// NewDefaultDiffEngine creates a new DefaultDiffEngine
func NewDefaultDiffEngine() *DefaultDiffEngine {
	return &DefaultDiffEngine{
		pruneRedundant: true,
	}
}

// WithPruneRedundant sets whether to prune redundant changes
func (de *DefaultDiffEngine) WithPruneRedundant(prune bool) *DefaultDiffEngine {
	de.pruneRedundant = prune
	return de
}

// Diff implements DiffEngine interface
func (de *DefaultDiffEngine) Diff(desired, actual interface{}, s schema.Schema) (*DiffResult, error) {
	result := NewDiffResult()
	walker := &diffWalker{
		result: result,
	}
	err := de.walk("", "", desired, actual, walker)
	if err != nil {
		return nil, err
	}
	if de.pruneRedundant {
		walker.result = de.pruneChanges(walker.result.Changes)
	}
	return walker.result, nil
}

// DiffAtPath implements DiffEngine interface
func (de *DefaultDiffEngine) DiffAtPath(basePath string, desired, actual interface{}) (*DiffResult, error) {
	result := NewDiffResult()
	walker := &diffWalker{
		result: result,
	}
	err := de.walk("", basePath, desired, actual, walker)
	if err != nil {
		return nil, err
	}
	if de.pruneRedundant {
		walker.result = de.pruneChanges(walker.result.Changes)
	}
	return walker.result, nil
}

type diffWalker struct {
	result *DiffResult
}

func (de *DefaultDiffEngine) walk(parentPath, schemaPath string, desired, actual interface{}, w *diffWalker) error {
	// Check for nil/nil-pointer cases
	// Handle both nil interface and nil pointer inside interface
	desiredNil := desired == nil || (reflect.ValueOf(desired).Kind() == reflect.Ptr && reflect.ValueOf(desired).IsNil())
	actualNil := actual == nil || (reflect.ValueOf(actual).Kind() == reflect.Ptr && reflect.ValueOf(actual).IsNil())

	switch {
	case desiredNil && actualNil:
		// No change
		return nil
	case desiredNil && !actualNil:
		// Entire subtree deleted
		// We only need to record the delete at this level
		w.result.AddChange(Change{
			Type:       DeleteChange,
			Path:       parentPath,
			OldValue:   actual,
			NewValue:   nil,
			SchemaPath: schemaPath,
		})
		return nil
	case !desiredNil && actualNil:
		// Entire subtree added
		w.result.AddChange(Change{
			Type:       AddChange,
			Path:       parentPath,
			OldValue:   nil,
			NewValue:   desired,
			SchemaPath: schemaPath,
		})
		return nil
	}

	// Check if they are the same type
	dv := reflect.ValueOf(desired)
	av := reflect.ValueOf(actual)

	if dv.Kind() != av.Kind() {
		// Different types - treat as modify
		w.result.AddChange(Change{
			Type:       ModifyChange,
			Path:       parentPath,
			OldValue:   actual,
			NewValue:   desired,
			SchemaPath: schemaPath,
		})
		return nil
	}

	// Handle based on kind
	switch dv.Kind() {
	case reflect.Ptr, reflect.Interface:
		// Both are non-nil, dereference and continue
		return de.walk(parentPath, schemaPath, dv.Elem().Interface(), av.Elem().Interface(), w)

	case reflect.Slice, reflect.Array:
		return de.walkSlice(parentPath, schemaPath, desired, actual, w)

	case reflect.Struct:
		return de.walkStruct(parentPath, schemaPath, desired, actual, w)

	default:
		// Leaf value - compare directly
		if !reflect.DeepEqual(desired, actual) {
			w.result.AddChange(Change{
				Type:       ModifyChange,
				Path:       parentPath,
				OldValue:   actual,
				NewValue:   desired,
				SchemaPath: schemaPath,
			})
		}
		return nil
	}
}

func (de *DefaultDiffEngine) walkStruct(parentPath, schemaPath string, desired, actual interface{}, w *diffWalker) error {
	dv := reflect.ValueOf(desired)
	av := reflect.ValueOf(actual)
	dType := dv.Type()

	// Iterate over all fields
	for i := 0; i < dv.NumField(); i++ {
		dField := dv.Field(i)
		aField := av.Field(i)
		structField := dType.Field(i)

		if !structField.IsExported() {
			continue // Skip unexported fields
		}

		fieldName := structField.Name
		// JSON/YANG field name is usually the same as the struct field name
		// For ygot generated structs, this matches
		childPath := parentPath
		if childPath == "" {
			childPath = fieldName
		} else {
			childPath = childPath + "/" + fieldName
		}
		childSchemaPath := schemaPath
		if childSchemaPath == "" {
			childSchemaPath = fieldName
		} else {
			childSchemaPath = childSchemaPath + "/" + fieldName
		}

		dVal := dField.Interface()
		aVal := aField.Interface()

		if err := de.walk(childPath, childSchemaPath, dVal, aVal, w); err != nil {
			return err
		}
	}

	return nil
}

func (de *DefaultDiffEngine) walkSlice(parentPath, schemaPath string, desired, actual interface{}, w *diffWalker) error {
	// For YANG lists, we expect this to be a slice of structs with key fields
	// We compare by matching keys and then comparing the content of each entry

	dSlice := reflect.ValueOf(desired)
	aSlice := reflect.ValueOf(actual)

	// Create map of existing entries by key
	aMap := de.indexListEntries(aSlice)

	// Process each desired entry
	for i := 0; i < dSlice.Len(); i++ {
		dEntry := dSlice.Index(i)
		dKey := de.extractKey(dEntry)

		if aEntry, exists := aMap[dKey]; exists {
			// Entry exists in both, compare content
			childPath := fmt.Sprintf("%s[%s]", parentPath, dKey)
			if err := de.walk(childPath, schemaPath, dEntry.Interface(), aEntry.Interface(), w); err != nil {
				return err
			}
			delete(aMap, dKey)
		} else {
			// Entry is new, add it
			childPath := fmt.Sprintf("%s[%s]", parentPath, dKey)
			w.result.AddChange(Change{
				Type:       AddChange,
				Path:       childPath,
				OldValue:   nil,
				NewValue:   dEntry.Interface(),
				SchemaPath: schemaPath,
			})
		}
	}

	// Any remaining entries in actual that are not in desired get deleted
	for key, aEntry := range aMap {
		childPath := fmt.Sprintf("%s[%s]", parentPath, key)
		w.result.AddChange(Change{
			Type:       DeleteChange,
			Path:       childPath,
			OldValue:   aEntry.Interface(),
			NewValue:   nil,
			SchemaPath: schemaPath,
		})
	}

	return nil
}

// extractKey extracts a string key from a list entry
// For simplicity, we concatenate all key fields with = separator
// This matches YANG path syntax
func (de *DefaultDiffEngine) extractKey(entry reflect.Value) string {
	// For ygot generated lists, key fields have the corresponding `...Key` field
	// We look for fields ending with Key and extract them
	// Fallback: use the first field that can be converted to string
	var keyParts []string

	// First pass: look for fields named *Key
	t := entry.Type()
	for i := 0; i < entry.NumField(); i++ {
		f := entry.Field(i)
		if !f.CanInterface() {
			continue
		}
		fieldName := t.Field(i).Name
		if len(fieldName) > 3 && fieldName[len(fieldName)-3:] == "Key" {
			// This is a key field per ygot convention
			keyStr := fmt.Sprintf("%v", f.Interface())
			keyParts = append(keyParts, fieldName[:len(fieldName)-3]+"="+keyStr)
		}
	}

	if len(keyParts) == 0 {
		// Fallback: use the first exported field
		for i := 0; i < entry.NumField(); i++ {
			f := entry.Field(i)
			if f.CanInterface() {
				keyStr := fmt.Sprintf("%v", f.Interface())
				keyParts = append(keyParts, t.Field(i).Name+"="+keyStr)
				break
			}
		}
	}

	// Join all key parts with commas
	result := ""
	for i, p := range keyParts {
		if i > 0 {
			result += ","
		}
		result += p
	}
	return result
}

// indexListEntries creates a map of entries by their key
func (de *DefaultDiffEngine) indexListEntries(slice reflect.Value) map[string]reflect.Value {
	result := make(map[string]reflect.Value)
	for i := 0; i < slice.Len(); i++ {
		entry := slice.Index(i)
		key := de.extractKey(entry)
		result[key] = entry
	}
	return result
}

// pruneChanges removes redundant changes when a parent is already changed
// If a parent node is added or deleted, all child changes are redundant
// This reduces the number of changes sent to the device
func (de *DefaultDiffEngine) pruneChanges(changes []Change) *DiffResult {
	result := NewDiffResult()

	// For each change, check if it has an ancestor that is already
	// an add or delete - if so, this change is redundant
	for _, c := range changes {
		redundant := false
		parentPath := c.Path
		for {
			parentPath = schema.GetParentPath(parentPath)
			if parentPath == "" {
				break
			}
			// Check if any ancestor is add/delete
			for _, ancestor := range changes {
				if ancestor.Path == parentPath && (ancestor.Type == AddChange || ancestor.Type == DeleteChange) {
					// Ancestor is already being added/deleted, this change is redundant
					redundant = true
					break
				}
			}
			if redundant {
				break
			}
		}
		if !redundant {
			result.AddChange(c)
		}
	}

	return result
}
