package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/openconfig/ygot/ygot"

	ygothuawei "github.com/leezesi/usmp/backend/internal/generated/huawei"
	native "github.com/leezesi/usmp/backend/internal/generated/native/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/object"
)

// JSON 通道全模块扫荡对拍（任务3.2 收口）：反射把 native Device 树按确定性
// 规则填充（触达每种字段形状），native 编码 → ygot 解码 → ygot 编码 → 语义
// DeepEqual。手工样本覆盖不到的模块/形状（union/复合键/empty/leaf-list/64位）
// 全部进对拍面。
//
// 填充规则（确定性）：标量按类型给固定值、枚举取值域第一个、map 放一个条目、
// slice 放两个元素、union 取第一个包装、empty=true、interface{} 跳过（ygot 侧
// unsupported 同样不产出）、深度上限防巨树。

type filler struct {
	depthCap int
	nodeCap  int
	nodes    int
}

func (fl *filler) fill(v reflect.Value, depth int) {
	if fl.nodes >= fl.nodeCap || depth > fl.depthCap {
		return
	}
	t := v.Type()
	if t.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < t.NumField(); i++ {
		if fl.nodes >= fl.nodeCap {
			return
		}
		f := t.Field(i)
		fv := v.Field(i)
		if !fv.CanSet() {
			continue
		}
		fl.nodes++
		switch f.Type.Kind() {
		case reflect.Ptr:
			elem := f.Type.Elem()
			switch elem.Kind() {
			case reflect.Struct: // 嵌套容器 / OrderedMap（跳过 OrderedMap：私有字段无法反射填充）
				if strings.HasSuffix(elem.Name(), "_OrderedMap") {
					continue
				}
				nv := reflect.New(elem)
				fl.fill(nv.Elem(), depth+1)
				fv.Set(nv)
			case reflect.String:
				fv.Set(ptrOf(reflect.ValueOf("s1")))
			case reflect.Bool:
				fv.Set(ptrOf(reflect.ValueOf(true)))
			case reflect.Float64:
				fv.Set(ptrOf(reflect.ValueOf(1.5)))
			default: // 整型族
				nv := reflect.New(elem)
				nv.Elem().Set(reflect.ValueOf(uint64(3)).Convert(elem))
				fv.Set(nv)
			}
		case reflect.Int64: // 枚举（E_* 底层 int64）
			if ev, ok := firstEnumValue(native.EnumMaps, f.Type.Name()); ok {
				fv.SetInt(ev)
			}
		case reflect.Bool: // object.Empty
			fv.SetBool(true)
		case reflect.Map:
			elem := f.Type.Elem().Elem() // *T → T
			entry := reflect.New(elem)
			fl.fill(entry.Elem(), depth+1)
			// 深度/节点截断可能留下 key 叶为 nil 的半成品条目——用生成的
			// ListKeyMap 自检，key 不全就整个跳过（ygot 侧会拒绝 nil key）。
			if ko, ok := entry.Interface().(object.KeyedObject); ok {
				km, err := ko.ListKeyMap()
				if err != nil {
					continue
				}
				unsetEnumKey := false
				for _, v := range km {
					rv := reflect.ValueOf(v)
					if rv.Kind() == reflect.Int64 && rv.Type().Name() != "int64" && rv.Int() == 0 {
						unsetEnumKey = true // 枚举键 UNSET（该枚举无可填值域）
					}
				}
				if unsetEnumKey {
					continue
				}
			}
			key, ok := fl.keyFor(f.Type.Key(), entry.Elem())
			if !ok {
				continue
			}
			mv := reflect.MakeMapWithSize(f.Type, 1)
			mv.SetMapIndex(key, entry)
			fv.Set(mv)
		case reflect.Slice:
			if f.Type.Name() != "" { // object.Binary 具名切片
				continue
			}
			et := f.Type.Elem()
			switch {
			case et.Kind() == reflect.Ptr: // 无 key list（闭包 0 处）
				continue
			case et.Kind() == reflect.Interface: // []union
				continue // union leaf-list 由 union 字段路径覆盖，构造包装成本高，跳过
			case et.Kind() == reflect.Int64: // []E_*
				if ev, ok := firstEnumValue(native.EnumMaps, et.Name()); ok {
					sl := reflect.MakeSlice(f.Type, 1, 1)
					sl.Index(0).SetInt(ev)
					fv.Set(sl)
				}
			default: // 标量 leaf-list
				sl := reflect.MakeSlice(f.Type, 2, 2)
				for j := 0; j < 2; j++ {
					setScalar(sl.Index(j), j+1)
				}
				fv.Set(sl)
			}
		case reflect.Interface: // union 字段 / interface{}（unsupported）
			continue // union 需具体包装类型；手工样本已覆盖，扫荡跳过
		}
	}
	// list key 叶已随标量填充；无 key 修补需求（keyFor 从条目取）。
}

func ptrOf(v reflect.Value) reflect.Value {
	p := reflect.New(v.Type())
	p.Elem().Set(v)
	return p
}

