package diff

// ChangeType represents the type of configuration change
type ChangeType int

const (
	// AddChange indicates a new node was added
	AddChange ChangeType = iota
	// DeleteChange indicates a node was removed
	DeleteChange
	// ModifyChange indicates a leaf value was modified
	ModifyChange
)

// String returns the string representation of ChangeType
func (t ChangeType) String() string {
	switch t {
	case AddChange:
		return "ADD"
	case DeleteChange:
		return "DELETE"
	case ModifyChange:
		return "MODIFY"
	default:
		return "UNKNOWN"
	}
}

// Change represents a single configuration change between desired and actual
type Change struct {
	// Type is the change type
	Type ChangeType
	// Path is the YANG path to the changed node
	Path string
	// OldValue is the actual value before change
	OldValue interface{}
	// NewValue is the desired value after change
	NewValue interface{}
	// SchemaPath is the path in the schema
	SchemaPath string
}

// DiffResult contains all changes between desired and actual
type DiffResult struct {
	// Changes is the list of changes
	Changes []Change
	// Summary provides statistics about changes
	Summary DiffSummary
}

// DiffSummary summarizes the number of changes by type
type DiffSummary struct {
	// Adds is the number of add changes
	Adds int
	// Deletes is the number of delete changes
	Deletes int
	// Modifies is the number of modify changes
	Modifies int
	// Total is the total number of changes
	Total int
}

// NewDiffResult creates a new DiffResult
func NewDiffResult() *DiffResult {
	return &DiffResult{
		Changes: make([]Change, 0),
	}
}

// AddChange adds a change to the result
func (dr *DiffResult) AddChange(c Change) {
	dr.Changes = append(dr.Changes, c)
	switch c.Type {
	case AddChange:
		dr.Summary.Adds++
	case DeleteChange:
		dr.Summary.Deletes++
	case ModifyChange:
		dr.Summary.Modifies++
	}
	dr.Summary.Total++
}
