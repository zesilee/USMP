package schema

import (
	"fmt"
	"strings"
)

// PathResolver resolves YANG paths against the schema tree
type PathResolver struct {
	schema Schema
}

// NewPathResolver creates a new PathResolver
func NewPathResolver(schema Schema) *PathResolver {
	return &PathResolver{
		schema: schema,
	}
}

// Resolve resolves a YANG path string to a schema node
// Supports paths like: "/interfaces/interface[name='eth0']/config/description"
// Returns the node and whether it was found
func (r *PathResolver) Resolve(path string) (Node, bool) {
	// Fast path: check cache in schema first
	if node, ok := r.schema.Path(path); ok {
		return node, ok
	}

	components := SplitPath(path)
	if len(components) == 0 {
		return nil, false
	}

	// Start from root of first module
	var current Node
	modules := r.schema.Modules()
	if len(modules) == 0 {
		return nil, false
	}

	// First component is module namespace/name
	// Find module with matching name
	for _, m := range modules {
		if m.Name() == components[0] {
			current = m.Root()
			break
		}
	}

	if current == nil {
		// Try first module root
		current = modules[0].Root()
	}

	if len(components) == 1 {
		return current, true
	}

	// Resolve remaining components
	for _, comp := range components[1:] {
		// Strip list key predicate if present: "name='eth0'" → "name"
		name := comp
		if strings.Contains(comp, "[") {
			name = comp[:strings.Index(comp, "[")]
		}

		var child Node
		switch n := current.(type) {
		case ContainerNode:
			child, _ = n.Child(name)
		case ListNode:
			child, _ = n.Child(name)
		case ChoiceNode:
			// For choice, look in all cases
			for _, c := range n.Cases() {
				if child, _ = c.Child(name); child != nil {
					break
				}
			}
		default:
			// Leaf node can't have children
			return nil, false
		}

		if child == nil {
			return nil, false
		}
		current = child
	}

	return current, current != nil
}

// GetParentPath returns the parent path for a given path
func GetParentPath(path string) string {
	if path == "/" {
		return ""
	}
	components := SplitPath(path)
	if len(components) <= 1 {
		return "/"
	}
	return JoinPath(components[:len(components)-1])
}

// GetLastComponent returns the last component of the path
func GetLastComponent(path string) string {
	if path == "/" {
		return ""
	}
	components := SplitPath(path)
	if len(components) == 0 {
		return ""
	}
	return components[len(components)-1]
}

// ParseListKey parses a list key predicate like "[name='eth0']"
// Returns key name and value
func ParseListKey(predicate string) (string, string, error) {
	// predicate format: "name='value'" or "name = \"value\""
	predicate = strings.Trim(predicate, "[]")
	parts := strings.SplitN(predicate, "=", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid list key predicate: %s", predicate)
	}
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	// Strip quotes
	value = strings.Trim(value, "'\"")
	return key, value, nil
}

// IsListEntryPath returns true if the path contains a list entry with a key
func IsListEntryPath(path string) bool {
	return strings.Contains(path, "[") && strings.Contains(path, "]")
}

// PathWithoutKeys returns the path with list key predicates removed
// Example: "/interfaces/interface[name=eth0]/config" → "/interfaces/interface/config"
func PathWithoutKeys(path string) string {
	if !strings.Contains(path, "[") {
		return path
	}
	var result strings.Builder
	inBracket := false
	for _, c := range path {
		if c == '[' {
			inBracket = true
			continue
		}
		if c == ']' {
			inBracket = false
			continue
		}
		if !inBracket {
			result.WriteRune(c)
		}
	}
	// Clean up double slashes
	cleanPath := strings.ReplaceAll(result.String(), "//", "/")
	return cleanPath
}
