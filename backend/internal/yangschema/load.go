// Package yangschema builds the framework YANG schema tree from the generated
// ygot models (huawei + openconfig). This is the offline fallback source for the
// hybrid schema resolution: device NETCONF capabilities narrow the usable module
// set at runtime, while attribute-level schema comes from these generated models
// (R04: schema derived from ygot-generated models, not hand-written).
package yangschema

import (
	"fmt"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/internal/generated/openconfig"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// Load builds a Schema containing the huawei and openconfig modules from their
// generated ygot schemas.
func Load() (schema.Schema, error) {
	ds := schema.NewSchema()

	hs, err := huawei.Schema()
	if err != nil {
		return nil, fmt.Errorf("load huawei schema: %w", err)
	}
	schema.AddYgotSchemaWithVendor(ds, hs, "huawei")

	os, err := openconfig.Schema()
	if err != nil {
		return nil, fmt.Errorf("load openconfig schema: %w", err)
	}
	schema.AddYgotSchemaWithVendor(ds, os, "openconfig")

	return ds, nil
}
