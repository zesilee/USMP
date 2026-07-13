package xmlcodec

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"reflect"
	"strconv"

	"github.com/openconfig/ygot/ygot"
)

// Decode parses a NETCONF get-config reply (raw XML — wrapped in
// <rpc-reply>/<data>, bare, or namespace-prefixed) into dst, a ygot
// list-container GoStruct (XC-02). It scans the token stream for the list
// entry element (robust to outer wrappers, like the legacy parsers), fills
// every field addressable by `path:` tag — decode coverage is therefore a
// superset of what Encode can emit（消除「可下发但回读丢失」漂移债, D3b）—
// and inserts entries keyed via ΛListKeyMap, synthesizing a key when the key
// leaf is absent（legacy 宽容语义）. Empty input yields an initialized empty
// map; malformed XML or non-numeric enum text returns an explicit error (R08).
func Decode(spec *Spec, raw []byte, dst ygot.GoStruct) error {
	cv, err := derefContainer(dst)
	if err != nil {
		return err
	}
	mapVal, elemTag, found, err := findContainerMap(cv)
	if err != nil {
		return err
	}
	if !found {
		// Plain-container root (e.g. /bgp:bgp): parse the container element's
		// children straight into dst via the shared struct machinery.
		return decodeContainer(spec, raw, cv)
	}
	if mapVal.IsNil() {
		mapVal.Set(reflect.MakeMap(mapVal.Type()))
	}
	if len(raw) == 0 {
		return nil
	}
	r, err := spec.resolve(elemTag)
	if err != nil {
		return err
	}

	// List entries are only decoded while inside their list container (r.root,
	// e.g. <vlans>/<interfaces>). Anchoring here is essential for Huawei models
	// whose module container shares the entry's name（<vlan><vlans><vlan>）——a
	// naive "scan for <vlan> anywhere" would mis-match the outer module container
	// as a bogus entry. depth counts open r.root containers (robust to the
	// rpc-reply/data/module-container wrappers a get-config reply carries).
	dec := xml.NewDecoder(bytes.NewReader(raw))
	idx := 0
	depth := 0
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("xmlcodec decode: %w", err)
		}
		switch se := tok.(type) {
		case xml.StartElement:
			switch se.Name.Local {
			case r.root:
				depth++
			case elemTag:
				if depth == 0 {
					continue // outside the list container: not an entry (wrapper collision)
				}
				entry := reflect.New(mapVal.Type().Elem().Elem()) // *EntryStruct
				if err := decodeStruct(dec, se, entry.Elem()); err != nil {
					return fmt.Errorf("xmlcodec decode <%s>: %w", elemTag, err)
				}
				key, err := entryKey(entry, mapVal.Type().Key(), elemTag, idx)
				if err != nil {
					return err
				}
				mapVal.SetMapIndex(key, entry)
				idx++
			}
		case xml.EndElement:
			if se.Name.Local == r.root && depth > 0 {
				depth--
			}
		}
	}
}

// decodeContainer parses a get-config reply into a plain-container root struct
// (no root YANG list, e.g. /bgp:bgp). It scans the token stream for the module
// root element (robust to rpc-reply/data/namespace-prefixed wrappers, matching
// by local name like the list path) and fills cv via decodeStruct — the same
// machinery list entries use. Root element absent → cv left empty (lenient,
// mirrors the list path's empty-map behavior).
func decodeContainer(spec *Spec, raw []byte, cv reflect.Value) error {
	if len(raw) == 0 {
		return nil
	}
	r, err := spec.resolve("")
	if err != nil {
		return err
	}
	dec := xml.NewDecoder(bytes.NewReader(raw))
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("xmlcodec decode: %w", err)
		}
		if se, ok := tok.(xml.StartElement); ok && se.Name.Local == r.root {
			return decodeStruct(dec, se, cv)
		}
	}
}

// decodeStruct consumes tokens through the matching EndElement of start,
// populating sv (a struct value) by `path:` tag lookup. Unknown elements are
// skipped (forward compatible, legacy 行为).
func decodeStruct(dec *xml.Decoder, start xml.StartElement, sv reflect.Value) error {
	for {
		tok, err := dec.Token()
		if err != nil {
			return fmt.Errorf("in <%s>: %w", start.Name.Local, err)
		}
		switch t := tok.(type) {
		case xml.EndElement:
			return nil
		case xml.StartElement:
			f, ok := fieldByTag(sv, t.Name.Local)
			if !ok {
				if err := dec.Skip(); err != nil {
					return fmt.Errorf("skip <%s>: %w", t.Name.Local, err)
				}
				continue
			}
			if err := decodeField(dec, t, f); err != nil {
				return err
			}
		}
	}
}

