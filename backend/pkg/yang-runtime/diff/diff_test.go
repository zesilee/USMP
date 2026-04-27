package diff

import (
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
	"github.com/stretchr/testify/assert"
)

// Test struct definitions for our test cases
type TestInterface struct {
	Interface []TestInterfaceEntry `yang:"interfaces"`
}

type TestInterfaceEntry struct {
	Name        string `yang:"name"`
	Enabled     bool   `yang:"enabled"`
	Description string `yang:"description"`
	MTU         int    `yang:"mtu"`
}

type TestVLAN struct {
	VLAN []TestVLANEntry `yang:"vlans"`
}

type TestVLANEntry struct {
	ID   int    `yang:"id"`
	Name string `yang:"name"`
}

func TestDiffNoChanges(t *testing.T) {
	de := NewDefaultDiffEngine()

	desired := &TestInterface{
		Interface: []TestInterfaceEntry{
			{Name: "eth0", Enabled: true, Description: "Uplink", MTU: 1500},
		},
	}
	actual := &TestInterface{
		Interface: []TestInterfaceEntry{
			{Name: "eth0", Enabled: true, Description: "Uplink", MTU: 1500},
		},
	}

	s := schema.NewSchema()
	result, err := de.Diff(desired, actual, s)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.Summary.Total)
	assert.Empty(t, result.Changes)
}

func TestDiffLeafModify(t *testing.T) {
	de := NewDefaultDiffEngine()

	desired := &TestInterfaceEntry{Name: "eth0", Enabled: true, Description: "New Description", MTU: 1500}
	actual := &TestInterfaceEntry{Name: "eth0", Enabled: true, Description: "Old Description", MTU: 1500}

	s := schema.NewSchema()
	result, err := de.Diff(desired, actual, s)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Summary.Total)
	assert.Equal(t, 0, result.Summary.Adds)
	assert.Equal(t, 0, result.Summary.Deletes)
	assert.Equal(t, 1, result.Summary.Modifies)

	change := result.Changes[0]
	assert.Equal(t, ModifyChange, change.Type)
	assert.Equal(t, "Description", change.Path)
}

func TestDiffAddLeaf(t *testing.T) {
	de := NewDefaultDiffEngine()

	desired := &TestInterfaceEntry{Name: "eth0", Enabled: true, Description: "Uplink", MTU: 1500}
	actual := &TestInterfaceEntry{Name: "eth0", Enabled: true}

	s := schema.NewSchema()
	result, err := de.Diff(desired, actual, s)
	assert.NoError(t, err)
	// Description and MTU are both added
	assert.Equal(t, 2, result.Summary.Total)
}

func TestDiffDeleteWhole(t *testing.T) {
	de := NewDefaultDiffEngine()

	var desired *TestInterfaceEntry
	actual := &TestInterfaceEntry{Name: "eth0", Enabled: true}

	s := schema.NewSchema()
	result, err := de.Diff(desired, actual, s)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Summary.Total)
	assert.Equal(t, 1, result.Summary.Deletes)
}

func TestDiffAddWhole(t *testing.T) {
	de := NewDefaultDiffEngine()

	desired := &TestInterfaceEntry{Name: "eth0", Enabled: true}
	var actual *TestInterfaceEntry

	s := schema.NewSchema()
	result, err := de.Diff(desired, actual, s)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Summary.Total)
	assert.Equal(t, 1, result.Summary.Adds)
}

func TestDiffListAddEntry(t *testing.T) {
	de := NewDefaultDiffEngine()

	desired := &TestVLAN{
		VLAN: []TestVLANEntry{
			{ID: 100, Name: "vlan100"},
			{ID: 200, Name: "vlan200"},
		},
	}
	actual := &TestVLAN{
		VLAN: []TestVLANEntry{
			{ID: 100, Name: "vlan100"},
		},
	}

	s := schema.NewSchema()
	result, err := de.Diff(desired, actual, s)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Summary.Total)
	assert.Equal(t, 1, result.Summary.Adds)

	change := result.Changes[0]
	assert.Equal(t, AddChange, change.Type)
	assert.Contains(t, change.Path, "[ID=200]")
}

