package netconfsim

import "fmt"

// edit-config operation semantics (RFC 6241 §7.2) applied to the generic XML
// data tree. Because the simulator is schema-less, list-entry matching is
// heuristic: an element is treated as a keyed list entry (matched by its first
// leaf child, the YANG-conventional key) when the operation is explicit
// (create/delete/remove/replace) or when several same-named siblings exist;
// otherwise it is treated as a single container and merged in place. This is
// sufficient for a test double and covers the Huawei/OpenConfig models in use.
const (
	opMerge   = "merge"
	opReplace = "replace"
	opCreate  = "create"
	opDelete  = "delete"
	opRemove  = "remove"
)

// applyEdit applies the parsed <config> subtree (edit) onto target, which is the
// candidate root. Each top-level element of edit is processed with the default
// "merge" operation unless it carries an explicit operation attribute.
func (target *dataNode) applyEdit(edit *dataNode) error {
	for _, ec := range edit.Children {
		if err := applyNode(target, ec, opMerge, countSameName(edit.Children, ec)); err != nil {
			return err
		}
	}
	return nil
}

// countSameName returns how many of siblings share ec's qualified name — the
// "edit-side list" signal for findMatch.
func countSameName(siblings []*dataNode, ec *dataNode) int {
	n := 0
	for _, c := range siblings {
		if c.Name.Local == ec.Name.Local && c.Name.Space == ec.Name.Space {
			n++
		}
	}
	return n
}

// applyNode applies a single edit element ec into parent according to its
// effective operation (its own operation attribute, else the inherited one).
// editSiblings is how many same-named siblings ec has in the edit itself —
// a second list signal beyond the store side (see findMatch).
func applyNode(parent, ec *dataNode, inherited string, editSiblings int) error {
	op := effectiveOp(ec, inherited)
	match := findMatch(parent, ec, op, editSiblings)

	switch op {
	case opDelete:
		if match == nil {
			return fmt.Errorf("edit-config delete: target %q not found (data-missing)", ec.Name.Local)
		}
		removeChild(parent, match)
	case opRemove:
		if match != nil {
			removeChild(parent, match)
		}
	case opCreate:
		if match != nil {
			return fmt.Errorf("edit-config create: target %q already exists (data-exists)", ec.Name.Local)
		}
		parent.Children = append(parent.Children, ec.cloneClean())
	case opReplace:
		if match != nil {
			removeChild(parent, match)
		}
		parent.Children = append(parent.Children, ec.cloneClean())
	default: // merge
		if match == nil {
			parent.Children = append(parent.Children, ec.cloneClean())
			return nil
		}
		if ec.isLeaf() {
			match.Text = ec.Text
			return nil
		}
		for _, gc := range ec.Children {
			if err := applyNode(match, gc, opMerge, countSameName(ec.Children, gc)); err != nil {
				return err
			}
		}
	}
	return nil
}

// effectiveOp returns the operation for ec: its own operation attribute if
// present, otherwise the inherited operation from its ancestor.
func effectiveOp(ec *dataNode, inherited string) string {
	if v := opAttr(ec); v != "" {
		return v
	}
	return inherited
}

// opAttr returns the value of the NETCONF operation attribute on n, matching any
// namespace prefix (nc:operation / xc:operation / operation). Empty if absent.
func opAttr(n *dataNode) string {
	for _, a := range n.Attrs {
		if a.Name.Local == "operation" {
			return a.Value
		}
	}
	return ""
}

// findMatch locates the child of parent that ec refers to. For explicit
// operations or genuine lists (multiple same-named siblings on either the store
// side or the edit side) it matches by the element name plus the key leaf;
// otherwise it matches the single same-named container. The edit-side signal
// (editSiblings > 1) is essential: merging a two-entry list batch into a store
// holding a single entry must key-match, not fold the new entry into the old
// one（config-delete-semantics 回归——此前被 server 整树替换掩盖）。
func findMatch(parent, ec *dataNode, op string, editSiblings int) *dataNode {
	same := sameNameChildren(parent, ec)
	if len(same) == 0 {
		return nil
	}
	key := keyLeaf(ec)
	needKey := key != nil && (op != opMerge || len(same) > 1 || editSiblings > 1)
	if !needKey {
		return same[0]
	}
	for _, cand := range same {
		if k := cand.child(key.Name.Local); k != nil && k.leafText() == key.leafText() {
			return cand
		}
	}
	return nil
}

// sameNameChildren returns parent's children whose qualified name (local +
// namespace) equals ec's.
func sameNameChildren(parent, ec *dataNode) []*dataNode {
	var out []*dataNode
	for _, c := range parent.Children {
		if c.Name.Local == ec.Name.Local && c.Name.Space == ec.Name.Space {
			out = append(out, c)
		}
	}
	return out
}

// keyLeaf returns ec's first leaf child, treated as the list key by convention
// (YANG list keys precede other nodes). Returns nil if ec has no leaf child.
func keyLeaf(ec *dataNode) *dataNode {
	for _, c := range ec.Children {
		if c.isLeaf() {
			return c
		}
	}
	return nil
}

// removeChild removes the first pointer-identical child from parent.
func removeChild(parent, child *dataNode) {
	for i, c := range parent.Children {
		if c == child {
			parent.Children = append(parent.Children[:i], parent.Children[i+1:]...)
			return
		}
	}
}

// cloneClean deep-copies n and strips NETCONF operation attributes from the copy
// so they never persist in the datastore.
func (n *dataNode) cloneClean() *dataNode {
	cp := n.clone()
	stripOperationAttrs(cp)
	return cp
}

func stripOperationAttrs(n *dataNode) {
	if len(n.Attrs) > 0 {
		kept := n.Attrs[:0]
		for _, a := range n.Attrs {
			if a.Name.Local == "operation" {
				continue
			}
			kept = append(kept, a)
		}
		if len(kept) == 0 {
			n.Attrs = nil
		} else {
			n.Attrs = kept
		}
	}
	for _, c := range n.Children {
		stripOperationAttrs(c)
	}
}
