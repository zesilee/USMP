package crdsource

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "github.com/leezesi/usmp/backend/api/biz/v1"
	"github.com/leezesi/usmp/backend/pkg/translator"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/source"
)

// InterfaceProjectFunc must satisfy the framework source's ProjectFunc contract.
var _ source.ProjectFunc = InterfaceProjectFunc

// InterfacePath is the YANG path a BusinessInterface intent reconciles at —
// identical to the Huawei interface reconciler's path.
const InterfacePath = "/ifm:ifm/ifm:interfaces"

// InterfaceProjectFunc keeps the pre-registry fixed-huawei behavior（最小测试装配用）。
func InterfaceProjectFunc(obj client.Object) (deviceID, path string, desired interface{}, err error) {
	return NewInterfaceProjectFunc(nil)(obj)
}

// NewInterfaceProjectFunc builds a ProjectFunc that maps a BusinessInterface CR to
// its desired vendor ifm ygot config, resolving the driver by the device's Vendor
// in the DeviceStore（TE-02；miss 降级 huawei，R08）。Uses the same
// translator.TranslateConfig as the legacy Actor path (semantically-equal desired).
func NewInterfaceProjectFunc(ds device.Store) source.ProjectFunc {
	return func(obj client.Object) (deviceID, path string, desired interface{}, err error) {
		cr, ok := obj.(*apiv1.BusinessInterface)
		if !ok {
			return "", "", nil, fmt.Errorf("crdsource: expected *apiv1.BusinessInterface, got %T", obj)
		}
		desired, err = translator.TranslateConfig(vendorFor(ds, cr.Spec.DeviceID), translator.ConfigTypeInterface, cr.Spec)
		if err != nil {
			return "", "", nil, fmt.Errorf("crdsource: translate BusinessInterface %s: %w", cr.Name, err)
		}
		return cr.Spec.DeviceID, InterfacePath, desired, nil
	}
}

// InterfaceObject returns the CRD prototype to watch for BusinessInterface sources.
func InterfaceObject() client.Object { return &apiv1.BusinessInterface{} }