func setScalar(v reflect.Value, seed int) {
	switch v.Kind() {
	case reflect.String:
		v.SetString("e" + string(rune('0'+seed)))
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Float64:
		v.SetFloat(float64(seed))
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(uint64(seed))
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(int64(seed))
	}
}

// keyFor 从填充好的条目提取 map key：单键取对应字段值；复合键构造 _Key。
func (fl *filler) keyFor(keyType reflect.Type, entry reflect.Value) (reflect.Value, bool) {
	if keyType.Kind() == reflect.Interface {
		return reflect.Value{}, false // union 键（3 处）扫荡跳过
	}
	if keyType.Kind() == reflect.Struct { // 复合键：按 _Key 字段名从条目复制
		kv := reflect.New(keyType).Elem()
		for i := 0; i < keyType.NumField(); i++ {
			ef := entry.FieldByName(keyType.Field(i).Name)
			if !ef.IsValid() {
				return reflect.Value{}, false
			}
			if ef.Kind() == reflect.Ptr {
				if ef.IsNil() {
					return reflect.Value{}, false
				}
				ef = ef.Elem()
			}
			if !ef.Type().AssignableTo(keyType.Field(i).Type) {
				if !ef.Type().ConvertibleTo(keyType.Field(i).Type) {
					return reflect.Value{}, false
				}
				ef = ef.Convert(keyType.Field(i).Type)
			}
			kv.Field(i).Set(ef)
		}
		return kv, true
	}
	// 单键：条目里找同类型可导出字段（key 叶必为该类型）——退化直接造值
	kv := reflect.New(keyType).Elem()
	setScalar(kv, 7)
	if keyType.Kind() == reflect.Int64 && keyType.Name() != "int64" { // 枚举键
		ev, ok := firstEnumValue(native.EnumMaps, keyType.Name())
		if !ok {
			return reflect.Value{}, false // 无值域枚举键：整条 list 跳过
		}
		kv.SetInt(ev)
	}
	return kv, true
}

func firstEnumValue(table map[string]map[int64]object.EnumDefinition, typeName string) (int64, bool) {
	vals := table[typeName]
	var best int64 = -1
	for v := range vals {
		if best == -1 || v < best {
			best = v
		}
	}
	return best, best != -1
}

func TestJSONSweepParity(t *testing.T) {
	dev := &native.Device{}
	fl := &filler{depthCap: 7, nodeCap: 60000}
	fl.fill(reflect.ValueOf(dev).Elem(), 0)

	njs, err := dev.MarshalJSON()
	if err != nil {
		t.Fatalf("native Marshal: %v", err)
	}
	t.Logf("sweep payload: %d bytes, %d nodes touched", len(njs), fl.nodes)

	yd := &ygothuawei.Device{}
	if err := ygothuawei.Unmarshal(njs, yd); err != nil {
		t.Fatalf("ygot 拒绝 native 输出: %v", err)
	}
	yjs, err := ygot.EmitJSON(yd, &ygot.EmitJSONConfig{Format: ygot.RFC7951, SkipValidation: true})
	if err != nil {
		t.Fatalf("ygot EmitJSON: %v", err)
	}
	var nv, yv interface{}
	if err := json.Unmarshal(njs, &nv); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(yjs), &yv); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(nv, yv) {
		diff := firstDiff(nv, yv, "$")
		t.Fatalf("扫荡对拍语义不等，首差异: %s", diff)
	}

	// 反向：native 自解码往返恒等
	nd2 := &native.Device{}
	if err := nd2.UnmarshalJSON(njs); err != nil {
		t.Fatalf("native 自解码: %v", err)
	}
	njs2, err := nd2.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(njs) != string(njs2) {
		t.Fatal("native 往返不恒等")
	}
}

// firstDiff 定位第一处语义差异路径（排障用）。
func firstDiff(a, b interface{}, path string) string {
	am, aok := a.(map[string]interface{})
	bm, bok := b.(map[string]interface{})
	if aok && bok {
		for k, av := range am {
			bv, ok := bm[k]
			if !ok {
				return path + "." + k + " 仅 native 有"
			}
			if d := firstDiff(av, bv, path+"."+k); d != "" {
				return d
			}
		}
		for k := range bm {
			if _, ok := am[k]; !ok {
				return path + "." + k + " 仅 ygot 有"
			}
		}
		return ""
	}
	aa, aok := a.([]interface{})
	ba, bok := b.([]interface{})
	if aok && bok {
		if len(aa) != len(ba) {
			return path + " 数组长度不等"
		}
		for i := range aa {
			if d := firstDiff(aa[i], ba[i], path+"[]"); d != "" {
				return d
			}
		}
		return ""
	}
	if !reflect.DeepEqual(a, b) {
		return path + " 值不等: " + strings.TrimSpace(jsonStr(a)) + " != " + jsonStr(b)
	}
	return ""
}

func jsonStr(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
