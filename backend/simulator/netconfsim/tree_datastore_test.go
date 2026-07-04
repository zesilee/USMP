package netconfsim

import (
	"sync"
	"testing"
)

const (
	sampleVlan100 = `<vlan xmlns="urn:huawei:vlan"><vlans><vlan><id>100</id><name>office</name></vlan></vlans></vlan>`
	sampleVlan200 = `<vlan xmlns="urn:huawei:vlan"><vlans><vlan><id>200</id><name>guest</name></vlan></vlans></vlan>`
)

// treeFromXML is a test helper.
func treeFromXML(t *testing.T, s string) *dataNode {
	t.Helper()
	n, err := parseXML([]byte(s))
	if err != nil {
		t.Fatalf("parseXML: %v", err)
	}
	return n
}

func TestTreeDatastoreSetCandidateCommit(t *testing.T) {
	ds := newTreeDatastore()

	// running starts empty
	if got := treeFromXML(t, string(ds.GetRunning())); len(got.Children) != 0 {
		t.Fatalf("running should start empty, got %s", ds.GetRunning())
	}

	if err := ds.SetCandidate([]byte(sampleVlan100)); err != nil {
		t.Fatalf("SetCandidate: %v", err)
	}
	// candidate reflects the write, running still empty (not committed)
	if !nodesEqual(treeFromXML(t, string(ds.GetCandidate())), treeFromXML(t, sampleVlan100)) {
		t.Fatalf("candidate mismatch: %s", ds.GetCandidate())
	}
	if got := treeFromXML(t, string(ds.GetRunning())); len(got.Children) != 0 {
		t.Fatalf("running should still be empty before commit, got %s", ds.GetRunning())
	}

	if err := ds.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if !nodesEqual(treeFromXML(t, string(ds.GetRunning())), treeFromXML(t, sampleVlan100)) {
		t.Fatalf("running mismatch after commit: %s", ds.GetRunning())
	}
}

func TestTreeDatastoreDiscard(t *testing.T) {
	ds := newTreeDatastore()
	if err := ds.SetRunning([]byte(sampleVlan100)); err != nil {
		t.Fatal(err)
	}
	// change candidate, then discard -> candidate reverts to running
	if err := ds.SetCandidate([]byte(sampleVlan200)); err != nil {
		t.Fatal(err)
	}
	ds.DiscardCandidate()
	if !nodesEqual(treeFromXML(t, string(ds.GetCandidate())), treeFromXML(t, sampleVlan100)) {
		t.Fatalf("candidate should revert to running after discard: %s", ds.GetCandidate())
	}
}

func TestTreeDatastoreCommitIsolation(t *testing.T) {
	// Committing then discarding a subsequent candidate change must not corrupt
	// the committed running tree (deep-copy on commit).
	ds := newTreeDatastore()
	if err := ds.SetCandidate([]byte(sampleVlan100)); err != nil {
		t.Fatal(err)
	}
	if err := ds.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := ds.SetCandidate([]byte(sampleVlan200)); err != nil {
		t.Fatal(err)
	}
	// running must remain vlan100
	if !nodesEqual(treeFromXML(t, string(ds.GetRunning())), treeFromXML(t, sampleVlan100)) {
		t.Fatalf("running leaked candidate mutation: %s", ds.GetRunning())
	}
}

func TestTreeDatastoreSetCandidateError(t *testing.T) {
	ds := newTreeDatastore()
	if err := ds.SetCandidate([]byte(`<vlan><bad></vlan>`)); err == nil {
		t.Fatal("expected error for malformed candidate XML")
	}
}

// TestTreeDatastoreConcurrent exercises the RWMutex under -race.
func TestTreeDatastoreConcurrent(t *testing.T) {
	ds := newTreeDatastore()
	_ = ds.SetRunning([]byte(sampleVlan100))

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(3)
		go func() { defer wg.Done(); _ = ds.SetCandidate([]byte(sampleVlan200)) }()
		go func() { defer wg.Done(); _ = ds.Commit() }()
		go func() { defer wg.Done(); _ = ds.GetRunning() }()
	}
	wg.Wait()
}
