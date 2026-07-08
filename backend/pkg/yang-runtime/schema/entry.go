package schema

import (
	"reflect"
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ytypes"
)

// AddYgotSchema populates ds with the modules described by an ygot-generated
// *ytypes.Schema (as returned by a generated package's Schema()). It converts the
// resolved goyang yang.Entry tree into the framework's Schema node model. This
// keeps the framework generic — the caller supplies the ygot schema; the
// framework does not import any concrete generated model package.
//
// The ygot schema's root entry (keyed in SchemaTree by the root struct name) has
// each top-level YANG container as a child; each such child is exposed as a Module.
func AddYgotSchema(ds *DefaultSchema, s *ytypes.Schema) {
	AddYgotSchemaWithVendor(ds, s, "")
}

// AddYgotSchemaWithVendor is like AddYgotSchema but tags each added module with
// the given vendor label (known by the caller from which generated package the
// schema came, since the unzipped entries carry no namespace).
func AddYgotSchemaWithVendor(ds *DefaultSchema, s *ytypes.Schema, vendor string) {
	if ds == nil || s == nil {
		return
	}
	rootEntry := ygotRootEntry(s)
	if rootEntry == nil {
		return
	}
	for _, child := range sortedDir(rootEntry) {
		ds.AddModule(entryToModule(child, vendor))
	}
}

// ygotRootEntry finds the root yang.Entry of an ygot schema. The SchemaTree map
// is keyed by generated struct name; the root struct is s.Root, so its type name
// is the map key for the root entry.
func ygotRootEntry(s *ytypes.Schema) *yang.Entry {
	if s.SchemaTree == nil {
		return nil
	}
	if s.Root != nil {
		t := reflect.TypeOf(s.Root)
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if e, ok := s.SchemaTree[t.Name()]; ok {
			return e
		}
	}
	// Fallback: an entry with no parent that has children.
	for _, e := range s.SchemaTree {
		if e != nil && e.Parent == nil && e.Dir != nil {
			return e
		}
	}
	return nil
}

// sortedDir returns e's directory children ordered by name for deterministic output.
func sortedDir(e *yang.Entry) []*yang.Entry {
	if e == nil || e.Dir == nil {
		return nil
	}
	names := make([]string, 0, len(e.Dir))
	for n := range e.Dir {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]*yang.Entry, 0, len(names))
	for _, n := range names {
		out = append(out, e.Dir[n])
	}
	return out
}

func entryNamespace(e *yang.Entry) string {
	if ns := e.Namespace(); ns != nil {
		return ns.Name
	}
	return ""
}

// entryToModule wraps a top-level container entry as a Module tagged with vendor.
func entryToModule(e *yang.Entry, vendor string) Module {
	root := entryToContainer(e, nil, "/"+e.Name)
	m := NewModule(e.Name, entryNamespace(e), "", root).(*defaultModule)
	m.vendor = vendor
	return m
}

// entryToNode dispatches an entry to the appropriate node kind.
func entryToNode(e *yang.Entry, parent Node, path string) Node {
	switch {
	case e.IsLeafList():
		// Modeled as a leaf carrying the element type, flagged as a leaf-list so the
		// form renders repeatable scalar values.
		leaf := entryToLeaf(e, parent, path, false)
		if dl, ok := leaf.(*defaultLeaf); ok {
			dl.leafList = true
		}
		return leaf
	case e.IsLeaf():
		return entryToLeaf(e, parent, path, false)
	case e.IsList():
		return entryToList(e, parent, path)
	default:
		return entryToContainer(e, parent, path)
	}
}

func entryToContainer(e *yang.Entry, parent Node, path string) ContainerNode {
	c := NewContainer(e.Name, e.Description, path, parent, false).(*defaultContainer)
	for _, child := range sortedDir(e) {
		if child.IsChoice() {
			// A choice contributes no data-path segment: its case members inherit
			// this container's `path` so their data paths stay flat.
			c.AddChild(entryToChoice(child, c, path))
			continue
		}
		if n := entryToNode(child, c, path+"/"+child.Name); n != nil {
			c.AddChild(n)
		}
	}
	return c
}

