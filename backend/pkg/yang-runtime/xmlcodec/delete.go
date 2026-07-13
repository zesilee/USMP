package xmlcodec

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/openconfig/ygot/ygot"
)

// EncodeDelete builds a keyed edit-config delete for every entry of a ygot
// list container (XC-03, DP-07): outer model container with module namespace,
// entry elements carrying nc:operation="delete" in the NETCONF base namespace,
// and ONLY the key leaf (first and sole child). Keys come from ΛListKeyMap,
// falling back to the schema Key statement + map key when the key leaf is
// nil（legacy 宽容语义）. An empty target or undeterminable key is an explicit
// error — a bare untargeted delete is never emitted (R08).
func EncodeDelete(spec *Spec, v ygot.GoStruct) (string, error) {
	cv, err := derefContainer(v)
	if err != nil {
		return "", err
	}
	mapVal, elemTag, err := containerMap(cv)
	if err != nil {
		return "", err
	}
	r, err := spec.resolve(elemTag)
	if err != nil {
		return "", err
	}
	if mapVal.IsNil() || mapVal.Len() == 0 {
		return "", fmt.Errorf("xmlcodec delete: empty %s target", elemTag)
	}

	var b strings.Builder
	wrapped := openWrappers(&b, r)
	if wrapped {
		fmt.Fprintf(&b, "<%s>", r.root)
	} else {
		fmt.Fprintf(&b, "<%s xmlns=%q>", r.root, r.ns)
	}
	for _, mk := range sortedKeys(mapVal) {
		ev := mapVal.MapIndex(mk)
		keyName, keyVal, err := deleteKey(ev, mk, r)
		if err != nil {
			return "", fmt.Errorf("xmlcodec delete %s: %w", elemTag, err)
		}
		fmt.Fprintf(&b, `<%s nc:operation="delete" xmlns:nc=%q>`, elemTag, NetconfBaseNS)
		if err := encodeLeaf(&b, keyName, keyVal, ""); err != nil {
			return "", fmt.Errorf("xmlcodec delete %s: %w", elemTag, err)
		}
		fmt.Fprintf(&b, "</%s>", elemTag)
	}
	fmt.Fprintf(&b, "</%s>", r.root)
	closeWrappers(&b, r)
	return b.String(), nil
}

// deleteKey resolves the single key leaf (name, value) for one list entry.
func deleteKey(ev reflect.Value, mapKey reflect.Value, r *resolved) (string, reflect.Value, error) {
	if ev.Kind() == reflect.Ptr && !ev.IsNil() {
		if kh, ok := ev.Interface().(ygot.KeyHelperGoStruct); ok {
			if km, err := kh.ΛListKeyMap(); err == nil && len(km) > 0 {
				if len(km) > 1 {
					return "", reflect.Value{}, fmt.Errorf("multi-key lists unsupported")
				}
				names := make([]string, 0, 1)
				for n := range km {
					names = append(names, n)
				}
				sort.Strings(names)
				return names[0], reflect.ValueOf(km[names[0]]), nil
			}
		}
	}
	// Key leaf nil on the entry: schema key name + map key value（legacy 回退）.
	if names := r.keyNames(); len(names) == 1 {
		return names[0], mapKey, nil
	}
	return "", reflect.Value{}, fmt.Errorf("cannot determine list key (no ΛListKeyMap value, no single schema key)")
}
