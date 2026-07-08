package schema

import (
	"fmt"
	"sync"
)

// DefaultSchema is the default implementation of Schema
type DefaultSchema struct {
	modules   map[string]Module
	pathCache map[string]Node
	mu        sync.RWMutex
}

// NewSchema creates a new empty DefaultSchema
func NewSchema() *DefaultSchema {
	return &DefaultSchema{
		modules:   make(map[string]Module),
		pathCache: make(map[string]Node),
	}
}

// Module implements Schema interface
func (s *DefaultSchema) Module(name string) (Module, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	mod, ok := s.modules[name]
	return mod, ok
}

// Modules implements Schema interface
func (s *DefaultSchema) Modules() []Module {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Module, 0, len(s.modules))
	for _, m := range s.modules {
		result = append(result, m)
	}
	return result
}

// AddModule adds a module to the schema
func (s *DefaultSchema) AddModule(m Module) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.modules[m.Name()] = m
	// Build path cache by traversing the tree
	s.buildPathCacheLocked(m.Root())
}

// buildPathCacheLocked builds the path cache by traversing the tree
func (s *DefaultSchema) buildPathCacheLocked(node Node) {
	s.pathCache[node.Path()] = node
	switch n := node.(type) {
	case ContainerNode:
		for _, child := range n.Children() {
			s.buildPathCacheLocked(child)
		}
	case ListNode:
		for _, child := range n.Children() {
			s.buildPathCacheLocked(child)
		}
	case ChoiceNode:
		for _, c := range n.Cases() {
			for _, child := range c.Children() {
				s.buildPathCacheLocked(child)
			}
		}
	}
}

// Path implements Schema interface
func (s *DefaultSchema) Path(path string) (Node, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Normalize path
	if path == "" {
		return nil, false
	}
	if path[0] != '/' {
		path = "/" + path
	}
	node, ok := s.pathCache[path]
	return node, ok
}

// Validate implements Schema interface
func (s *DefaultSchema) Validate(path string, config interface{}) error {
	_, ok := s.Path(path)
	if !ok {
		return fmt.Errorf("schema: path %q not found in schema", path)
	}
	// TODO: Implement validation against schema
	// This will be implemented in a later iteration
	return nil
}

// defaultNode is the base implementation of Node
type defaultNode struct {
	name        string
	description string
	path        string
	schemaPath  string
	nodeType    NodeType
	parent      Node
}

// Name implements Node interface
func (n *defaultNode) Name() string {
	return n.name
}

// Description implements Node interface
func (n *defaultNode) Description() string {
	return n.description
}

// Path implements Node interface
func (n *defaultNode) Path() string {
	return n.path
}

// Type implements Node interface
func (n *defaultNode) Type() NodeType {
	return n.nodeType
}

// Parent implements Node interface
func (n *defaultNode) Parent() Node {
	return n.parent
}

// SchemaPath implements Node interface
func (n *defaultNode) SchemaPath() string {
	return n.schemaPath
}

// defaultModule is the default implementation of Module
type defaultModule struct {
	defaultNode
	name      string
	namespace string
	revision  string
	vendor    string
	root      ContainerNode
}

// Vendor implements Module interface
func (m *defaultModule) Vendor() string {
	return m.vendor
}

// Name implements Module interface
func (m *defaultModule) Name() string {
	return m.name
}

// Namespace implements Module interface
func (m *defaultModule) Namespace() string {
	return m.namespace
}

// Revision implements Module interface
func (m *defaultModule) Revision() string {
	return m.revision
}

// Root implements Module interface
func (m *defaultModule) Root() ContainerNode {
	return m.root
}

// NewModule creates a new module
func NewModule(name, namespace, revision string, root ContainerNode) Module {
	return &defaultModule{
		defaultNode: defaultNode{
			name:     name,
			path:     "/",
			nodeType: ContainerNodeType,
		},
		name:      name,
		namespace: namespace,
		revision:  revision,
		root:      root,
	}
}

// defaultContainer is the default implementation of ContainerNode
type defaultContainer struct {
	defaultNode
	children    []Node
	childrenMap map[string]Node
	isPresence  bool
	whenExpr    string
	mustExprs   []string
	opExcludes  []string
}

// Children implements ContainerNode interface
func (c *defaultContainer) Children() []Node {
	return c.children
}

// Child implements ContainerNode interface
func (c *defaultContainer) Child(name string) (Node, bool) {
	node, ok := c.childrenMap[name]
	return node, ok
}

