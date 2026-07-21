package netconfsim

// State overlay merging for the <get> RPC (NS-08). The state tree holds
// config-false data injected via SetState; <get> responds with running+state
// merged, while <get-config> keeps returning the running tree only.
//
// Semantics mirror edit-config merge (same findMatch/wellKnownListKeys
// machinery) with one tightening: a keyed list entry whose key finds no config
// match is dropped instead of created — state for a deleted config entry must
// not resurface as a ghost entry. Pure state nodes (containers/leaves/lists
// with no config counterpart at all) are appended as-is, and a leaf present in
// both trees takes the state value.

// mergeState merges the state overlay into target, a private clone of running.
func mergeState(target, state *dataNode) {
	for _, sc := range state.Children {
		mergeStateNode(target, sc, countSameName(state.Children, sc))
	}
}

// mergeStateNode merges a single state element sc into parent. stateSiblings is
// how many same-named siblings sc has on the state side (list signal, as in
// applyNode).
func mergeStateNode(parent, sc *dataNode, stateSiblings int) {
	match := findMatch(parent, sc, opMerge, stateSiblings)
	if match == nil {
		// findMatch misses with same-named config children only on a keyed-list
		// key mismatch — drop, no ghost entries.
		if len(sameNameChildren(parent, sc)) > 0 {
			return
		}
		// Registered config-backed list with zero remaining config entries:
		// the key-carrying state entry is equally a ghost.
		if key := wellKnownListKey(parent, sc); key != "" && sc.child(key) != nil {
			return
		}
		parent.Children = append(parent.Children, sc.clone())
		return
	}
	if sc.isLeaf() {
		match.Text = sc.Text
		return
	}
	for _, gc := range sc.Children {
		mergeStateNode(match, gc, countSameName(sc.Children, gc))
	}
}
