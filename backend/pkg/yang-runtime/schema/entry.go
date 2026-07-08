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
		// Modeled as a leaf carrying the element type (leaf-list refinement later).
		return entryToLeaf(e, parent, path, false)
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
		if n := entryToNode(child, c, path+"/"+child.Name); n != nil {
			c.AddChild(n)
		}
	}
	return c
}

func entryToList(e *yang.Entry, parent Node, path string) ListNode {
	keyNames := map[string]bool{}
	for _, k := range strings.Fields(e.Key) {
		keyNames[k] = true
	}
	l := NewList(e.Name, e.Description, path, parent, nil, false).(*defaultList)
	var keys []LeafNode
	for _, child := range sortedDir(e) {
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
	}
	leaf.whenExpr = firstExtraExpr(e.Extra["when"])
	return leaf
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
