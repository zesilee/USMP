package netconfsim

import (
	"strings"
	"sync"
)

// treeDatastore is the structured, model-agnostic replacement for the legacy
// blob Datastore. running/candidate are generic XML data trees (*dataNode)
// instead of opaque []byte, which is the foundation for real edit-config
// merge/delete (T3) and get-config subtree filtering (T4).
//
// It intentionally mirrors the legacy Datastore's method surface
// (SetCandidate/GetRunning/GetCandidate/Commit/DiscardCandidate) so it can be
// swapped into the server in a later step (T5) after dual-path validation.
type treeDatastore struct {
	mu        sync.RWMutex
	running   *dataNode
	candidate *dataNode
}

// newTreeDatastore returns an empty tree datastore (running and candidate are
// empty synthetic roots).
func newTreeDatastore() *treeDatastore {
	return &treeDatastore{
		running:   &dataNode{},
		candidate: &dataNode{},
	}
}

// SetCandidate parses the given config XML and replaces the candidate tree.
// (Whole-tree replace for T2; per-operation merge/delete arrives in T3.)
func (d *treeDatastore) SetCandidate(xmlBytes []byte) error {
	node, err := parseXML(xmlBytes)
	if err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.candidate = node
	return nil
}

// EditConfig applies an edit-config <config> subtree to the candidate tree using
// per-node operation semantics (merge/replace/create/delete/remove), instead of
// the whole-tree replace that SetCandidate performs. Errors (malformed XML,
// create-on-existing, delete-of-missing) leave the candidate unchanged.
func (d *treeDatastore) EditConfig(xmlBytes []byte) error {
	edit, err := parseXML(xmlBytes)
	if err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	// Apply to a working copy so a mid-way error cannot leave a partial edit.
	next := d.candidate.clone()
	if err := next.applyEdit(edit); err != nil {
		return err
	}
	d.candidate = next
	return nil
}

// SetRunning parses the given config XML and replaces the running tree
// (candidate is reset to match). Used to seed initial device state.
func (d *treeDatastore) SetRunning(xmlBytes []byte) error {
	node, err := parseXML(xmlBytes)
	if err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.running = node
	d.candidate = node.clone()
	return nil
}

// Commit copies the candidate tree onto running.
func (d *treeDatastore) Commit() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.running = d.candidate.clone()
	return nil
}

// DiscardCandidate resets the candidate tree to match running.
func (d *treeDatastore) DiscardCandidate() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.candidate = d.running.clone()
}

// GetRunning serializes the running tree to XML.
func (d *treeDatastore) GetRunning() []byte {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.running.xmlBytes()
}

// GetConfigFiltered serializes the running tree after applying a subtree filter.
// An empty/nil filter returns the whole running config. filterXML is the inner
// content of a NETCONF <filter> element (its top-level filter nodes).
func (d *treeDatastore) GetConfigFiltered(filterXML []byte) ([]byte, error) {
	if len(strings.TrimSpace(string(filterXML))) == 0 {
		return d.GetRunning(), nil
	}
	filter, err := parseXML(filterXML)
	if err != nil {
		return nil, err
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	return filterTree(d.running, filter).xmlBytes(), nil
}

// GetCandidate serializes the candidate tree to XML.
func (d *treeDatastore) GetCandidate() []byte {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.candidate.xmlBytes()
}

// runningTree returns the running tree for structured assertions/queries.
// Callers must not mutate the returned tree.
func (d *treeDatastore) runningTree() *dataNode {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.running
}

// candidateTree returns the candidate tree for structured assertions/queries.
// Callers must not mutate the returned tree.
func (d *treeDatastore) candidateTree() *dataNode {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.candidate
}
