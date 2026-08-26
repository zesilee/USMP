package api

import (
	"testing"

	"github.com/leezesi/usmp/backend/internal/yangschema"
)

// 「不得误拒」守护（change config-write-validation 任务 4，design D6）。
//
// 为什么单独立这个用例：全量 API 测试几乎都用 manager.New()——**空 schema**，
// 新接入的约束校验在它们里面直接跳过。也就是说「全量测试全绿」并**不**证明
// 校验器不误伤存量配置，那个结论是假的安全感。
//
// 本用例用**真实装载的 schema IR** 跑各模块的已知合法载荷（取自既有全属性用例，
// 都是曾被端到端下发到模拟网元验证过的形状），断言一条都不被拒。
// 这才是拒绝面的真实测量。新增模块接入设备配置时，把它的合法载荷补进本表。
func TestValidateAgainstSchema_KnownGoodPayloadsNotRejected(t *testing.T) {
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("load schema: %v", err)
	}

	cases := []struct {
		name string
		path string
		json string
	}{
		{
			name: "ifm 接口全属性",
			path: "/ifm:ifm/ifm:interfaces",
			json: `{"interface":[{
				"name":"GigabitEthernet0/0/1","description":"Test Interface with full attributes",
				"index":12345,"number":"0/0/1","position":"0/0/1","parent-name":"GigabitEthernet0",
				"admin-status":"up","type":"GigabitEthernet","class":"main-interface",
				"router-type":"broadcast","service-type":"none","mtu":1500,
				"mac-address":"0011-2233-4455","bandwidth":1000,"bandwidth-kbps":1000000,
				"vrf-name":"public","vs-name":"vs1"}]}`,
		},
		{
			name: "vlan 全属性",
			path: "/vlan:vlan/vlan:vlans",
			json: `{"vlan":[{
				"id":10,"name":"VLAN10","description":"Test VLAN","type":"common",
				"admin-status":"up","broadcast-discard":"enable",
				"unknown-multicast-discard":"disable","mac-learning":"enable",
				"mac-aging-time":300,"statistic-enable":"disable","super-vlan":100}]}`,
		},
		{
			name: "system 基本信息",
			path: "/system:system/system:system-info",
			json: `{"sys-name":"TestRouter","sys-contact":"admin@example.com"}`,
		},
		{
			name: "ifm 子接口（choice 分支）",
			path: "/ifm:ifm/ifm:interfaces",
			json: `{"interface":[{"name":"GigabitEthernet0/0/1.100","class":"sub-interface",
				"parent-name":"GigabitEthernet0/0/1","admin-status":"up"}]}`,
		},
		{
			name: "vlan 最小载荷（仅主键）",
			path: "/vlan:vlan/vlan:vlans",
			json: `{"vlan":[{"id":1}]}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			typed, anchor, cerr := convertConfigAnchored(tc.path, decodeJSON(t, tc.json))
			if cerr != nil {
				t.Fatalf("转换失败（与校验无关，先修转换）: %v", cerr)
			}
			if err := validateAgainstSchema(s, anchor, typed); err != nil {
				t.Errorf("已知合法载荷被误拒——按 D6 应修校验器而非放宽接入: %v", err)
			}
		})
	}
}

// 反向对照：确认上面的"全过"不是因为校验被静默跳过了。
// 同一条链路喂一个明确违约的值，必须被拒——否则说明锚点没解析上、校验根本没跑，
// 上面那组"全过"就是假绿。
func TestValidateAgainstSchema_ProbeRejectsKnownBad(t *testing.T) {
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("load schema: %v", err)
	}

	// mac-address 有 pattern 约束，喂一个明显不合形状的值。
	typed, anchor, cerr := convertConfigAnchored("/ifm:ifm/ifm:interfaces", decodeJSON(t,
		`{"interface":[{"name":"GigabitEthernet0/0/1","mac-address":"NOT-A-MAC"}]}`))
	if cerr != nil {
		t.Fatalf("convert: %v", cerr)
	}
	if err := validateAgainstSchema(s, anchor, typed); err == nil {
		t.Fatal("探针未被拒：校验链路没真正生效（锚点未解析？），上面的『全过』不可信")
	}
}
