package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSchema(t *testing.T) {
	s := NewSchema()
	assert.NotNil(t, s)
	assert.Empty(t, s.Modules())
}

func TestSchemaAddModule(t *testing.T) {
	s := NewSchema()
	root := NewContainer("root", "root container", "/", nil, false)
	module := NewModule("test-module", "http://example.com/test", "2024-01-01", root)
	s.AddModule(module)

	assert.Len(t, s.Modules(), 1)
	mod, ok := s.Module("test-module")
	assert.True(t, ok)
	assert.Equal(t, "test-module", mod.Name())
	assert.Equal(t, "http://example.com/test", mod.Namespace())
}

func TestNodeBasics(t *testing.T) {
	root := NewContainer("root", "root desc", "/", nil, false)
	assert.Equal(t, "root", root.Name())
	assert.Equal(t, "/", root.Path())
	assert.Equal(t, ContainerNodeType, root.Type())
	assert.Nil(t, root.Parent())
	assert.True(t, root.IsPresence() == false)
}

func TestContainerAddChild(t *testing.T) {
	root := NewContainer("root", "root desc", "/", nil, false)
	leaf := NewLeaf("name", "name leaf", "/name", root, LeafTypeString, false, false)
	root.(*defaultContainer).AddChild(leaf)

	children := root.Children()
	assert.Len(t, children, 1)

	child, ok := root.Child("name")
	assert.True(t, ok)
	assert.Equal(t, leaf, child)
}

func TestListWithKeys(t *testing.T) {
	root := NewContainer("interfaces", "interfaces container", "/interfaces", nil, false)
	nameLeaf := NewLeaf("name", "interface name", "/interfaces/interface/name", nil, LeafTypeString, true, true)
	keys := []LeafNode{nameLeaf}
	ifList := NewList("interface", "interface list", "/interfaces/interface", root, keys, false)
	root.(*defaultContainer).AddChild(ifList)

	assert.Equal(t, "interface", ifList.Name())
	assert.Equal(t, keys, ifList.Keys())
	assert.Len(t, ifList.Keys(), 1)
	assert.True(t, ifList.IsUserOrdered() == false)
}

func TestLeafBasics(t *testing.T) {
	parent := NewContainer("container", "parent", "/container", nil, false)
	leaf := NewLeaf("enabled", "enable flag", "/container/enabled", parent, LeafTypeBoolean, false, true)
	leaf.(*defaultLeaf).SetDefault(true)

	assert.Equal(t, "enabled", leaf.Name())
	assert.Equal(t, LeafTypeBoolean, leaf.LeafType())
	assert.True(t, leaf.Mandatory())
	assert.Equal(t, true, leaf.DefaultValue())
}

func TestLeafEnum(t *testing.T) {
	parent := NewContainer("container", "parent", "/container", nil, false)
	leaf := NewLeaf("type", "interface type", "/container/type", parent, LeafTypeEnum, false, false)
	leaf.(*defaultLeaf).SetEnumValues([]string{"ethernet", "loopback", "vlan"})

	assert.Equal(t, []string{"ethernet", "loopback", "vlan"}, leaf.EnumValues())
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		path     string
		expected []string
	}{
		{"/", []string{}},
		{"/interfaces", []string{"interfaces"}},
		{"/interfaces/interface", []string{"interfaces", "interface"}},
		{"interfaces/interface", []string{"interfaces", "interface"}},
		{"/interfaces/interface/config/description", []string{"interfaces", "interface", "config", "description"}},
	}

	for _, tt := range tests {
		result := SplitPath(tt.path)
		assert.Equal(t, tt.expected, result, "path: %q", tt.path)
	}
}

func TestJoinPath(t *testing.T) {
	tests := []struct {
		components []string
		expected    string
	}{
		{[]string{}, "/"},
		{[]string{"interfaces"}, "/interfaces"},
		{[]string{"interfaces", "interface"}, "/interfaces/interface"},
	}

	for _, tt := range tests {
		result := JoinPath(tt.components)
		assert.Equal(t, tt.expected, result)
	}
}

func TestGetParentPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/", ""},
		{"/interfaces", "/"},
		{"/interfaces/interface", "/interfaces"},
		{"/interfaces/interface/config", "/interfaces/interface"},
	}

	for _, tt := range tests {
		result := GetParentPath(tt.path)
		assert.Equal(t, tt.expected, result)
	}
}

func TestGetLastComponent(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/", ""},
		{"/interfaces", "interfaces"},
		{"/interfaces/interface", "interface"},
		{"/interfaces/interface/description", "description"},
	}

	for _, tt := range tests {
		result := GetLastComponent(tt.path)
		assert.Equal(t, tt.expected, result)
	}
}

func TestParseListKey(t *testing.T) {
	keyName, value, err := ParseListKey("[name='eth0']")
	assert.NoError(t, err)
	assert.Equal(t, "name", keyName)
	assert.Equal(t, "eth0", value)

	keyName, value, err = ParseListKey("[name = \"eth0\"]")
	assert.NoError(t, err)
	assert.Equal(t, "name", keyName)
	assert.Equal(t, "eth0", value)
}

func TestPathWithoutKeys(t *testing.T) {
	path := "/interfaces/interface[name=eth0]/config/description"
	result := PathWithoutKeys(path)
	assert.Equal(t, "/interfaces/interface/config/description", result)

	path = "/vlans/vlan[id=100]/config/name"
	result = PathWithoutKeys(path)
	assert.Equal(t, "/vlans/vlan/config/name", result)
}

func TestIsListEntryPath(t *testing.T) {
	assert.True(t, IsListEntryPath("/interfaces/interface[name=eth0]"))
	assert.False(t, IsListEntryPath("/interfaces/interface"))
}
