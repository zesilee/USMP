package xmlcodec

import (
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/openconfig/ygot/ygot"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
)

const devmNS = "urn:huawei:yang:huawei-devm"

func devmSpec() *Spec {
	return &Spec{
		Namespace: devmNS,
		Schema:    irTestNode("/devm"),
	}
}

func devmEntitysSpec() *Spec {
	return &Spec{
		Namespace: devmNS,
		Schema:    irTestNode("/devm/physical-entitys"),
	}
}

// 真机回读形态（rpc-reply/data 包裹 + 模块顶层容器），两个三键条目
// （class+position+serial-number，含 enum 键叶）——复现 devm 空表回归的原始报文形状。
const devmMultiKeyDoc = `<rpc-reply><data><devm xmlns="` + devmNS + `"><physical-entitys>` +
	`<physical-entity><class>chassis</class><position>1</position><serial-number>SN-A</serial-number><name>Chassis 1</name><is-fru>true</is-fru></physical-entity>` +
	`<physical-entity><class>port</class><position>1/0/1</position><serial-number>SN-B</serial-number><name>GE1/0/1</name></physical-entity>` +
	`</physical-entitys></devm></data></rpc-reply>`

// TestDecodeMultiKeyListRoot：多键列表作为解码根（list 模式）——XC-02 多键回归。
// 修复前此处报 `list physical-entity: multi-key lists unsupported`（decode.go:298）。
func TestDecodeMultiKeyListRoot(t *testing.T) {
	got := &huawei.HuaweiDevm_Devm_PhysicalEntitys{}
	if err := Decode(devmEntitysSpec(), []byte(devmMultiKeyDoc), got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(got.PhysicalEntity) != 2 {
		t.Fatalf("want 2 entries, got %d: %+v", len(got.PhysicalEntity), got.PhysicalEntity)
	}
	kA := huawei.HuaweiDevm_Devm_PhysicalEntitys_PhysicalEntity_Key{
		Class: huawei.HuaweiDevm_EntityClassType_chassis, Position: "1", SerialNumber: "SN-A",
	}
	eA := got.PhysicalEntity[kA]
	if eA == nil {
		t.Fatalf("entry for key %+v missing, keys: %v", kA, mapKeys(got.PhysicalEntity))
	}
	if eA.Name == nil || *eA.Name != "Chassis 1" || eA.IsFru == nil || !*eA.IsFru {
		t.Errorf("entry A fields wrong: %+v", eA)
	}
	if eA.Class != huawei.HuaweiDevm_EntityClassType_chassis || eA.Position == nil || *eA.Position != "1" {
		t.Errorf("entry A key leaves not populated on entry: %+v", eA)
	}
	kB := huawei.HuaweiDevm_Devm_PhysicalEntitys_PhysicalEntity_Key{
		Class: huawei.HuaweiDevm_EntityClassType_port, Position: "1/0/1", SerialNumber: "SN-B",
	}
	if eB := got.PhysicalEntity[kB]; eB == nil || eB.Name == nil || *eB.Name != "GE1/0/1" {
		t.Errorf("entry B missing or wrong: %+v", got.PhysicalEntity[kB])
	}
}

// TestDecodeMultiKeyNested：多键列表作为嵌套 list（container 模式经 decodeField
// Map 分支）——decodeRunningConfig 真实链路的解码根是模块根容器 HuaweiDevm_Devm。
func TestDecodeMultiKeyNested(t *testing.T) {
	got := &huawei.HuaweiDevm_Devm{}
	if err := Decode(devmSpec(), []byte(devmMultiKeyDoc), got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.PhysicalEntitys == nil || len(got.PhysicalEntitys.PhysicalEntity) != 2 {
		t.Fatalf("want 2 nested entries, got: %+v", got.PhysicalEntitys)
	}
	k := huawei.HuaweiDevm_Devm_PhysicalEntitys_PhysicalEntity_Key{
		Class: huawei.HuaweiDevm_EntityClassType_chassis, Position: "1", SerialNumber: "SN-A",
	}
	if e := got.PhysicalEntitys.PhysicalEntity[k]; e == nil || e.Name == nil || *e.Name != "Chassis 1" {
		t.Errorf("nested entry missing or wrong: %+v", got.PhysicalEntitys.PhysicalEntity)
	}
}

// TestDecodeMultiKeyMissingKeyLeaf：部分键叶缺失（ΛListKeyMap 报错）→ 宽容语义：
// 以「已有键字段 + 缺失字段零值」构造 key struct 保留条目，不丢行（对齐单键合成 key）。
func TestDecodeMultiKeyMissingKeyLeaf(t *testing.T) {
	doc := `<devm xmlns="` + devmNS + `"><physical-entitys>` +
		`<physical-entity><class>fan</class><position>2</position><name>FAN 2</name></physical-entity>` +
		`</physical-entitys></devm>`
	got := &huawei.HuaweiDevm_Devm_PhysicalEntitys{}
	if err := Decode(devmEntitysSpec(), []byte(doc), got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	k := huawei.HuaweiDevm_Devm_PhysicalEntitys_PhysicalEntity_Key{
		Class: huawei.HuaweiDevm_EntityClassType_fan, Position: "2", SerialNumber: "",
	}
	e := got.PhysicalEntity[k]
	if e == nil {
		t.Fatalf("entry with missing key leaf dropped, keys: %v", mapKeys(got.PhysicalEntity))
	}
	if e.Name == nil || *e.Name != "FAN 2" {
		t.Errorf("entry fields wrong: %+v", e)
	}
}

// TestDecodeMultiKeyEmpty：空回读边界——多键列表容器同样返回非 nil 空 map。
func TestDecodeMultiKeyEmpty(t *testing.T) {
	got := &huawei.HuaweiDevm_Devm_PhysicalEntitys{}
	if err := Decode(devmEntitysSpec(), nil, got); err != nil {
		t.Fatalf("Decode empty: %v", err)
	}
	if got.PhysicalEntity == nil || len(got.PhysicalEntity) != 0 {
		t.Fatalf("want initialized empty map, got: %+v", got.PhysicalEntity)
	}
}

// TestDecodeMultiKeyConcurrent：多键解码并发安全（-race 下跑，对齐 TestDecodeConcurrent）。
func TestDecodeMultiKeyConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got := &huawei.HuaweiDevm_Devm_PhysicalEntitys{}
			if err := Decode(devmEntitysSpec(), []byte(devmMultiKeyDoc), got); err != nil {
				t.Errorf("Decode: %v", err)
				return
			}
			if len(got.PhysicalEntity) != 2 {
				t.Errorf("want 2 entries, got %d", len(got.PhysicalEntity))
			}
		}()
	}
	wg.Wait()
}

