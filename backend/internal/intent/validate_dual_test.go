package intent

import (
	"testing"

	"github.com/leezesi/usmp/backend/internal/yangschema"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/validate"
)

// 任务5.2 双跑一致门：IR 校验器对全部快照用例与 ygot Validate 结论一致
// （接受/拒绝逐用例相同），一致后 cr.go 方可切换（W02 同精神）。
func TestValidateSnapshotIRDual(t *testing.T) {
	s, err := yangschema.Load()
	if err != nil {
		t.Fatal(err)
	}
	node, ok := s.Path("/business-vlan-service")
	if !ok {
		t.Fatal("business-vlan-service schema node missing")
	}
	for name, tc := range validateSnapshotCases() {
		t.Run(name, func(t *testing.T) {
			root := decodeBusinessDevice(t, tc.spec)
			var err error
			if root == nil {
				err = errRejectedAtDecode
			} else {
				err = validate.Object(node, root.BusinessVlanService)
			}
			if (err == nil) != tc.ok {
				t.Fatalf("IR 校验器与快照失配: want ok=%v, got err=%v", tc.ok, err)
			}
		})
	}
}