func decodeField(dec *xml.Decoder, start xml.StartElement, fv reflect.Value) error {
	tag := start.Name.Local
	if fv.Type().Implements(goEnumType) && fv.Kind() == reflect.Int64 {
		text, err := collectText(dec, start)
		if err != nil {
			return err
		}
		n, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return fmt.Errorf("leaf %s: non-numeric enum value %q", tag, text)
		}
		fv.SetInt(n)
		return nil
	}
	switch fv.Kind() {
	case reflect.Ptr:
		if fv.Type().Elem().Kind() == reflect.Struct { // nested container
			if fv.IsNil() {
				fv.Set(reflect.New(fv.Type().Elem()))
			}
			return decodeStruct(dec, start, fv.Elem())
		}
		text, err := collectText(dec, start)
		if err != nil {
			return err
		}
		out := reflect.New(fv.Type().Elem())
		if err := parseScalar(out.Elem(), text); err != nil {
			return fmt.Errorf("leaf %s: %w", tag, err)
		}
		fv.Set(out)
		return nil
	case reflect.Map: // nested YANG list
		if fv.IsNil() {
			fv.Set(reflect.MakeMap(fv.Type()))
		}
		entry := reflect.New(fv.Type().Elem().Elem())
		if err := decodeStruct(dec, start, entry.Elem()); err != nil {
			return err
		}
		key, err := entryKey(entry, fv.Type().Key(), tag, fv.Len())
		if err != nil {
			return err
		}
		fv.SetMapIndex(key, entry)
		return nil
	case reflect.Slice: // leaf-list
		if fv.Type().Elem().Kind() == reflect.Uint8 {
			return fmt.Errorf("leaf %s: binary unsupported", tag)
		}
		text, err := collectText(dec, start)
		if err != nil {
			return err
		}
		item := reflect.New(fv.Type().Elem())
		if err := parseScalar(item.Elem(), text); err != nil {
			return fmt.Errorf("leaf-list %s: %w", tag, err)
		}
		fv.Set(reflect.Append(fv, item.Elem()))
		return nil
	default:
		return fmt.Errorf("leaf %s: unsupported field form %s", tag, fv.Kind())
	}
}

// collectText reads the character content of a leaf element through its
// EndElement; a nested StartElement means the document does not match the
// schema shape and is an explicit error.
func collectText(dec *xml.Decoder, start xml.StartElement) (string, error) {
	var b bytes.Buffer
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", fmt.Errorf("leaf <%s>: %w", start.Name.Local, err)
		}
		switch t := tok.(type) {
		case xml.CharData:
			b.Write(t)
		case xml.EndElement:
			return b.String(), nil
		case xml.StartElement:
			return "", fmt.Errorf("leaf <%s>: unexpected child <%s>", start.Name.Local, t.Name.Local)
		}
	}
}

func parseScalar(v reflect.Value, text string) error {
	switch v.Kind() {
	case reflect.String:
		v.SetString(text)
	case reflect.Bool:
		b, err := strconv.ParseBool(text)
		if err != nil {
			return fmt.Errorf("bad bool %q", text)
		}
		v.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(text, 10, v.Type().Bits())
		if err != nil {
			return fmt.Errorf("bad integer %q", text)
		}
		v.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(text, 10, v.Type().Bits())
		if err != nil {
			return fmt.Errorf("bad unsigned integer %q", text)
		}
		v.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(text, v.Type().Bits())
		if err != nil {
			return fmt.Errorf("bad decimal %q", text)
		}
		v.SetFloat(f)
	default:
		return fmt.Errorf("unsupported scalar kind %s", v.Kind())
	}
	return nil
}

// entryKey derives the map key for a decoded list entry: the single key from
// ΛListKeyMap when present, else a synthesized fallback（legacy 宽容语义：
// 无 key 的回读条目不丢弃）. Multi-key lists are explicitly unsupported.
func entryKey(entry reflect.Value, keyType reflect.Type, elemTag string, idx int) (reflect.Value, error) {
	if kh, ok := entry.Interface().(ygot.KeyHelperGoStruct); ok {
		if km, err := kh.ΛListKeyMap(); err == nil {
			if len(km) > 1 {
				return reflect.Value{}, fmt.Errorf("list %s: multi-key lists unsupported", elemTag)
			}
			for _, kv := range km {
				rv := reflect.ValueOf(kv)
				if !rv.Type().ConvertibleTo(keyType) {
					return reflect.Value{}, fmt.Errorf("list %s: key type %s not convertible to %s", elemTag, rv.Type(), keyType)
				}
				return rv.Convert(keyType), nil
			}
		}
	}
	// Key leaf missing: synthesize so the entry survives（对齐 legacy）.
	switch keyType.Kind() {
	case reflect.String:
		return reflect.ValueOf(fmt.Sprintf("%s-%d", elemTag, idx)).Convert(keyType), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return reflect.ValueOf(uint64(idx)).Convert(keyType), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return reflect.ValueOf(int64(idx)).Convert(keyType), nil
	default:
		return reflect.Value{}, fmt.Errorf("list %s: cannot synthesize key of type %s", elemTag, keyType)
	}
}
