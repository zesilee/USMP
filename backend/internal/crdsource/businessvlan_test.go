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

func sampleVlanCR() *apiv1.BusinessVlan {
	return &apiv1.BusinessVlan{
		ObjectMeta: metav1.ObjectMeta{Name: "vlan-100"},
		Spec: apiv1.BusinessVlanSpec{
			DeviceID: "10.0.0.1:830",
			VlanID:   100,
			Name:     "office",
		},
	}
}

// TestVlanProjectFunc: a BusinessVlan CR projects to the correct deviceID/path and
// a Huawei VLAN ygot desired.
func TestVlanProjectFunc(t *testing.T) {
	deviceID, path, desired, err := VlanProjectFunc(sampleVlanCR())
	if err != nil {
		t.Fatalf("VlanProjectFunc: %v", err)
	}
	if deviceID != "10.0.0.1:830" || path != VlanPath {
		t.Fatalf("deviceID/path = %s/%s, want 10.0.0.1:830/%s", deviceID, path, VlanPath)
	}
	vlans, ok := desired.(*huawei.HuaweiVlan_Vlan_Vlans)
	if !ok || vlans.Vlan[100] == nil {
		t.Fatalf("desired not a Huawei VLAN struct with vlan 100: %T", desired)
	}
}

// TestVlanProjectEquivalentToActorTranslation: the CRD-source desired equals the
// (legacy) Actor path's translation of the same Spec — both call the same
// translator.TranslateConfig (double-path desired equivalence, task 2.2).
func TestVlanProjectEquivalentToActorTranslation(t *testing.T) {
	cr := sampleVlanCR()
	_, _, crdDesired, err := VlanProjectFunc(cr)
	if err != nil {
		t.Fatal(err)
	}
	actorDesired, err := translator.TranslateConfig(translator.VendorHuawei, translator.ConfigTypeVlan, cr.Spec)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(crdDesired, actorDesired) {
		t.Fatal("CRD-source desired differs from Actor-path translation")
	}
}

// TestVlanProjectWrongType: a non-BusinessVlan object errors.
func TestVlanProjectWrongType(t *testing.T) {
	var notVlan client.Object = &apiv1.BusinessInterface{}
	if _, _, _, err := VlanProjectFunc(notVlan); err == nil {
		t.Fatal("expected error for wrong CR type")
	}
}
