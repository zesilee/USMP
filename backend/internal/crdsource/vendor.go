// vendor resolution for intent sources (TE-02, P5-1): the translator driver is
// selected by the target device's Vendor in the shared DeviceStore, no longer a
// hard-coded vendor constant at the call sites.
package crdsource

import (
	"github.com/leezesi/usmp/backend/pkg/translator"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
)

// vendorFor resolves the driver vendor for a device. Store miss / empty Vendor /
// nil store (minimal test rigs) all fall back to huawei — semantically equal to
// the pre-registry behavior (R08 降级). Labels are case-insensitive ("huawei" →
// VendorHuawei)；无法识别的标签原样透传，让 GetTranslator 报含厂商名的明确错误。
func vendorFor(ds device.Store, deviceID string) translator.VendorType {
	if ds == nil {
		return translator.VendorHuawei
	}
	info, ok := ds.Get(deviceID)
	if !ok || info.Vendor == "" {
		return translator.VendorHuawei
	}
	if vt, known := translator.VendorFromString(info.Vendor); known {
		return vt
	}
	return translator.VendorType(info.Vendor)
}
