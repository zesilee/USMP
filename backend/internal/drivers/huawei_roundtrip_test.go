package drivers

import (
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/internal/testutil/yangsample"
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
			eq, err := ygotDiffEmpty(src, dst)
			if err != nil {
				t.Fatalf("diff: %v", err)
			}
			if !eq {
				t.Fatalf("往返不相等（populated=%v）\nXML: %s", populated, xml)
			}
		})
	}
}

func specSchemaOf(t *testing.T, pm plainModule) func() *yang.Entry {
	t.Helper()
	key := schemaKeyOf(pm.newFn)
	return func() *yang.Entry { return huawei.SchemaTree[key] }
}

func ygotDiffEmpty(a, b ygot.GoStruct) (bool, error) {
	n, err := ygot.Diff(a, b)
	if err != nil {
		return false, err
	}
	return len(n.GetUpdate()) == 0 && len(n.GetDelete()) == 0, nil
}