// --- 负路径：key struct 字段与 ΛListKeyMap 值不可转换（生成物不一致的异常形态）---

type badKey struct {
	A string `path:"a"`
	B string `path:"b"`
}

type badEntry struct {
	A *string `path:"a"`
	B *string `path:"b"`
}

func (*badEntry) IsYANGGoStruct() {}
func (e *badEntry) ΛListKeyMap() (map[string]interface{}, error) {
	// "b" 返回与 key struct 字段（string）不可转换的类型
	return map[string]interface{}{"a": "x", "b": struct{ X int }{1}}, nil
}
func (*badEntry) ΛValidate(...ygot.ValidationOption) error { return nil }
func (*badEntry) ΛEnumTypeMap() map[string][]reflect.Type  { return nil }
func (*badEntry) ΛBelongingModule() string                 { return "" }

// TestEntryKeyMultiKeyNotConvertible：不可转换 SHALL 返回命名该 list 的明确错误
// （R08，不 panic、不静默错键）。
func TestEntryKeyMultiKeyNotConvertible(t *testing.T) {
	entry := reflect.ValueOf(&badEntry{})
	_, err := entryKey(entry, reflect.TypeOf(badKey{}), "bad-list", 0)
	if err == nil {
		t.Fatal("want error for non-convertible multi-key value, got nil")
	}
	if !strings.Contains(err.Error(), "bad-list") {
		t.Errorf("error should name the list, got: %v", err)
	}
}

// TestEncodeDeleteMultiKeyStillUnsupported：XC-03 范围注记守护——删除通道的多键
// 不支持契约保持不变（明确错误，不发送错键 delete），解码侧多键支持不外溢到删除侧。
func TestEncodeDeleteMultiKeyStillUnsupported(t *testing.T) {
	sn := "SN-A"
	pos := "1"
	set := &huawei.HuaweiDevm_Devm_PhysicalEntitys{
		PhysicalEntity: map[huawei.HuaweiDevm_Devm_PhysicalEntitys_PhysicalEntity_Key]*huawei.HuaweiDevm_Devm_PhysicalEntitys_PhysicalEntity{
			{Class: huawei.HuaweiDevm_EntityClassType_chassis, Position: "1", SerialNumber: "SN-A"}: {
				Class: huawei.HuaweiDevm_EntityClassType_chassis, Position: &pos, SerialNumber: &sn,
			},
		},
	}
	if _, err := EncodeDelete(devmEntitysSpec(), set); err == nil || !strings.Contains(err.Error(), "multi-key") {
		t.Fatalf("want explicit multi-key unsupported error, got: %v", err)
	}
}

func mapKeys(m interface{}) []interface{} {
	v := reflect.ValueOf(m)
	out := make([]interface{}, 0, v.Len())
	for _, k := range v.MapKeys() {
		out = append(out, k.Interface())
	}
	return out
}
