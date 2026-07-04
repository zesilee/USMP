package crdsource

import (
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "github.com/leezesi/usmp/backend/api/v1"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/translator"
)

func sampleInterfaceCR() *apiv1.BusinessInterface {
	return &apiv1.BusinessInterface{
		ObjectMeta: metav1.ObjectMeta{Name: "ge-1"},
		Spec: apiv1.BusinessInterfaceSpec{
			DeviceID:      "10.0.0.2:830",
			InterfaceName: "GigabitEthernet0/0/1",
			Description:   "uplink",
		},
	}
}

// TestInterfaceProjectFunc: a BusinessInterface CR projects to the correct
// deviceID/path and a Huawei ifm ygot desired.
func TestInterfaceProjectFunc(t *testing.T) {
	deviceID, path, desired, err := InterfaceProjectFunc(sampleInterfaceCR())
	if err != nil {
		t.Fatalf("InterfaceProjectFunc: %v", err)
	}
	if deviceID != "10.0.0.2:830" || path != InterfacePath {
		t.Fatalf("deviceID/path = %s/%s, want 10.0.0.2:830/%s", deviceID, path, InterfacePath)
	}
	ifaces, ok := desired.(*huawei.HuaweiIfm_Ifm_Interfaces)
	if !ok || ifaces.Interface["GigabitEthernet0/0/1"] == nil {
		t.Fatalf("desired not a Huawei ifm struct with the interface: %T", desired)
	}
}

// TestInterfaceProjectEquivalentToActorTranslation: the CRD-source desired equals
// the Actor path's translation of the same Spec (double-path equivalence, 3.1).
func TestInterfaceProjectEquivalentToActorTranslation(t *testing.T) {
	cr := sampleInterfaceCR()
	_, _, crdDesired, err := InterfaceProjectFunc(cr)
	if err != nil {
		t.Fatal(err)
	}
	actorDesired, err := translator.TranslateConfig(translator.VendorHuawei, translator.ConfigTypeInterface, cr.Spec)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(crdDesired, actorDesired) {
		t.Fatal("CRD-source desired differs from Actor-path translation")
	}
}

// TestInterfaceProjectWrongType: a non-BusinessInterface object errors.
func TestInterfaceProjectWrongType(t *testing.T) {
	var notIface client.Object = &apiv1.BusinessVlan{}
	if _, _, _, err := InterfaceProjectFunc(notIface); err == nil {
		t.Fatal("expected error for wrong CR type")
	}
}
