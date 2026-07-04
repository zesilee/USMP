package api

import (
	"encoding/json"
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
)

func toMap(t *testing.T, s string) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	return m
}

// TestEncodeToYgotVlan (task 3.1): RFC7951 vlan data decodes into the ygot struct
// via the generic codec (single ygot.Unmarshal).
func TestEncodeToYgotVlan(t *testing.T) {
	data := toMap(t, `{"vlan":[{"id":100,"name":"office","admin-status":"up"}]}`)
	v, matched, err := encodeToYgot("/vlan:vlan/vlan:vlans", data)
	if !matched || err != nil {
		t.Fatalf("encodeToYgot: matched=%v err=%v", matched, err)
	}
	vlans, ok := v.(*huawei.HuaweiVlan_Vlan_Vlans)
	if !ok {
		t.Fatalf("wrong type %T", v)
	}
	vlan := vlans.Vlan[100]
	if vlan == nil || vlan.Name == nil || *vlan.Name != "office" {
		t.Fatalf("vlan 100 not decoded correctly: %+v", vlans.Vlan)
	}
}

// TestEncodeToYgotUnregistered (task 3.1): an unregistered path is not matched, so
// the caller falls back.
func TestEncodeToYgotUnregistered(t *testing.T) {
	_, matched, err := encodeToYgot("/some:unknown/path", toMap(t, `{"x":1}`))
	if matched || err != nil {
		t.Fatalf("unregistered path should not match: matched=%v err=%v", matched, err)
	}
}

// TestConvertConfigDispatch (task 3.3, double-path): the unified dispatcher routes
// RFC7951 input through the generic codec and legacy (integer-enum) input through
// the legacy fallback — both yielding valid ygot structs, no regression.
func TestConvertConfigDispatch(t *testing.T) {
	// RFC7951 vlan → generic codec.
	v, err := convertConfig("/vlan:vlan/vlan:vlans", toMap(t, `{"vlan":[{"id":100,"name":"office","admin-status":"up"}]}`))
	if err != nil {
		t.Fatalf("generic dispatch: %v", err)
	}
	if v.(*huawei.HuaweiVlan_Vlan_Vlans).Vlan[100] == nil {
		t.Fatal("generic dispatch did not decode vlan 100")
	}

	// Legacy integer-enum ifm shape → generic fails (enum type) → legacy fallback.
	iface, err := convertConfig("/ifm:ifm/ifm:interfaces", toMap(t, `{"interface":[{"name":"GE0/0/1","admin-status":2,"mtu":1500}]}`))
	if err != nil {
		t.Fatalf("legacy dispatch: %v", err)
	}
	ifm, ok := iface.(*huawei.HuaweiIfm_Ifm_Interfaces)
	if !ok || ifm.Interface["GE0/0/1"] == nil {
		t.Fatalf("legacy fallback did not decode interface: %T", iface)
	}
}