// IsPresence implements ContainerNode interface
func (c *defaultContainer) IsPresence() bool {
	return c.isPresence
}

// WhenExpr implements ContainerNode interface
func (c *defaultContainer) WhenExpr() string {
	return c.whenExpr
}

// MustExprs implements ContainerNode interface
func (c *defaultContainer) MustExprs() []string {
	return c.mustExprs
}

// OperationExcludes implements ContainerNode interface
func (c *defaultContainer) OperationExcludes() []string {
	return c.opExcludes
}

// NewContainer creates a new container node
func NewContainer(name, description, path string, parent Node, isPresence bool) ContainerNode {
	c := &defaultContainer{
		defaultNode: defaultNode{
			name:        name,
			description: description,
			path:        path,
			nodeType:    ContainerNodeType,
			parent:      parent,
		},
		isPresence:  isPresence,
		childrenMap: make(map[string]Node),
	}
	c.children = make([]Node, 0)
	return c
}

// AddChild adds a child to the container
func (c *defaultContainer) AddChild(child Node) {
	c.children = append(c.children, child)
	c.childrenMap[child.Name()] = child
}

// defaultList is the default implementation of ListNode
type defaultList struct {
	defaultNode
	keys          []LeafNode
	children      []Node
	childrenMap   map[string]Node
	isUserOrdered bool
	whenExpr      string
	mustExprs     []string
	opExcludes    []string
}

// Keys implements ListNode interface
func (l *defaultList) Keys() []LeafNode {
	return l.keys
}

// Children implements ListNode interface
func (l *defaultList) Children() []Node {
	return l.children
}

// Child implements ListNode interface
func (l *defaultList) Child(name string) (Node, bool) {
	node, ok := l.childrenMap[name]
	return node, ok
}

// IsUserOrdered implements ListNode interface
func (l *defaultList) IsUserOrdered() bool {
	return l.isUserOrdered
}

// NewList creates a new list node
func NewList(name, description, path string, parent Node, keys []LeafNode, isUserOrdered bool) ListNode {
	l := &defaultList{
		defaultNode: defaultNode{
			name:        name,
			description: description,
			path:        path,
			nodeType:    ListNodeType,
			parent:      parent,
		},
		keys:          keys,
		isUserOrdered: isUserOrdered,
		childrenMap:   make(map[string]Node),
	}
	l.children = make([]Node, 0)
	return l
}

// AddChild adds a child to the list
func (l *defaultList) AddChild(child Node) {
	l.children = append(l.children, child)
	l.childrenMap[child.Name()] = child
}

// IsPresence implements ContainerNode interface for defaultList when used as parent
func (l *defaultList) IsPresence() bool {
	return false
}

// WhenExpr implements ContainerNode interface for defaultList when used as parent
func (l *defaultList) WhenExpr() string {
	return l.whenExpr
}

// MustExprs implements ContainerNode interface for defaultList when used as parent
func (l *defaultList) MustExprs() []string {
	return l.mustExprs
}

// OperationExcludes implements ListNode/ContainerNode interfaces
func (l *defaultList) OperationExcludes() []string {
	return l.opExcludes
}

// defaultChoice is the default implementation of ChoiceNode. A choice groups
// mutually-exclusive cases; it is schema-only and contributes no data-path segment
// (its members carry the enclosing container/list path — see entry.go).
type defaultChoice struct {
	defaultNode
	cases       []CaseNode
	casesMap    map[string]CaseNode
	defaultCase string
}

// Cases implements ChoiceNode interface
func (c *defaultChoice) Cases() []CaseNode {
	return c.cases
}

// Case implements ChoiceNode interface
func (c *defaultChoice) Case(name string) (CaseNode, bool) {
	n, ok := c.casesMap[name]
	return n, ok
}

// DefaultCase implements ChoiceNode interface
func (c *defaultChoice) DefaultCase() string {
	return c.defaultCase
}

// AddCase adds a case to the choice
func (c *defaultChoice) AddCase(cs CaseNode) {
	c.cases = append(c.cases, cs)
	c.casesMap[cs.Name()] = cs
}

// NewChoice creates a new choice node
func NewChoice(name, description, path string, parent Node) ChoiceNode {
	return &defaultChoice{
		defaultNode: defaultNode{
			name:        name,
			description: description,
			path:        path,
			nodeType:    ChoiceNodeType,
			parent:      parent,
		},
		casesMap: make(map[string]CaseNode),
	}
}

