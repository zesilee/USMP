package api

import (
	"encoding/json"
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/native/huawei"
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

// 回归（真机 devm ports，PR 前实测报文缩样）：批量接入模块（plainModules）的解码根
// 是模块根容器，读子路径时必须把多余外层剥掉、返回以请求路径为根的子树——否则前端
// 把容器键（"ports"→"port"）当成唯一一行渲染（位置=port）。契约对齐 ifm/vlan：
// decodeRunningConfig 恒返回「以请求 path 为根」的 RFC7951 子树。
func TestDecodeRunningConfigPeelsToRequestPath(t *testing.T) {
	devmRaw := []byte(`<rpc-reply message-id="16" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"><data>` +
		`<devm xmlns="urn:huawei:yang:huawei-devm"><ports>` +
		`<port><position>MEth0/0/0</position><admin-state>active</admin-state></port>` +
		`<port><position>400GE1/0/1</position><admin-state>active</admin-state></port>` +
		`</ports></devm></data></rpc-reply>`)

	tests := []struct {
		name    string
		path    string
		raw     []byte
		wantKey string // 期望出现在返回 map 顶层的键
		wantLen int    // wantKey 为数组时的期望行数（-1 = 不校验）
	}{
		{
			// 主症状：list Tab 容器路径读，须剥到 {"port":[...]}。
			name: "devm ports 子路径剥层", path: "/devm:devm/devm:ports",
			raw: devmRaw, wantKey: "port", wantLen: 2,
		},
		{
			// 单行状态读（include_state）：谓词段停剥，返回谓词段的父容器子树
			//（ItemDetailPane 契约：sub["port"] 取行）。注意前端拼谓词段不带模块前缀。
			name: "谓词段停剥返回父容器", path: "/devm:devm/devm:ports/port[position='400GE1/0/1']",
			raw: devmRaw, wantKey: "port", wantLen: -1,
		},
		{
			// 模块根路径读（path==解码根）：形状不变，仍含 "ports" 外层。
			name: "模块根路径不剥", path: "/devm:devm",
			raw: devmRaw, wantKey: "ports", wantLen: -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := decodeRunningConfig(tt.path, tt.raw)
			m, ok := out.(map[string]interface{})
			if !ok {
				t.Fatalf("decode degraded to %T, want map", out)
			}
			v, ok := m[tt.wantKey]
			if !ok {
				keys := make([]string, 0, len(m))
				for k := range m {
					keys = append(keys, k)
				}
				t.Fatalf("top-level key %q missing, got keys %v", tt.wantKey, keys)
			}
			if tt.wantLen >= 0 {
				rows, ok := v.([]interface{})
				if !ok {
					t.Fatalf("%q = %T, want array", tt.wantKey, v)
				}
				if len(rows) != tt.wantLen {
					t.Fatalf("%q rows = %d, want %d", tt.wantKey, len(rows), tt.wantLen)
				}
			}
		})
	}
}

// 回归：请求的子路径在设备回包里不存在（如该容器未配置）→ 返回空 map（前端零行），
// 不得把整棵模块树原样透出（否则又回到「容器键当行」的老症状）。
func TestDecodeRunningConfigPeelMissingSubtree(t *testing.T) {
	raw := []byte(`<data><devm xmlns="urn:huawei:yang:huawei-devm"><ports>` +
		`<port><position>MEth0/0/0</position></port></ports></devm></data>`)
	out := decodeRunningConfig("/devm:devm/devm:physical-entitys", raw)
	m, ok := out.(map[string]interface{})
	if !ok {
		t.Fatalf("decode degraded to %T, want map", out)
	}
	if len(m) != 0 {
		t.Fatalf("want empty map for absent subtree, got %v", m)
	}
}

// 回归：network-instance 双前缀口径（控制台按根名派生 /network-instance:...，
// 描述符锚点是 /ni:...）——剥层按段局部名对齐，前缀不一致不得导致误剥或不剥。
func TestDecodeRunningConfigPeelNiDualPrefix(t *testing.T) {
	raw := []byte(`<data><network-instance xmlns="urn:huawei:yang:huawei-network-instance"><instances>` +
		`<instance><name>vrf-a</name></instance><instance><name>vrf-b</name></instance>` +
		`</instances></network-instance></data>`)
	out := decodeRunningConfig("/network-instance:network-instance/network-instance:instances", raw)
	m, ok := out.(map[string]interface{})
	if !ok {
		t.Fatalf("decode degraded to %T, want map", out)
	}
	rows, ok := m["instance"].([]interface{})
	if !ok || len(rows) != 2 {
		t.Fatalf("instance rows = %#v, want 2 rows", m)
	}
}

// peelToPath 边界（表格驱动）：RFC7951 emit 在跨模块边界会产出 "module:name"
// 前缀键（augment 子树），剥层须按局部名命中；中途遇非 map 形状按 R08 降级。
func TestPeelToPathEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		m      map[string]interface{}
		anchor string
		path   string
		want   interface{}
	}{
		{
			name:   "跨模块前缀键按局部名命中",
			m:      map[string]interface{}{"huawei-pic:ethernet": map[string]interface{}{"speed": "auto"}},
			anchor: "/devm:devm",
			path:   "/devm:devm/devm:ethernet",
			want:   map[string]interface{}{"speed": "auto"},
		},
		{
			name:   "中途值非map降级返回已剥层",
			m:      map[string]interface{}{"ports": []interface{}{"x"}},
			anchor: "/devm:devm",
			path:   "/devm:devm/devm:ports/devm:deeper",
			want:   []interface{}{"x"},
		},
		{
			name:   "空anchor不剥",
			m:      map[string]interface{}{"a": 1.0},
			anchor: "",
			path:   "/devm:devm/devm:ports",
			want:   map[string]interface{}{"a": 1.0},
		},
		{
			name:   "path短于anchor不剥",
			m:      map[string]interface{}{"vlan": []interface{}{}},
			anchor: "/vlan:vlan/vlan:vlans",
			path:   "/vlan:vlan",
			want:   map[string]interface{}{"vlan": []interface{}{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := peelToPath(tt.m, tt.anchor, tt.path)
			gb, _ := json.Marshal(got)
			wb, _ := json.Marshal(tt.want)
			if string(gb) != string(wb) {
				t.Fatalf("peelToPath = %s, want %s", gb, wb)
			}
		})
	}
}

// 回读值不满足本地 YANG pattern（如设备返回带点号的子接口 number）时，decode 必须
// 仍产出 RFC7951 map 而非降级为不透明 XML bytes（R08：一叶异常不拖垮整个回读）。
func TestDecodeRunningConfigSkipsValidationOnReadback(t *testing.T) {
	raw := []byte(`<data><ifm xmlns="urn:huawei:yang:huawei-ifm"><interfaces>` +
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
