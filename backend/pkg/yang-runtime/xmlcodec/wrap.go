package xmlcodec

import (
	"fmt"
	"reflect"

	"github.com/openconfig/ygot/ygot"
)

// ListMapType returns the reflect.Type of the container's YANG-list map field
// (e.g. map[uint16]*Vlan for the vlans container). Drivers/registry use it to
// match "inner map" change values — the diff engine emits the list itself as
// a typed map（IFM 漏发 bug 的根因形态）.
func ListMapType(container ygot.GoStruct) (reflect.Type, error) {
	cv, err := derefContainer(container)
	if err != nil {
		return nil, err
	}
	mapVal, _, err := containerMap(cv)
	if err != nil {
		return nil, err
	}
	return mapVal.Type(), nil
}

// WrapListMap sets m (which must be the container's list map type) as the
// container's list map field, so inner-map change values can be encoded
// through the same container path.
func WrapListMap(container ygot.GoStruct, m interface{}) error {
	cv, err := derefContainer(container)
	if err != nil {
		return err
	}
	mapVal, tag, err := containerMap(cv)
	if err != nil {
		return err
	}
	rv := reflect.ValueOf(m)
	if !rv.IsValid() || rv.Type() != mapVal.Type() {
		return fmt.Errorf("xmlcodec: value type %T does not match list map field %s (%s)", m, tag, mapVal.Type())
	}
	mapVal.Set(rv)
	return nil
}
