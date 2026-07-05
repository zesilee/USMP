package api

import "testing"

// VLAN ID 边界校验：1-4094 合法，0/4095+ 必须被拒（YANG 模型未编码此范围）。
func TestValidateConfig_VlanIDRange(t *testing.T) {
	cases := []struct {
		id      float64
		wantErr bool
	}{
		{1, false},
		{100, false},
		{4094, false},
		{0, true},
		{4095, true},
		{5000, true},
	}
	for _, tc := range cases {
		typed, err := convertMapToHuaweiVlan(map[string]interface{}{
			"vlans": []interface{}{map[string]interface{}{"id": tc.id, "name": "v"}},
		})
		if err != nil {
			t.Fatalf("convert id=%v: %v", tc.id, err)
		}
		verr := validateConfig(typed)
		if tc.wantErr && verr == nil {
			t.Errorf("VLAN ID %v 应被拒绝但通过了", tc.id)
		}
		if !tc.wantErr && verr != nil {
			t.Errorf("VLAN ID %v 应通过但被拒: %v", tc.id, verr)
		}
	}
}

// 非 VLAN 类型不受此校验影响（返回 nil）。
func TestValidateConfig_NonVlanPasses(t *testing.T) {
	if err := validateConfig(map[string]interface{}{"foo": "bar"}); err != nil {
		t.Errorf("非 VLAN 配置不应被校验拒绝: %v", err)
	}
}
