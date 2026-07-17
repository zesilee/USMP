package api

import (
	"fmt"
	"strings"
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
)

func managerNew() manager.Manager { return manager.New() }

// BR-06/DR-05（矩阵 B1）：锚点相对包裹 + 单一 RFC7951 解码路径。
// 正常（零包裹/单层/多层）· 负路径（非前缀/谓词/未注册/旧形状）· 错误透出。

func TestConvertConfig_AnchorWrap(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		data     map[string]interface{}
		wantType interface{}
		wantErr  string // 空=成功
	}{
		{
			name:     "path=锚点零包裹（vlan list 流）",
			path:     "/vlan:vlan/vlan:vlans",
			data:     map[string]interface{}{"vlan": []interface{}{map[string]interface{}{"id": float64(100), "name": "V100"}}},
			wantType: &huawei.HuaweiVlan_Vlan_Vlans{},
		},
		{
			name:     "子路径单层包裹（system form-tab 扁平载荷）",
			path:     "/system:system/system:system-info",
			data:     map[string]interface{}{"sys-name": "sw-01"},
			wantType: &huawei.HuaweiSystem_System{},
		},
		{
			name:     "ifm list 流",
			path:     "/ifm:ifm/ifm:interfaces",
			data:     map[string]interface{}{"interface": []interface{}{map[string]interface{}{"name": "GE0/0/1"}}},
			wantType: &huawei.HuaweiIfm_Ifm_Interfaces{},
		},
		{
			name:    "旧复数键形状拒绝",
			path:    "/vlan:vlan/vlan:vlans",
			data:    map[string]interface{}{"vlans": []interface{}{map[string]interface{}{"id": float64(100)}}},
			wantErr: "vlans",
		},
		{
			name:    "camelCase 旧叶名拒绝",
			path:    "/system:system/system:system-info",
			data:    map[string]interface{}{"sysName": "sw-01"},
			wantErr: "sysName",
		},
		{
			name:    "未注册路径显式 400 语义",
			path:    "/nowhere:x/nowhere:y",
			data:    map[string]interface{}{"a": "b"},
			wantErr: "未注册",
		},
		{
			name:    "path 段含 list 谓词拒绝",
			path:    "/vlan:vlan/vlan:vlans/vlan:vlan[id=100]",
			data:    map[string]interface{}{"name": "V100"},
			wantErr: "谓词",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v, err := convertConfig(c.path, c.data)
			if c.wantErr != "" {
				if err == nil {
					t.Fatalf("want error containing %q, got value %T", c.wantErr, v)
				}
				if !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("error %q should contain %q", err.Error(), c.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if v == nil {
				t.Fatal("nil value")
			}
			got, want := typeName(v), typeName(c.wantType)
			if got != want {
				t.Fatalf("type = %s, want %s", got, want)
			}
		})
	}
}

func typeName(v interface{}) string { return fmt.Sprintf("%T", v) }

// BR-05 补全：子路径下发的 desired 必须归一化存储在描述符锚点路径下——
// 周期对账（模块路径入队）才能看到它；请求路径只用于定位与包裹。
func TestSetConfig_SubpathStoresAtAnchor(t *testing.T) {
	mgr := managerNew()
	h := NewConfigHandler(mgr)

	w := postConfigRaw(h, "10.0.0.9", "/system:system/system:system-info", "", `{"sys-name":"anchor-sw"}`)
	env := decodeLockEnvelope(t, w)
	if env.Code != 0 {
		t.Fatalf("subpath post rejected: %s", w.Body.String())
	}
	// desired 在锚点路径可见
	desired, err := mgr.GetConfigStore().Get("10.0.0.9", "/system:system")
	if err != nil || desired == nil {
		t.Fatalf("desired not stored at anchor: %v %v", desired, err)
	}
	sys, ok := desired.(*huawei.HuaweiSystem_System)
	if !ok || sys.SystemInfo == nil || sys.SystemInfo.SysName == nil || *sys.SystemInfo.SysName != "anchor-sw" {
		t.Fatalf("anchor desired wrong: %#v", desired)
	}
	// 子路径 key 不留分叉副本
	if v, _ := mgr.GetConfigStore().Get("10.0.0.9", "/system:system/system:system-info"); v != nil {
		t.Fatalf("subpath key should not hold a fork: %#v", v)
	}
}