// entryToChoice converts a goyang choice Entry to a ChoiceNode. parentPath is the
// enclosing container/list path — case members are flattened onto it (choice and
// case names never appear in data paths), keeping the NETCONF write path intact.
func entryToChoice(e *yang.Entry, parent Node, parentPath string) ChoiceNode {
	ch := NewChoice(e.Name, e.Description, parentPath+"/"+e.Name, parent).(*defaultChoice)
	for _, caseEntry := range sortedDir(e) {
		ch.AddCase(entryToCase(caseEntry, ch, parentPath))
	}
	return ch
}

// entryToCase converts a goyang case Entry to a CaseNode. A "shorthand" case (a
// bare node directly under the choice, not wrapped in `case`) is treated as an
// implicit single-member case. Members inherit parentPath (no case segment).
func entryToCase(e *yang.Entry, parent Node, parentPath string) CaseNode {
	cs := NewCase(e.Name, e.Description, parentPath+"/"+e.Name, parent).(*defaultCase)
	if !e.IsCase() {
		// Shorthand case: the entry itself is the single member node.
		if n := caseMember(e, cs, parentPath); n != nil {
			cs.AddChild(n)
		}
		return cs
	}
	for _, child := range sortedDir(e) {
		if n := caseMember(child, cs, parentPath); n != nil {
			cs.AddChild(n)
		}
	}
	return cs
}

// caseMember builds one member of a case, flattening onto parentPath. A choice
// nested directly in a case (no intervening container) recurses as a nested choice.
func caseMember(e *yang.Entry, parent Node, parentPath string) Node {
	if e.IsChoice() {
		return entryToChoice(e, parent, parentPath)
	}
	return entryToNode(e, parent, parentPath+"/"+e.Name)
}

func entryToList(e *yang.Entry, parent Node, path string) ListNode {
	keyNames := map[string]bool{}
	for _, k := range strings.Fields(e.Key) {
		keyNames[k] = true
	}
	l := NewList(e.Name, e.Description, path, parent, nil, false).(*defaultList)
	var keys []LeafNode
	for _, child := range sortedDir(e) {
		if child.IsChoice() {
			// Choice members are flattened onto the list path (see entryToChoice).
			l.AddChild(entryToChoice(child, l, path))
			continue
		}
		childPath := path + "/" + child.Name
		var n Node
		if child.IsLeaf() && keyNames[child.Name] {
			leaf := entryToLeaf(child, l, childPath, true)
			keys = append(keys, leaf)
			n = leaf
		} else {
			n = entryToNode(child, l, childPath)
		}
		if n != nil {
			l.AddChild(n)
		}
	}
	l.keys = keys
	return l
}

func entryToLeaf(e *yang.Entry, parent Node, path string, isKey bool) LeafNode {
	leaf := NewLeaf(e.Name, e.Description, path, parent, mapLeafType(e.Type), isKey, e.Mandatory.Value()).(*defaultLeaf)
	if e.Type != nil {
		if e.Type.Enum != nil {
			leaf.enumValues = append([]string(nil), e.Type.Enum.Names()...)
		}
		leaf.units = e.Type.Units
		if e.Type.Default != "" {
			leaf.defaultValue = e.Type.Default
		}
		if len(e.Type.Pattern) > 0 {
			leaf.pattern = e.Type.Pattern[0]
		}
		leaf.rangeMin, leaf.hasMin, leaf.rangeMax, leaf.hasMax = leafRangeBounds(e.Type)
	}
	leaf.whenExpr = firstExtraExpr(e.Extra["when"])
	leaf.mustExprs = allExtraExprs(e.Extra["must"])
	return leaf
}

