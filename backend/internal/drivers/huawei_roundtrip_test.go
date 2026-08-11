package drivers

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/leezesi/usmp/backend/internal/testutil/yangsample"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/object"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/xmlcodec"
)

// TestFullOnboardingEncodeDecodeRoundtrip 对每个表行模块构造最小真值实例，
// Encode→Decode 断言相等（T02b 参数化矩阵之编解码往返；无可赋值标量的模块
// 走空容器往返，保证 namespace/根元素管线不缺）。
func TestFullOnboardingEncodeDecodeRoundtrip(t *testing.T) {
	for _, pm := range plainModules {
		pm := pm
		t.Run(pm.module, func(t *testing.T) {
			src := pm.newFn()
			spec := &xmlcodec.Spec{Namespace: pm.ns, Schema: specSchemaOf(t, pm)}
			entry := spec.Schema()
			if entry == nil {
				t.Fatalf("SchemaTree 入口缺失")
			}
			populated := yangsample.Populate(src, entry)

			xml, err := xmlcodec.Encode(spec, src)
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			dst := pm.newFn()
			if err := xmlcodec.Decode(spec, []byte(xml), dst); err != nil {
				t.Fatalf("Decode: %v", err)
			}
			eq, err := jsonEqual(src, dst)
			if err != nil {
				t.Fatalf("diff: %v", err)
			}
			if !eq {
				t.Fatalf("往返不相等（populated=%v）\nXML: %s", populated, xml)
			}
		})
	}
}

func specSchemaOf(t *testing.T, pm plainModule) func() schema.Node {
	t.Helper()
	return irNode("/" + pm.module)
}

func jsonEqual(a, b object.Object) (bool, error) {
	am, ok := a.(json.Marshaler)
	if !ok {
		return false, fmt.Errorf("%T lacks MarshalJSON", a)
	}
	bm, ok := b.(json.Marshaler)
	if !ok {
		return false, fmt.Errorf("%T lacks MarshalJSON", b)
	}
	ab, err := am.MarshalJSON()
	if err != nil {
		return false, err
	}
	bb, err := bm.MarshalJSON()
	if err != nil {
		return false, err
	}
	return string(ab) == string(bb), nil
}
