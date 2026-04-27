package schema

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

// Loader loads YANG files from a directory and builds a Schema
type Loader struct {
	schemaDir string
}

// NewLoader creates a new Loader
func NewLoader(schemaDir string) *Loader {
	return &Loader{
		schemaDir: schemaDir,
	}
}

// Load loads all YANG files from the schema directory and builds a Schema
func (l *Loader) Load() (Schema, error) {
	schema := NewSchema()

	// Find all .yang files in the directory
	var yangFiles []string
	err := filepath.Walk(l.schemaDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".yang") {
			yangFiles = append(yangFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("schema: failed to walk directory: %w", err)
	}

	if len(yangFiles) == 0 {
		return schema, nil
	}

	// Parse YANG files with goyang
	modules := yang.NewModules()
	// Add the schema directory to the search path for imports
	modules.AddPath(l.schemaDir)
	modules.AddPath(filepath.Dir(l.schemaDir))

	var errs []error
	for _, f := range yangFiles {
		err := modules.Read(f)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("schema: failed to read YANG files: %v", errs)
	}

	errs = modules.Process()
	if len(errs) > 0 {
		return nil, fmt.Errorf("schema: failed to process YANG modules: %v", errs)
	}

	// Process each module into our schema model
	for _, m := range modules.Modules {
		module, err := l.convertModule(m)
		if err != nil {
			return nil, err
		}
		schema.AddModule(module)
	}

	return schema, nil
}

// convertModule converts a yang.Module to our Module interface
func (l *Loader) convertModule(m *yang.Module) (Module, error) {
	description := ""
	if m.Description != nil {
		description = m.Description.Name
	}

	namespace := ""
	if m.Namespace != nil {
		namespace = m.Namespace.Name
	}

	revision := ""
	if len(m.Revision) > 0 {
		revision = m.Revision[0].Name
	}

	// Build the root container from the module
	root := NewContainer(m.Name, description, "/", nil, false)

	// Process top-level containers, leafs, lists
	for _, container := range m.Container {
		err := l.processContainer(container, root, "/")
		if err != nil {
			return nil, err
		}
	}
	for _, leaf := range m.Leaf {
		yLeaf := l.convertLeaf(leaf, "/"+leaf.Name, root, false)
		root.(*defaultContainer).AddChild(yLeaf)
	}
	for _, list := range m.List {
		err := l.processList(list, root, "/")
		if err != nil {
			return nil, err
		}
	}

	module := NewModule(m.Name, namespace, revision, root)
	return module, nil
}

// processContainer processes a yang.Container into our ContainerNode
func (l *Loader) processContainer(c *yang.Container, parent ContainerNode, parentPath string) error {
	childPath := parentPath
	if childPath != "/" {
		childPath += "/"
	}
	childPath += c.Name

	description := ""
	if c.Description != nil {
		description = c.Description.Name
	}

	isPresence := false
	container := NewContainer(c.Name, description, childPath, parent, isPresence)

	// Process child nodes
	for _, childContainer := range c.Container {
		err := l.processContainer(childContainer, container, childPath)
		if err != nil {
			return err
		}
	}
	for _, leaf := range c.Leaf {
		yLeaf := l.convertLeaf(leaf, childPath+"/"+leaf.Name, container, false)
		container.(*defaultContainer).AddChild(yLeaf)
	}
	for _, list := range c.List {
		err := l.processList(list, container, childPath)
		if err != nil {
			return err
		}
	}

	parent.(*defaultContainer).AddChild(container)
	return nil
}

// processList processes a yang.List into our ListNode
func (l *Loader) processList(listDef *yang.List, parent ContainerNode, parentPath string) error {
	childPath := parentPath
	if childPath != "/" {
		childPath += "/"
	}
	childPath += listDef.Name

	description := ""
	if listDef.Description != nil {
		description = listDef.Description.Name
	}

	// Extract keys from the list - listDef.Key is a space-separated string
	var keys []LeafNode
	if listDef.Key != nil {
		keyNames := strings.Fields(listDef.Key.Name)
		for _, keyName := range keyNames {
			// Find the key leaf in the list
			found := false
			for _, leaf := range listDef.Leaf {
				if leaf.Name == keyName {
					leafPath := childPath + "/" + leaf.Name
					yLeaf := l.convertLeaf(leaf, leafPath, parent, true)
					keys = append(keys, yLeaf)
					found = true
					break
				}
			}
			if !found {
				// Key not found, skip
				continue
			}
		}
	}

	// Check if user-ordered
	isUserOrdered := false
	if listDef.OrderedBy != nil && listDef.OrderedBy.Name == "user" {
		isUserOrdered = true
	}

	list := NewList(listDef.Name, description, childPath, parent, keys, isUserOrdered)

	// Process child nodes
	for _, childContainer := range listDef.Container {
		err := l.processContainer(childContainer, list.(*defaultList), childPath)
		if err != nil {
			return err
		}
	}
	for _, leaf := range listDef.Leaf {
		// Check if this leaf is already a key
		isKey := false
		for _, k := range keys {
			if k.Name() == leaf.Name {
				isKey = true
				break
			}
		}
		if isKey {
			// Already processed as key
			continue
		}
		yLeaf := l.convertLeaf(leaf, childPath+"/"+leaf.Name, list, false)
		list.(*defaultList).AddChild(yLeaf)
	}
	for _, childList := range listDef.List {
		err := l.processList(childList, list.(*defaultList), childPath)
		if err != nil {
			return err
		}
	}

	parent.(*defaultContainer).AddChild(list)
	return nil
}

// convertLeaf converts a yang.Leaf to our LeafNode
func (l *Loader) convertLeaf(leafDef *yang.Leaf, path string, parent Node, isKey bool) LeafNode {
	description := ""
	if leafDef.Description != nil {
		description = leafDef.Description.Name
	}

	leafType := convertType(leafDef.Type)
	mandatory := false
	if leafDef.Mandatory != nil {
		mandatory = leafDef.Mandatory.Name == "true"
	}
	dl := NewLeaf(leafDef.Name, description, path, parent, leafType, isKey, mandatory)

	// Set default value if present
	if leafDef.Default != nil {
		dl.(*defaultLeaf).SetDefault(leafDef.Default.Name)
	}

	// Set enum values if this is an enumeration
	if leafDef.Type != nil {
		kindName := leafDef.Type.Kind()
		kind, ok := yang.TypeKindFromName[kindName]
		if ok && kind == yang.Yenum {
			var enums []string
			for _, e := range leafDef.Type.Enum {
				enums = append(enums, e.Name)
			}
			dl.(*defaultLeaf).SetEnumValues(enums)
		}
	}

	// Set units if present
	if leafDef.Units != nil {
		dl.(*defaultLeaf).SetUnits(leafDef.Units.Name)
	}

	return dl
}

// convertType converts yang type to our LeafType
func convertType(t *yang.Type) LeafType {
	if t == nil {
		return LeafTypeString
	}
	kindName := t.Kind()
	kind, ok := yang.TypeKindFromName[kindName]
	if !ok {
		return LeafTypeString
	}

	switch kind {
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
	case yang.Ystring:
		return LeafTypeString
	case yang.Yenum:
		return LeafTypeEnum
	case yang.Yempty:
		return LeafTypeEmpty
	case yang.Ydecimal64:
		return LeafTypeDecimal64
	case yang.Ybits:
		return LeafTypeBits
	default:
		// Default to string for unknown types
		return LeafTypeString
	}
}
