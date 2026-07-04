package netconfsim

import (
	"encoding/xml"
	"strings"
)

// get-config subtree filtering (RFC 6241 §6) on the generic XML data tree.
//
// Filter node kinds:
//   - selection node   : empty element -> selects the whole matching subtree.
//   - content match node: a leaf with text -> a predicate on a sibling set; the
//     containing element is included only if a same-named data leaf matches.
//   - containment node  : an element with element children -> recurse; the node
//     is included only if some descendant is selected.
//
// When a matched element's filter carries only content-match children (no
// selection/containment siblings), the whole matched subtree is returned, as in
// RFC 6241 §6.4.2.
//
// Namespaces on filter nodes are optional: a filter node with an empty namespace
// matches any namespace (client subtree/XPath filters routinely omit them).

// filterTree returns a new synthetic root holding the parts of data selected by
// filter. data and filter are both synthetic roots (their Children are the
// top-level elements).
func filterTree(data, filter *dataNode) *dataNode {
	res := &dataNode{}
	for _, ff := range filter.Children {
		for _, dn := range data.Children {
			if !nameMatch(dn, ff) {
				continue
			}
			if m := matchNode(dn, ff); m != nil {
				res.Children = append(res.Children, m)
			}
		}
	}
	return res
}

// matchNode filters a single data node dn against filter node ff, returning the
// selected subtree or nil if dn does not match.
func matchNode(dn, ff *dataNode) *dataNode {
	// Selection node: empty filter element selects the entire subtree.
	if len(ff.Children) == 0 && strings.TrimSpace(ff.Text) == "" {
		return dn.clone()
	}

	// Content-match leaf: predicate satisfied iff dn is a leaf with equal text.
	if ff.isLeaf() {
		if dn.isLeaf() && dn.leafText() == ff.leafText() {
			return dn.clone()
		}
		return nil
	}

	// Containment node: partition children into content matches and selectors.
	var contentMatches, selectors []*dataNode
	for _, c := range ff.Children {
		if c.isLeaf() && strings.TrimSpace(c.Text) != "" {
			contentMatches = append(contentMatches, c)
		} else {
			selectors = append(selectors, c)
		}
	}

	// Every content match must be satisfied by some same-named data leaf.
	for _, cm := range contentMatches {
		if matchedLeaf(dn, cm) == nil {
			return nil
		}
	}

	out := &dataNode{Name: dn.Name, Text: dn.Text}
	if len(dn.Attrs) > 0 {
		out.Attrs = append([]xml.Attr(nil), dn.Attrs...)
	}

	// No selection/containment siblings: return the whole matched subtree.
	if len(selectors) == 0 {
		for _, c := range dn.Children {
			out.Children = append(out.Children, c.clone())
		}
		return out
	}

	// Include the matched content-match leaves (typically list keys) so the
	// output identifies which entry was selected.
	for _, cm := range contentMatches {
		if leaf := matchedLeaf(dn, cm); leaf != nil {
			out.Children = append(out.Children, leaf.clone())
		}
	}
	// Apply each selector against the matching data children.
	for _, sf := range selectors {
		for _, dc := range dn.Children {
			if !nameMatch(dc, sf) {
				continue
			}
			if m := matchNode(dc, sf); m != nil {
				out.Children = append(out.Children, m)
			}
		}
	}

	// A containment node with selectors that matched nothing is excluded.
	if len(out.Children) == 0 {
		return nil
	}
	return out
}

// matchedLeaf returns dn's first same-named leaf child whose text equals cm's,
// or nil.
func matchedLeaf(dn, cm *dataNode) *dataNode {
	for _, dc := range dn.Children {
		if nameMatch(dc, cm) && dc.isLeaf() && dc.leafText() == cm.leafText() {
			return dc
		}
	}
	return nil
}

// nameMatch reports whether data node d matches filter node f by local name,
// requiring namespace equality only when the filter node declares one.
func nameMatch(d, f *dataNode) bool {
	if d.Name.Local != f.Name.Local {
		return false
	}
	return f.Name.Space == "" || d.Name.Space == f.Name.Space
}
