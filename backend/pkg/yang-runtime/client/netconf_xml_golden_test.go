package client

import (
	"testing"

	"github.com/leezesi/usmp/backend/internal/testutil/hwfix"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/xmlcodec"
)

// goldenCases 枚举 golden fixture 与「现有手写 builder」的产出函数（D3 冻结基线）。
// 通用引擎（xmlcodec）测试以同一批 fixture 对拍同一批 golden 文件。
// 首次冻结 / 刷新：go test ./pkg/yang-runtime/client/ -run TestGoldenXML -args -update-golden
var goldenCases = []struct {
	name    string
	produce func() (string, error)
}{
	{"vlan_full", func() (string, error) { return buildHuaweiVlanVlansXML(hwfix.VlanFull()) }},
	{"vlan_minimal", func() (string, error) { return buildHuaweiVlanVlansXML(hwfix.VlanMinimal()) }},
	{"vlan_empty", func() (string, error) { return buildHuaweiVlanVlansXML(hwfix.VlanEmpty()) }},
	{"vlan_escape", func() (string, error) { return buildHuaweiVlanVlansXML(hwfix.VlanEscape()) }},
	{"ifm_full", func() (string, error) { return buildHuaweiIfmInterfacesXML(hwfix.IfmFull()) }},
	{"ifm_minimal", func() (string, error) { return buildHuaweiIfmInterfacesXML(hwfix.IfmMinimal()) }},
	{"ifm_empty", func() (string, error) { return buildHuaweiIfmInterfacesXML(hwfix.IfmEmpty()) }},
	{"delete_vlan", func() (string, error) { return marshalDeleteChange(hwfix.VlanDeleteSet()) }},
	{"delete_ifm", func() (string, error) { return marshalDeleteChange(hwfix.IfmDeleteSet()) }},
}

// TestGoldenXMLLegacyBuilders 冻结并验证既有手写 builder 输出（任务 1.3/1.4）。
// 规范化（同级全排序 + 相同同级去重，见 xmlcodec.Canonicalize doc）吸收 map 迭代
// 序非确定与 <suppression> 历史重复发送，因此无需豁免机制即全绿。
func TestGoldenXMLLegacyBuilders(t *testing.T) {
	for _, tc := range goldenCases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := tc.produce()
			if err != nil {
				t.Fatalf("legacy builder: %v", err)
			}
			canon, err := xmlcodec.Canonicalize([]byte(out))
			if err != nil {
				t.Fatalf("canonicalize legacy output: %v\nraw: %s", err, out)
			}
			if *hwfix.Update {
				hwfix.WriteGolden(t, tc.name, canon)
				return
			}
			if want := hwfix.Golden(t, tc.name); canon != want {
				t.Errorf("legacy builder output drifted from golden %s\n got: %s\nwant: %s", tc.name, canon, want)
			}
		})
	}
}
