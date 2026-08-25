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
	// Both a nil interface and a non-nil interface holding a nil pointer count as absent.
	desiredNil := desired == nil || (reflect.ValueOf(desired).Kind() == reflect.Ptr && reflect.ValueOf(desired).IsNil())
	actualNil := actual == nil || (reflect.ValueOf(actual).Kind() == reflect.Ptr && reflect.ValueOf(actual).IsNil())

	switch {
	case desiredNil && actualNil:
		return nil
	case desiredNil && !actualNil:
		// Entire subtree deleted: recording the delete at this level is enough.
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

	switch dv.Kind() {
	case reflect.Ptr, reflect.Interface:
		// Both are non-nil, dereference and continue
		return de.walk(parentPath, schemaPath, dv.Elem().Interface(), av.Elem().Interface(), w)

	case reflect.Slice, reflect.Array:
		return de.walkSlice(parentPath, schemaPath, desired, actual, w)

	case reflect.Map:
		return de.walkMap(parentPath, schemaPath, desired, actual, w)

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

	for i := 0; i < dv.NumField(); i++ {
		dField := dv.Field(i)
		aField := av.Field(i)
		structField := dType.Field(i)

		if !structField.IsExported() {
			continue
		}

		// The generated structs name fields after their YANG node, so the Go
		// field name doubles as the path segment.
		fieldName := structField.Name
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
	// A YANG list arrives as a slice of key-bearing structs: entries are paired
	// up by key and only then compared field by field.
	dSlice := reflect.ValueOf(desired)
	aSlice := reflect.ValueOf(actual)

	aMap := de.indexListEntries(aSlice)

	for i := 0; i < dSlice.Len(); i++ {
		dEntry := dSlice.Index(i)
		dKey := de.extractKey(dEntry)

		if aEntry, exists := aMap[dKey]; exists {
			childPath := fmt.Sprintf("%s[%s]", parentPath, dKey)
			if err := de.walk(childPath, schemaPath, dEntry.Interface(), aEntry.Interface(), w); err != nil {
				return err
			}
			delete(aMap, dKey)
		} else {
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

	// Whatever is left in aMap was matched by no desired entry, so it is gone.
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

// walkMap handles ygot-generated YANG lists, which ygot renders as Go maps
// (map[key]*Entry) rather than slices — so without this branch a list field would
// fall through to the leaf default and be compared with reflect.DeepEqual, which is
// always false when desired is the UI's sparse intent and actual is the device's full
// readback (extra keys + device defaults + read-only leaves). 那样每轮都产出 change →
// 对账永不收敛 → 前端「一直漂移」。
//
// 采用「合并/子集」语义，与 config_handler.storeConfigMerged 把 desired 当累积意图一致：
//   - desired 的每个 key 必须在 actual 出现，且其「已设字段」匹配，否则视为需下发的漂移；
//   - actual 独有的 key（设备物理口/默认条目）忽略，绝不产生 DeleteChange（不误删）。
//
// 一旦 desired 被 actual 满足即 0 change → 收敛。未满足时产出单个整表 ModifyChange
// （NewValue = desired 内层 map），交由 client.marshalChange 走对应模型的 XML builder
// 做 merge edit-config（VLAN/IFM 各有专用序列化分支）。
func (de *DefaultDiffEngine) walkMap(parentPath, schemaPath string, desired, actual interface{}, w *diffWalker) error {
	if de.subsetMatches(reflect.ValueOf(desired), reflect.ValueOf(actual)) {
		return nil
	}
	w.result.AddChange(Change{
		Type:       ModifyChange,
		Path:       parentPath,
		OldValue:   actual,
		NewValue:   desired,
		SchemaPath: schemaPath,
	})
	return nil
}

// subsetMatches reports whether every value the caller explicitly set in desired is
// already present and equal in actual — i.e. desired ⊆ actual under merge semantics.
// 「已设」= 非零值：nil 指针 / 空 map/slice / 枚举 0(UNSET) / "" / 0 / false 都算未设，
// 视为「不管理」→ 直接匹配，从而不会因设备侧的额外字段/条目而误报漂移。
func (de *DefaultDiffEngine) subsetMatches(d, a reflect.Value) bool {
	// desired 未设（零值/无效）→ 不管理，视为匹配
	if !d.IsValid() || d.IsZero() {
		return true
	}
	// desired 已设但 actual 缺失 → 漂移
	if !a.IsValid() {
		return false
	}

	// 解包 interface 层
	if d.Kind() == reflect.Interface {
		d = d.Elem()
	}
	if a.Kind() == reflect.Interface {
		a = a.Elem()
	}
	if !d.IsValid() || d.IsZero() {
		return true
	}
	if !a.IsValid() {
		return false
	}
	if d.Kind() != a.Kind() {
		return false
	}

	switch d.Kind() {
	case reflect.Ptr:
		if d.IsNil() {
			return true
		}
		if a.IsNil() {
			return false
		}
		return de.subsetMatches(d.Elem(), a.Elem())

	case reflect.Struct:
		// desired/actual 在对账中恒为同一 ygot 类型；类型不一致时退化为整体相等比较，
		// 避免按 desired 字段索引 actual 字段导致越界 panic（R08/R09 禁止崩溃）。
		if d.Type() != a.Type() {
			return reflect.DeepEqual(d.Interface(), a.Interface())
		}
		dType := d.Type()
		for i := 0; i < d.NumField(); i++ {
			if !dType.Field(i).IsExported() {
				continue
			}
			if !de.subsetMatches(d.Field(i), a.Field(i)) {
				return false
			}
		}
		return true

	case reflect.Map:
		for _, k := range d.MapKeys() {
			dv := d.MapIndex(k)
			if !dv.IsValid() || dv.IsZero() {
				continue // 未设条目不管理
			}
			av := a.MapIndex(k)
			if !av.IsValid() {
				return false // desired 声明的 key 不在设备上 → 需下发
			}
			if !de.subsetMatches(dv, av) {
				return false
			}
		}
		return true

	case reflect.Slice, reflect.Array:
		// leaf-list / 有序列表：desired 每个元素需能在 actual 中找到匹配（子集）
		for i := 0; i < d.Len(); i++ {
			found := false
			for j := 0; j < a.Len(); j++ {
				if de.subsetMatches(d.Index(i), a.Index(j)) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true

	default:
		return reflect.DeepEqual(d.Interface(), a.Interface())
	}
}

// extractKey builds a string key for a list entry by joining every key field as
// "name=value", which matches YANG path predicate syntax.
//
// Generated lists carry their keys in fields suffixed with "Key"; when an entry
// has none, the first exported field stands in as the key.
func (de *DefaultDiffEngine) extractKey(entry reflect.Value) string {
	var keyParts []string

	t := entry.Type()
	for i := 0; i < entry.NumField(); i++ {
		f := entry.Field(i)
		if !f.CanInterface() {
			continue
		}
		fieldName := t.Field(i).Name
		if len(fieldName) > 3 && fieldName[len(fieldName)-3:] == "Key" {
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

// pruneChanges drops changes whose ancestor is already being added or deleted:
// the ancestor change carries the whole subtree, so the descendants only inflate
// the payload sent to the device.
func (de *DefaultDiffEngine) pruneChanges(changes []Change) *DiffResult {
	result := NewDiffResult()

	for _, c := range changes {
		redundant := false
		parentPath := c.Path
		for {
			parentPath = schema.GetParentPath(parentPath)
			if parentPath == "" {
				break
			}
			for _, ancestor := range changes {
				if ancestor.Path == parentPath && (ancestor.Type == AddChange || ancestor.Type == DeleteChange) {
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