func TestDiffListDeleteEntry(t *testing.T) {
	de := NewDefaultDiffEngine()

	desired := &TestVLAN{
		VLAN: []TestVLANEntry{
			{ID: 100, Name: "vlan100"},
		},
	}
	actual := &TestVLAN{
		VLAN: []TestVLANEntry{
			{ID: 100, Name: "vlan100"},
			{ID: 200, Name: "vlan200"},
		},
	}

	s := schema.NewSchema()
	result, err := de.Diff(desired, actual, s)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Summary.Total)
	assert.Equal(t, 1, result.Summary.Deletes)
}

func TestDiffListModifyEntry(t *testing.T) {
	de := NewDefaultDiffEngine()

	desired := &TestVLAN{
		VLAN: []TestVLANEntry{
			{ID: 100, Name: "new-name"},
			{ID: 200, Name: "vlan200"},
		},
	}
	actual := &TestVLAN{
		VLAN: []TestVLANEntry{
			{ID: 100, Name: "old-name"},
			{ID: 200, Name: "vlan200"},
		},
	}

	s := schema.NewSchema()
	result, err := de.Diff(desired, actual, s)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Summary.Total)
	assert.Equal(t, 1, result.Summary.Modifies)
}

type testNested struct {
	Config testNestedConfig `yang:"config"`
}
type testNestedConfig struct {
	Description string `yang:"description"`
	Enabled     bool   `yang:"enabled"`
}

func TestDiffNestedContainer(t *testing.T) {
	desired := &testNested{
		Config: testNestedConfig{
			Description: "new",
			Enabled:     true,
		},
	}
	actual := &testNested{
		Config: testNestedConfig{
			Description: "old",
			Enabled:     true,
		},
	}

	de := NewDefaultDiffEngine()
	s := schema.NewSchema()
	result, err := de.Diff(desired, actual, s)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Summary.Total)
	assert.Equal(t, "Config/Description", result.Changes[0].Path)
}

func TestDiffPruneRedundant(t *testing.T) {
	de := NewDefaultDiffEngine().WithPruneRedundant(true)

	desired := &TestInterface{
		Interface: []TestInterfaceEntry{
			{Name: "eth0", Enabled: true, Description: "New", MTU: 9000},
		},
	}
	actual := &TestInterface{}

	s := schema.NewSchema()
	result, err := de.Diff(desired, actual, s)
	assert.NoError(t, err)
	// With pruning, we only get one add for the entire entry, not for each leaf
	// Because the whole entry is added, all child changes are redundant
	assert.Equal(t, 1, result.Summary.Total)
}

func TestDiffNoPrune(t *testing.T) {
	de := NewDefaultDiffEngine().WithPruneRedundant(false)

	// Test that without pruning, multiple changes are kept when parent is not changed
	// This tests that pruning only happens when requested
	type TestNested struct {
		Enabled bool
		Name    string
		Value   int
	}
	desired := &TestNested{
		Enabled: true,
		Name:    "new",
		Value:   42,
	}
	actual := &TestNested{
		Enabled: false,
		Name:    "old",
		Value:   10,
	}

	s := schema.NewSchema()
	result, err := de.Diff(desired, actual, s)
	assert.NoError(t, err)
	// Without pruning, all three modified leaves get changes
	assert.Equal(t, 3, result.Summary.Total)
}

func TestDiffMultipleChanges(t *testing.T) {
	de := NewDefaultDiffEngine()

	desired := &TestVLAN{
		VLAN: []TestVLANEntry{
			{ID: 100, Name: "vlan-100"},    // modified
			{ID: 200, Name: "new-vlan-200"}, // added
		},
	}
	actual := &TestVLAN{
		VLAN: []TestVLANEntry{
			{ID: 100, Name: "old-100"},  // modified
			{ID: 300, Name: "vlan-300"},  // deleted
		},
	}

	s := schema.NewSchema()
	result, err := de.Diff(desired, actual, s)
	assert.NoError(t, err)
	assert.Equal(t, 3, result.Summary.Total)
	assert.Equal(t, 1, result.Summary.Adds)
	assert.Equal(t, 1, result.Summary.Deletes)
	assert.Equal(t, 1, result.Summary.Modifies)
}
