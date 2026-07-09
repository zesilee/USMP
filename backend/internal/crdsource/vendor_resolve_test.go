package crdsource

import (
	"strings"
	"testing"

	yangclient "github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
)

// TE-02: 意图翻译按设备在 DeviceStore 中的 Vendor 解析驱动，不再硬编码厂商常量。

// 命中 huawei 设备：行为与既有完全等价。
func TestNewVlanProjectFunc_ResolvesHuaweiFromStore(t *testing.T) {
	ds := device.NewStore()
	ds.Put("192.168.1.10", yangclient.DeviceConnectionInfo{IP: "192.168.1.10", Vendor: "huawei"})

	project := NewVlanProjectFunc(ds)
	deviceID, path, desired, err := project(sampleVlanCR())
	if err != nil {
		t.Fatalf("huawei 设备翻译应成功: %v", err)
	}
	if deviceID == "" || path != VlanPath || desired == nil {
		t.Fatalf("投影结果应完整: id=%q path=%q", deviceID, path)
	}
}

// 设备未注册 / Vendor 为空：降级 huawei（R08，等价存量行为）。
func TestNewVlanProjectFunc_StoreMissDefaultsHuawei(t *testing.T) {
	project := NewVlanProjectFunc(device.NewStore())
	if _, _, _, err := project(sampleVlanCR()); err != nil {
		t.Fatalf("store miss 应降级 huawei 继续翻译: %v", err)
	}

	// nil store（最小测试装配）同样降级。
	projectNil := NewVlanProjectFunc(nil)
	if _, _, _, err := projectNil(sampleVlanCR()); err != nil {
		t.Fatalf("nil store 应降级 huawei 继续翻译: %v", err)
	}
}

// 设备标注无驱动厂商：错误明确透出（证明 Vendor 真被消费，而非摆设）。
func TestNewVlanProjectFunc_UnsupportedVendorSurfaces(t *testing.T) {
	ds := device.NewStore()
	cr := sampleVlanCR()
	ds.Put(cr.Spec.DeviceID, yangclient.DeviceConnectionInfo{IP: cr.Spec.DeviceID, Vendor: "nokia"})

	if _, _, _, err := NewVlanProjectFunc(ds)(cr); err == nil || !strings.Contains(err.Error(), "nokia") {
		t.Fatalf("无驱动厂商应明确报错并含厂商名, got: %v", err)
	}
}

// Interface 源同构（两调用点都要切换）。
func TestNewInterfaceProjectFunc_ResolvesVendor(t *testing.T) {
	ds := device.NewStore()
	cr := sampleInterfaceCR()
	ds.Put(cr.Spec.DeviceID, yangclient.DeviceConnectionInfo{IP: cr.Spec.DeviceID, Vendor: "nokia"})

	if _, _, _, err := NewInterfaceProjectFunc(ds)(cr); err == nil {
		t.Fatal("Interface 源同样应按 Vendor 解析并对无驱动厂商报错")
	}
	if _, _, _, err := NewInterfaceProjectFunc(nil)(sampleInterfaceCR()); err != nil {
		t.Fatalf("nil store 降级 huawei: %v", err)
	}
}
