package crdsource

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "github.com/leezesi/usmp/backend/api/biz/v1"
	"github.com/leezesi/usmp/backend/pkg/translator"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/source"
)

// InterfaceProjectFunc must satisfy the framework source's ProjectFunc contract.
var _ source.ProjectFunc = InterfaceProjectFunc

// InterfacePath is the YANG path a BusinessInterface intent reconciles at —
// identical to the Huawei interface reconciler's path.
const InterfacePath = "/ifm:ifm/ifm:interfaces"

// InterfaceProjectFunc maps a BusinessInterface CR to its desired Huawei ifm ygot
// config, using the same translator.TranslateConfig(Huawei, Interface, Spec) as the
// legacy Actor path (so the two paths produce semantically-equal desired config).
func InterfaceProjectFunc(obj client.Object) (deviceID, path string, desired interface{}, err error) {
	cr, ok := obj.(*apiv1.BusinessInterface)
	if !ok {
		return "", "", nil, fmt.Errorf("crdsource: expected *apiv1.BusinessInterface, got %T", obj)
	}
	desired, err = translator.TranslateConfig(translator.VendorHuawei, translator.ConfigTypeInterface, cr.Spec)
	if err != nil {
		return "", "", nil, fmt.Errorf("crdsource: translate BusinessInterface %s: %w", cr.Name, err)
	}
	return cr.Spec.DeviceID, InterfacePath, desired, nil
}

// InterfaceObject returns the CRD prototype to watch for BusinessInterface sources.
func InterfaceObject() client.Object { return &apiv1.BusinessInterface{} }
