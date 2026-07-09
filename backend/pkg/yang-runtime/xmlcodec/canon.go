// Package xmlcodec provides the generic, schema/tag data-driven NETCONF XML
// codec (yang-xml-codec XC-01/02/03): encoding, decoding and keyed-delete
// encoding for arbitrary ygot GoStructs, plus a canonicalizer used by golden
// tests. Element names come from ygot `path:` struct tags, namespaces and
// config-false filtering from the generated SchemaTree, list keys from
// ΛListKeyMap — adding a YANG module requires registering descriptor data,
// never per-model XML code.
package xmlcodec

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Canonicalize parses raw XML and returns a deterministic textual form for
// equality comparison (golden tests, D3):
//   - element/attribute names are resolved to "{namespaceURI}local" so prefix
//     choice is insignificant; xmlns declaration attributes are dropped;
//   - attributes are sorted by resolved name;
//   - sibling elements are fully sorted by their canonical serialization —
//     YANG leaf/list sibling order is semantically insignificant (key-first
//     ordering is asserted separately on raw output, not here);
//   - identical siblings are deduplicated after sorting: under NETCONF merge
//     semantics a repeated identical element is an idempotent no-op（顺带把
//     历史上 <suppression> 重复发送规范化掉）;
//   - inter-element whitespace is ignored; leaf text is entity-decoded and
//     trimmed, so escaping style is insignificant.
//
// It is stateless and safe for concurrent use (R09).
func Canonicalize(raw []byte) (string, error) {
	dec := xml.NewDecoder(bytes.NewReader(raw))
	var roots []string
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("canonicalize: %w", err)
		}
		if se, ok := tok.(xml.StartElement); ok {
			s, err := canonElement(dec, se)
			if err != nil {
				return "", err
			}
			roots = append(roots, s)
		}
	}
	if len(roots) == 0 {
		return "", fmt.Errorf("canonicalize: no XML element in input")
	}
	sort.Strings(roots)
	return strings.Join(roots, "\n"), nil
}

// canonElement consumes tokens through the matching EndElement of se and
// returns the canonical serialization of the element.
func canonElement(dec *xml.Decoder, se xml.StartElement) (string, error) {
	var attrs []string
	for _, a := range se.Attr {
		// Drop namespace declarations: resolved URIs already carry the info.
		if a.Name.Space == "xmlns" || (a.Name.Space == "" && a.Name.Local == "xmlns") {
			continue
		}
		attrs = append(attrs, fmt.Sprintf("%s=%q", qname(a.Name), a.Value))
	}
	sort.Strings(attrs)

	var children []string
	var text strings.Builder
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", fmt.Errorf("canonicalize <%s>: %w", se.Name.Local, err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			c, err := canonElement(dec, t)
			if err != nil {
				return "", err
			}
			children = append(children, c)
		case xml.CharData:
			text.Write(t)
		case xml.EndElement:
			sort.Strings(children)
			deduped := children[:0]
			for i, c := range children {
				if i == 0 || c != children[i-1] {
					deduped = append(deduped, c)
				}
			}
			var b strings.Builder
			b.WriteString(qname(se.Name))
			if len(attrs) > 0 {
				b.WriteString("[" + strings.Join(attrs, " ") + "]")
			}
			b.WriteString("(")
			if len(deduped) > 0 {
				b.WriteString(strings.Join(deduped, ","))
			} else {
				b.WriteString(strings.TrimSpace(text.String()))
			}
			b.WriteString(")")
			return b.String(), nil
		}
	}
}

func qname(n xml.Name) string {
	if n.Space == "" {
		return n.Local
	}
	return "{" + n.Space + "}" + n.Local
}