// defaultCase is the default implementation of CaseNode. Like choice it is
// schema-only; its children carry flat data paths (no case segment).
type defaultCase struct {
	defaultNode
	children    []Node
	childrenMap map[string]Node
}

// Children implements CaseNode interface
func (c *defaultCase) Children() []Node {
	return c.children
}

// Child implements CaseNode interface
func (c *defaultCase) Child(name string) (Node, bool) {
	node, ok := c.childrenMap[name]
	return node, ok
}

// AddChild adds a child to the case
func (c *defaultCase) AddChild(child Node) {
	c.children = append(c.children, child)
	c.childrenMap[child.Name()] = child
}

// NewCase creates a new case node
func NewCase(name, description, path string, parent Node) CaseNode {
	return &defaultCase{
		defaultNode: defaultNode{
			name:        name,
			description: description,
			path:        path,
			nodeType:    CaseNodeType,
			parent:      parent,
		},
		childrenMap: make(map[string]Node),
	}
}

// defaultLeaf is the default implementation of LeafNode
type defaultLeaf struct {
	defaultNode
	leafType      LeafType
	isKey         bool
	defaultValue  interface{}
	enumValues    []string
	mandatory     bool
	units         string
	whenExpr      string
	mustExprs     []string
	pattern       string
	rangeMin      int
	rangeMax      int
	hasMin        bool
	hasMax        bool
	leafList      bool
	supportFilter bool
	opExcludes    []string
}

// SupportFilter implements LeafNode interface
func (l *defaultLeaf) SupportFilter() bool {
	return l.supportFilter
}

// OperationExcludes implements LeafNode interface
func (l *defaultLeaf) OperationExcludes() []string {
	return l.opExcludes
}

// LeafType implements LeafNode interface
func (l *defaultLeaf) LeafType() LeafType {
	return l.leafType
}

// IsKey implements LeafNode interface
func (l *defaultLeaf) IsKey() bool {
	return l.isKey
}

// DefaultValue implements LeafNode interface
func (l *defaultLeaf) DefaultValue() interface{} {
	return l.defaultValue
}

// EnumValues implements LeafNode interface
func (l *defaultLeaf) EnumValues() []string {
	return l.enumValues
}

// Mandatory implements LeafNode interface
func (l *defaultLeaf) Mandatory() bool {
	return l.mandatory
}

// Units implements LeafNode interface
func (l *defaultLeaf) Units() string {
	return l.units
}

// WhenExpr implements LeafNode interface
func (l *defaultLeaf) WhenExpr() string {
	return l.whenExpr
}

// SetWhenExpr sets the leaf's YANG `when` XPath expression.
func (l *defaultLeaf) SetWhenExpr(expr string) {
	l.whenExpr = expr
}

// MustExprs implements LeafNode interface
func (l *defaultLeaf) MustExprs() []string {
	return l.mustExprs
}

// SetMustExprs sets the leaf's YANG `must` XPath expressions.
func (l *defaultLeaf) SetMustExprs(exprs []string) {
	l.mustExprs = exprs
}

// Pattern implements LeafNode interface
func (l *defaultLeaf) Pattern() string {
	return l.pattern
}

// RangeMin implements LeafNode interface
func (l *defaultLeaf) RangeMin() (int, bool) {
	return l.rangeMin, l.hasMin
}

// RangeMax implements LeafNode interface
func (l *defaultLeaf) RangeMax() (int, bool) {
	return l.rangeMax, l.hasMax
}

// IsLeafList implements LeafNode interface
func (l *defaultLeaf) IsLeafList() bool {
	return l.leafList
}

// SetLeafList marks this leaf as a leaf-list (repeatable scalar).
func (l *defaultLeaf) SetLeafList(v bool) {
	l.leafList = v
}

// NewLeaf creates a new leaf node
func NewLeaf(name, description, path string, parent Node, leafType LeafType, isKey bool, mandatory bool) LeafNode {
	return &defaultLeaf{
		defaultNode: defaultNode{
			name:        name,
			description: description,
			path:        path,
			nodeType:    LeafNodeType,
			parent:      parent,
		},
		leafType:   leafType,
		isKey:      isKey,
		mandatory:  mandatory,
		enumValues: []string{},
	}
}

// SetDefault sets the default value for the leaf
func (l *defaultLeaf) SetDefault(def interface{}) {
	l.defaultValue = def
}

// SetEnumValues sets the enumeration values
func (l *defaultLeaf) SetEnumValues(values []string) {
	l.enumValues = values
}

// SetUnits sets the units string
func (l *defaultLeaf) SetUnits(units string) {
	l.units = units
}
