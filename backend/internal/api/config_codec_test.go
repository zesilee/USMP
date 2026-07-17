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

// BR-05/BR-06 单一路径：RFC7951 子树经注册表锚点解码为强类型结构。
func TestConvertConfigVlan(t *testing.T) {
	v, err := convertConfig("/vlan:vlan/vlan:vlans", toMap(t, `{"vlan":[{"id":100,"name":"office","admin-status":"up"}]}`))
	if err != nil {
		t.Fatalf("convertConfig: %v", err)
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

// BR-06 负路径：整数枚举等 legacy 形状显式拒绝（一步到位，无回退）。
func TestConvertConfigRejectsLegacyIntegerEnum(t *testing.T) {
	_, err := convertConfig("/ifm:ifm/ifm:interfaces", toMap(t, `{"interface":[{"name":"GE0/0/1","admin-status":2,"mtu":1500}]}`))
	if err == nil {
		t.Fatal("integer-enum legacy shape should be rejected")
	}
}

// 回读值不满足本地 YANG pattern（如设备返回带点号的子接口 number）时，decode 必须
// 仍产出 RFC7951 map 而非降级为不透明 XML bytes（R08：一叶异常不拖垮整个回读）。
func TestDecodeRunningConfigSkipsValidationOnReadback(t *testing.T) {
	raw := []byte(`<data><ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm"><interfaces>` +
		`<interface><name>200GE0/1/0.1</name><class>2</class><number>0/1/0.1</number></interface>` +
		`</interfaces></ifm></data>`)
	out := decodeRunningConfig("/ifm:ifm/ifm:interfaces", raw)
	m, ok := out.(map[string]interface{})
	if !ok {
		t.Fatalf("decode degraded to %T, want map", out)
	}
	rows, ok := m["interface"].([]interface{})
	if !ok || len(rows) != 1 {
		t.Fatalf("interface rows = %#v", m)
	}
	row := rows[0].(map[string]interface{})
	if row["number"] != "0/1/0.1" {
		t.Errorf("number = %v", row["number"])
	}
	if row["class"] != "sub-interface" {
		t.Errorf("class = %v (want enum name string)", row["class"])
	}
}
