package schema

import "path/filepath"

// NodeType represents the type of a YANG schema node
type NodeType int

const (
	// ContainerNodeType represents a YANG container node
	ContainerNodeType NodeType = iota
	// ListNodeType represents a YANG list node
	ListNodeType
	// LeafNodeType represents a YANG leaf node
	LeafNodeType
	// LeafListNodeType represents a YANG leaf-list node
	LeafListNodeType
	// ChoiceNodeType represents a YANG choice node
	ChoiceNodeType
	// CaseNodeType represents a YANG case node
	CaseNodeType
)

// LeafType represents the data type of a YANG leaf
type LeafType int

const (
	// LeafTypeBoolean represents a boolean leaf
	LeafTypeBoolean LeafType = iota
	// LeafTypeInt8 represents an int8 leaf
	LeafTypeInt8
	// LeafTypeInt16 represents an int16 leaf
	LeafTypeInt16
	// LeafTypeInt32 represents an int32 leaf
	LeafTypeInt32
	// LeafTypeInt64 represents an int64 leaf
	LeafTypeInt64
	// LeafTypeUint8 represents an uint8 leaf
	LeafTypeUint8
	// LeafTypeUint16 represents an uint16 leaf
	LeafTypeUint16
	// LeafTypeUint32 represents an uint32 leaf
	LeafTypeUint32
	// LeafTypeUint64 represents an uint64 leaf
	LeafTypeUint64
	// LeafTypeString represents a string leaf
	LeafTypeString
	// LeafTypeEnum represents an enumeration leaf
	LeafTypeEnum
	// LeafTypeEmpty represents an empty leaf
	LeafTypeEmpty
	// LeafTypeDecimal64 represents a decimal64 leaf
	LeafTypeDecimal64
	// LeafTypeBits represents a bits leaf
	LeafTypeBits
)

// Schema represents a loaded and cached YANG schema
type Schema interface {
	// Module returns a YANG module by name
	Module(name string) (Module, bool)
	// Modules returns all loaded modules
	Modules() []Module
	// Path resolves a YANG path to the corresponding schema node
	Path(path string) (Node, bool)
	// Validate validates a configuration against the schema
	Validate(path string, config interface{}) error
}

// Module represents a YANG module
type Module interface {
	// Name returns the module name
	Name() string
	// Namespace returns the module namespace
	Namespace() string
	// Vendor returns the vendor label ("huawei"/"openconfig"/…) or "" if unknown
	Vendor() string
	// Revision returns the module revision
	Revision() string
	// Root returns the root container node
	Root() ContainerNode
	// Path returns the full path to the module root
	Path() string
}

// Node is the base interface for all YANG schema nodes
type Node interface {
	// Name returns the node name
	Name() string
	// Description returns the node description
	Description() string
	// Path returns the absolute path from the root
	Path() string
	// Type returns the node type
	Type() NodeType
	// Parent returns the parent node, nil for root
	Parent() Node
	// SchemaPath returns the schema path in the module
	SchemaPath() string
}

// ContainerNode represents a YANG container node
type ContainerNode interface {
	Node
	// Children returns all child nodes
	Children() []Node
	// Child returns a child by name
	Child(name string) (Node, bool)
	// IsPresence returns whether this is a presence container
	IsPresence() bool
}

// ListNode represents a YANG list node
type ListNode interface {
	Node
	// Keys returns the list of key leaf nodes
	Keys() []LeafNode
	// Children returns all child nodes
	Children() []Node
	// Child returns a child by name
	Child(name string) (Node, bool)
	// IsUserOrdered returns whether the list is user-ordered
	IsUserOrdered() bool
}

// LeafNode represents a YANG leaf node
type LeafNode interface {
	Node
	// LeafType returns the leaf data type
	LeafType() LeafType
	// IsKey returns whether this leaf is a key for a list
	IsKey() bool
	// DefaultValue returns the default value, nil if none
	DefaultValue() interface{}
	// EnumValues returns the enumeration values if this is an enum
	EnumValues() []string
	// Mandatory returns whether this leaf is mandatory
	Mandatory() bool
	// Units returns the units string, empty if none
	Units() string
	// WhenExpr returns the leaf's YANG `when` XPath expression, "" if none.
	// Drives data-driven conditional visibility in the dynamic form (R05).
	WhenExpr() string
	// MustExprs returns the leaf's YANG `must` XPath expressions (order-preserved),
	// empty if none. Drives data-driven cross-field validation in the form (R05).
	MustExprs() []string
}

// LeafListNode represents a YANG leaf-list node
type LeafListNode interface {
	Node
	// LeafType returns the leaf data type
	LeafType() LeafType
	// IsUserOrdered returns whether the leaf-list is user-ordered
	IsUserOrdered() bool
	// MinimumElements returns the minimum number of elements
	MinimumElements() uint64
	// MaximumElements returns the maximum number of elements, 0 means unlimited
	MaximumElements() uint64
}

// ChoiceNode represents a YANG choice node
type ChoiceNode interface {
	Node
	// Cases returns all case nodes
	Cases() []CaseNode
	// Case returns a case by name
	Case(name string) (CaseNode, bool)
	// DefaultCase returns the default case name, empty if none
	DefaultCase() string
}

// CaseNode represents a YANG case node
type CaseNode interface {
	Node
	// Children returns all child nodes
	Children() []Node
	// Child returns a child by name
	Child(name string) (Node, bool)
}

// SplitPath splits a YANG path into components
func SplitPath(path string) []string {
	if path == "" || path == "/" {
		return []string{}
	}
	// Strip leading and trailing slashes
	for len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	for len(path) > 0 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	if path == "" {
		return []string{}
	}
	// Split on forward slashes
	var components []string
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			components = append(components, path[start:i])
			start = i + 1
		}
	}
	components = append(components, path[start:])
	return components
}

// JoinPath joins path components into a YANG path
func JoinPath(components []string) string {
	if len(components) == 0 {
		return "/"
	}
	return "/" + filepath.Join(components...)
}
