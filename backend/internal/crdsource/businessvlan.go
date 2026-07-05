// Package crdsource wires business CRD types into Stack B's KubernetesCRDSource:
// it supplies the app-specific ProjectFunc (translate + extract deviceID/path)
// that maps a CRD to a desired ygot config, keeping the framework source generic.
package crdsource

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "github.com/leezesi/usmp/backend/api/biz/v1"
	"github.com/leezesi/usmp/backend/pkg/translator"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/source"
)

// VlanProjectFunc must satisfy the framework source's ProjectFunc contract.
var _ source.ProjectFunc = VlanProjectFunc

// VlanPath is the YANG path a BusinessVlan intent reconciles at — identical to the
// path the Huawei VLAN reconciler consumes, so the CRD-source and config-api paths
// converge on the same desired/actual reconciliation.
const VlanPath = "/vlan:vlan/vlan:vlans"

// VlanProjectFunc maps a BusinessVlan CR to its desired Huawei VLAN ygot config.
// It uses the SAME translator.TranslateConfig(Huawei, Vlan, Spec) as the (legacy)
// Actor path, so the two paths produce semantically-equal desired config.
func VlanProjectFunc(obj client.Object) (deviceID, path string, desired interface{}, err error) {
	cr, ok := obj.(*apiv1.BusinessVlan)
	if !ok {
		return "", "", nil, fmt.Errorf("crdsource: expected *apiv1.BusinessVlan, got %T", obj)
	}
	desired, err = translator.TranslateConfig(translator.VendorHuawei, translator.ConfigTypeVlan, cr.Spec)
	if err != nil {
		return "", "", nil, fmt.Errorf("crdsource: translate BusinessVlan %s: %w", cr.Name, err)
	}
	return cr.Spec.DeviceID, VlanPath, desired, nil
}

// VlanObject returns the CRD prototype to watch for BusinessVlan sources.
func VlanObject() client.Object { return &apiv1.BusinessVlan{} }
