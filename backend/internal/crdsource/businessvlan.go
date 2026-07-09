// Package crdsource wires business CRD types into Stack B's KubernetesCRDSource:
// it supplies the app-specific ProjectFunc (translate + extract deviceID/path)
// that maps a CRD to a desired ygot config, keeping the framework source generic.
package crdsource

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "github.com/leezesi/usmp/backend/api/biz/v1"
	"github.com/leezesi/usmp/backend/pkg/translator"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/source"
)

// VlanProjectFunc must satisfy the framework source's ProjectFunc contract.
var _ source.ProjectFunc = VlanProjectFunc

// VlanPath is the YANG path a BusinessVlan intent reconciles at — identical to the
// path the Huawei VLAN reconciler consumes, so the CRD-source and config-api paths
// converge on the same desired/actual reconciliation.
const VlanPath = "/vlan:vlan/vlan:vlans"

// VlanProjectFunc keeps the pre-registry fixed-huawei behavior（最小测试装配用）。
func VlanProjectFunc(obj client.Object) (deviceID, path string, desired interface{}, err error) {
	return NewVlanProjectFunc(nil)(obj)
}

// NewVlanProjectFunc builds a ProjectFunc that maps a BusinessVlan CR to its
// desired vendor ygot config, resolving the translator driver by the target
// device's Vendor in the DeviceStore（TE-02）；store miss / 空 Vendor 降级 huawei
// （R08，与既有行为语义等价）。It uses the SAME translator.TranslateConfig as the
// (legacy) Actor path, so the two paths produce semantically-equal desired config.
func NewVlanProjectFunc(ds device.Store) source.ProjectFunc {
	return func(obj client.Object) (deviceID, path string, desired interface{}, err error) {
		cr, ok := obj.(*apiv1.BusinessVlan)
		if !ok {
			return "", "", nil, fmt.Errorf("crdsource: expected *apiv1.BusinessVlan, got %T", obj)
		}
		desired, err = translator.TranslateConfig(vendorFor(ds, cr.Spec.DeviceID), translator.ConfigTypeVlan, cr.Spec)
		if err != nil {
			return "", "", nil, fmt.Errorf("crdsource: translate BusinessVlan %s: %w", cr.Name, err)
		}
		return cr.Spec.DeviceID, VlanPath, desired, nil
	}
}

// VlanObject returns the CRD prototype to watch for BusinessVlan sources.
func VlanObject() client.Object { return &apiv1.BusinessVlan{} }
