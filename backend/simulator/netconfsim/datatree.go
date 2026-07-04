package netconfsim

import (
	"bytes"
	"encoding/xml"
	"io"
	"strings"
)

// dataNode is a generic XML data-tree node used as the simulator's configuration
// store. It is model-agnostic (works for both Huawei and OpenConfig config), which
// lets edit-config merge/delete (T3) and get-config subtree filtering (T4) operate
// uniformly — without the per-model, per-XML-shape string parsing that the legacy
// blob datastore required.
//
// Namespaces are captured as the resolved URI on Name.Space and re-emitted as a
// default-namespace declaration (xmlns="…") on serialization; prefixes are dropped
// but semantics preserved. This is sufficient for a test double.
type dataNode struct {
	Name     xml.Name
	Attrs    []xml.Attr // non-namespace attributes only (e.g. operation="delete")
	Children []*dataNode
	Text     string // trimmed character data; meaningful only for leaf nodes
}

// parseXML parses config XML into a synthetic document root whose Children are the
// top-level elements. NETCONF <config> bodies may contain several sibling roots
// (e.g. <vlan/> and <ifm/>), so a synthetic root keeps them together.
func parseXML(data []byte) (*dataNode, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	root := &dataNode{}
	stack := []*dataNode{root}

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			n := &dataNode{Name: t.Name, Attrs: keepNonNamespaceAttrs(t.Attr)}
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, n)
			stack = append(stack, n)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			cur := stack[len(stack)-1]
			cur.Text += string(t)
		}
	}
	return root, nil
}

// keepNonNamespaceAttrs drops xmlns / xmlns:* declarations (namespaces are handled
// via Name.Space) and keeps real attributes such as operation="…".
func keepNonNamespaceAttrs(attrs []xml.Attr) []xml.Attr {
	var out []xml.Attr
	for _, a := range attrs {
		if a.Name.Local == "xmlns" || a.Name.Space == "xmlns" {
			continue
		}
		out = append(out, a)
	}
	return out
}

// isLeaf reports whether the node has no element children.
func (n *dataNode) isLeaf() bool { return len(n.Children) == 0 }

// leafText returns the trimmed text of a leaf node.
func (n *dataNode) leafText() string { return strings.TrimSpace(n.Text) }

// child returns the first child with the given local name, or nil.
func (n *dataNode) child(local string) *dataNode {
	for _, c := range n.Children {
		if c.Name.Local == local {
			return c
		}
	}
	return nil
}

// children returns all children with the given local name.
func (n *dataNode) children(local string) []*dataNode {
	var out []*dataNode
	for _, c := range n.Children {
		if c.Name.Local == local {
			out = append(out, c)
		}
	}
	return out
}

// find walks a path of local element names from this node and returns the first
// matching descendant, or nil. Example: find("vlan", "vlans", "vlan").
func (n *dataNode) find(path ...string) *dataNode {
	cur := n
	for _, seg := range path {
		if cur == nil {
			return nil
		}
		cur = cur.child(seg)
	}
	return cur
}

// clone returns a deep copy of the node (used for commit/discard snapshots).
func (n *dataNode) clone() *dataNode {
	if n == nil {
		return nil
	}
	cp := &dataNode{
		Name: n.Name,
		Text: n.Text,
	}
	if n.Attrs != nil {
		cp.Attrs = append([]xml.Attr(nil), n.Attrs...)
	}
	for _, c := range n.Children {
		cp.Children = append(cp.Children, c.clone())
	}
	return cp
}

// xmlBytes serializes the tree back to XML. The synthetic root emits only its
// children. Namespaces are emitted as default-namespace declarations where the
// node's namespace differs from its parent's.
func (n *dataNode) xmlBytes() []byte {
	var b strings.Builder
	n.write(&b, "")
	return []byte(b.String())
}

func (n *dataNode) write(b *strings.Builder, parentNS string) {
	// Synthetic document root: emit children only.
	if n.Name.Local == "" {
		for _, c := range n.Children {
			c.write(b, parentNS)
		}
		return
	}

	b.WriteByte('<')
	b.WriteString(n.Name.Local)
	if n.Name.Space != "" && n.Name.Space != parentNS {
		b.WriteString(` xmlns="`)
		xml.EscapeText(b, []byte(n.Name.Space))
		b.WriteString(`"`)
	}
	for _, a := range n.Attrs {
		b.WriteByte(' ')
		b.WriteString(a.Name.Local)
		b.WriteString(`="`)
		xml.EscapeText(b, []byte(a.Value))
		b.WriteString(`"`)
	}

	if n.isLeaf() && n.leafText() == "" {
		b.WriteString("/>")
		return
	}
	b.WriteByte('>')
	if n.isLeaf() {
		xml.EscapeText(b, []byte(n.leafText()))
	} else {
		ns := n.Name.Space
		if ns == "" {
			ns = parentNS
		}
		for _, c := range n.Children {
			c.write(b, ns)
		}
	}
	b.WriteString("</")
	b.WriteString(n.Name.Local)
	b.WriteByte('>')
}