// leafRangeBounds extracts integer min/max from a leaf's YANG `range`. It returns
// no bounds when: there is no range, the range is merely the type's full default
// (i.e. no explicit `range` statement), or a bound is non-integer/overflows int
// (callers then omit that bound — R08, no panic).
func leafRangeBounds(yt *yang.YangType) (min int, hasMin bool, max int, hasMax bool) {
	if yt == nil || len(yt.Range) == 0 {
		return
	}
	if def := defaultRangeForKind(yt.Kind); def != nil && yt.Range.String() == def.String() {
		return // full type-default range → not an explicit constraint
	}
	if v, err := yt.Range[0].Min.Int(); err == nil {
		min, hasMin = int(v), true
	}
	if v, err := yt.Range[len(yt.Range)-1].Max.Int(); err == nil {
		max, hasMax = int(v), true
	}
	return
}

// defaultRangeForKind returns goyang's full default range for an integer kind, or
// nil for non-integer kinds. Used to distinguish explicit ranges from type bounds.
func defaultRangeForKind(k yang.TypeKind) yang.YangRange {
	switch k {
	case yang.Yint8:
		return yang.Int8Range
	case yang.Yint16:
		return yang.Int16Range
	case yang.Yint32:
		return yang.Int32Range
	case yang.Yint64:
		return yang.Int64Range
	case yang.Yuint8:
		return yang.Uint8Range
	case yang.Yuint16:
		return yang.Uint16Range
	case yang.Yuint32:
		return yang.Uint32Range
	case yang.Yuint64:
		return yang.Uint64Range
	default:
		return nil
	}
}

// firstExtraExpr returns the XPath argument of the first element of a goyang
// Entry.Extra slice (e.g. Extra["when"]/["must"]). It tolerates the two shapes
// that occur at runtime: the ygot-unzipped JSON map ({"Name": "<xpath>"}) and the
// goyang-parsed *yang.Value struct (exported Name field). Returns "" if absent or
// unrecognized — callers degrade gracefully (R08), never panic.
func firstExtraExpr(extra []interface{}) string {
	for _, v := range extra {
		if s := extraExprName(v); s != "" {
			return s
		}
	}
	return ""
}

// allExtraExprs returns the XPath argument of every element of an Entry.Extra slice
// (order-preserved, empties skipped). Used for `must` where a leaf may carry many.
func allExtraExprs(extra []interface{}) []string {
	var out []string
	for _, v := range extra {
		if s := extraExprName(v); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func extraExprName(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case map[string]interface{}:
		if s, ok := t["Name"].(string); ok {
			return s
		}
		return ""
	case interface{ NName() string }:
		return t.NName()
	}
	// Reflection fallback for structs (e.g. *yang.Value) with an exported Name.
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Struct {
		if f := rv.FieldByName("Name"); f.IsValid() && f.Kind() == reflect.String {
			return f.String()
		}
	}
	return ""
}

// mapLeafType maps a resolved goyang YANG type to the framework LeafType.
func mapLeafType(yt *yang.YangType) LeafType {
	if yt == nil {
		return LeafTypeString
	}
	switch yt.Kind {
	case yang.Ybool:
		return LeafTypeBoolean
	case yang.Yint8:
		return LeafTypeInt8
	case yang.Yint16:
		return LeafTypeInt16
	case yang.Yint32:
		return LeafTypeInt32
	case yang.Yint64:
		return LeafTypeInt64
	case yang.Yuint8:
		return LeafTypeUint8
	case yang.Yuint16:
		return LeafTypeUint16
	case yang.Yuint32:
		return LeafTypeUint32
	case yang.Yuint64:
		return LeafTypeUint64
	case yang.Yenum, yang.Yidentityref:
		return LeafTypeEnum
	case yang.Yempty:
		return LeafTypeEmpty
	case yang.Ydecimal64:
		return LeafTypeDecimal64
	case yang.Ybits:
		return LeafTypeBits
	default:
		// string / union / leafref / binary / instance-identifier
		return LeafTypeString
	}
}
