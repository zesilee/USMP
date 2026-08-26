package networkinstance

import (
	"fmt"
	"strings"
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/native/huawei"
	"github.com/leezesi/usmp/backend/internal/yangschema"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/object"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// 边界/负路径（NI-05）：ygot ΛValidate 按 schema 强制 name/description 的 length 与
// description 的 pattern '([^?]*)'（不含 '?'）。valid 通过、越界拒绝。

func niWith(name, desc string) *huawei.HuaweiNetworkInstance_NetworkInstance {
	return &huawei.HuaweiNetworkInstance_NetworkInstance{
		Instances: &huawei.HuaweiNetworkInstance_NetworkInstance_Instances{
			Instance: map[string]*huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance{
				name: {Name: object.String(name), Description: object.String(desc)},
			},
		},
	}
}

func TestNiValidate_Boundary(t *testing.T) {
	cases := []struct {
		name    string
		instNm  string
		desc    string
		wantErr bool
	}{
		{"valid", "vpn-a", "a normal description", false},
		{"name-min-1", "x", "d", false},
		{"name-max-31", strings.Repeat("n", 31), "d", false},
		{"name-over-31", strings.Repeat("n", 32), "d", true},
		{"name-empty", "", "d", true},
		{"desc-max-242", "vpn-a", strings.Repeat("d", 242), false},
		{"desc-over-242", "vpn-a", strings.Repeat("d", 243), true},
		{"desc-question-mark", "vpn-a", "bad?desc", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := irValidateNI(niWith(tc.instNm, tc.desc))
			if tc.wantErr && err == nil {
				t.Fatalf("期望校验失败（越界/非法），实际通过：name=%q desc len=%d", tc.instNm, len(tc.desc))
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("期望校验通过，实际失败: %v", err)
			}
		})
	}
}

// global ipv4 类型字段：合法 ipv4 通过。（非法 ipv4 由 ygot 类型/pattern 兜，
// 此处正路径冒烟确保 global 校验链通。）
func TestNiValidate_GlobalIPv4(t *testing.T) {
	ni := &huawei.HuaweiNetworkInstance_NetworkInstance{
		Global: &huawei.HuaweiNetworkInstance_NetworkInstance_Global{
			CfgRouterId:              object.String("10.0.0.1"),
			RouteDistinguisherAutoIp: object.String("10.0.0.2"),
			AsNotationPlain:          object.Bool(true),
		},
	}
	if err := irValidateNI(ni); err != nil {
		t.Fatalf("合法 global 应通过校验: %v", err)
	}
}

// irValidateNI 以 IR 校验器替代已退役的 ygot ΛValidate（pattern/length/range
// 同为其覆盖面；retire-ygot-runtime S3 切换）。
func irValidateNI(ni *huawei.HuaweiNetworkInstance_NetworkInstance) error {
	s, err := yangschema.Load()
	if err != nil {
		return err
	}
	n, ok := s.Path("/network-instance")
	if !ok {
		return fmt.Errorf("network-instance schema node missing")
	}
	return schema.ValidateObject(n, ni)
}
