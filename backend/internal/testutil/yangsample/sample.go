// Package yangsample 按 schema 引导为 ygot 容器构造最小真值样例（测试专用）：
// 全量接入的参数化矩阵（编解码往返 / sim 端到端）共用，避免每模块手写 desired。
// 覆盖面刻意最小：首个 config-true 标量叶（string/bool/uint）或单简单键 list 行；
// enum/union/identityref/多键跳过（深层类型面按需求波次补，full-yang-onboarding D5）。
package yangsample

import (
	"reflect"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
)

// Populate 在容器根下寻找首个可赋值节点并赋样例值，返回是否成功。
func Populate(v ygot.GoStruct, e *yang.Entry) bool {
	return populate(reflect.ValueOf(v), e, 0)
}

func populate(sv reflect.Value, e *yang.Entry, depth int) bool {
	if depth > 4 || e == nil || sv.Kind() != reflect.Ptr || sv.IsNil() {
		return false
	}
	el := sv.Elem()
	st := el.Type()
	for i := 0; i < st.NumField(); i++ {
		child := childEntry(e, pathOf(st.Field(i)))
		if child == nil || !child.IsLeaf() || child.Config == yang.TSFalse {
			continue
		}
		f := el.Field(i)
		if f.Kind() != reflect.Ptr || !f.CanSet() {
			continue
		}
		switch f.Type().Elem().Kind() {
		case reflect.String:
			p := reflect.New(f.Type().Elem())
			p.Elem().SetString("usmp")
			f.Set(p)
			return true
		case reflect.Bool:
			p := reflect.New(f.Type().Elem())
			p.Elem().SetBool(true)
			f.Set(p)
			return true
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			p := reflect.New(f.Type().Elem())
			p.Elem().SetUint(1)
			f.Set(p)
			return true
		}
	}
	for i := 0; i < st.NumField(); i++ {
		child := childEntry(e, pathOf(st.Field(i)))
		if child == nil || !child.IsList() || child.Config == yang.TSFalse {
			continue
		}
		f := el.Field(i)
		if f.Kind() != reflect.Map || !f.CanSet() {
			continue
		}
		if populateListRow(f, child) {
			return true
		}
	}
	for i := 0; i < st.NumField(); i++ {
		child := childEntry(e, pathOf(st.Field(i)))
		if child == nil || child.IsLeaf() || child.IsList() || child.Config == yang.TSFalse {
			continue
		}
		f := el.Field(i)
		if f.Kind() != reflect.Ptr || f.Type().Elem().Kind() != reflect.Struct || !f.CanSet() {
			continue
		}
		f.Set(reflect.New(f.Type().Elem()))
		if populate(f, child, depth+1) {
			return true
		}
		f.Set(reflect.Zero(f.Type()))
	}
	return false
}

func populateListRow(f reflect.Value, list *yang.Entry) bool {
	keys := strings.Fields(list.Key)
	if len(keys) != 1 {
		return false
	}
	keyName := keys[0]
	keyType := f.Type().Key()
	elemType := f.Type().Elem()
	if elemType.Kind() != reflect.Ptr || elemType.Elem().Kind() != reflect.Struct {
		return false
	}
	var kv reflect.Value
	switch keyType.Kind() {
	case reflect.String:
		kv = reflect.ValueOf("usmp").Convert(keyType)
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		kv = reflect.ValueOf(uint64(1)).Convert(keyType)
	default:
		return false
	}
	elem := reflect.New(elemType.Elem())
	set := false
	est := elemType.Elem()
	for i := 0; i < est.NumField(); i++ {
		if pathOf(est.Field(i)) != keyName {
			continue
		}
		ef := elem.Elem().Field(i)
		if ef.Kind() != reflect.Ptr || ef.Type().Elem() != keyType {
			return false
		}
		p := reflect.New(keyType)
		p.Elem().Set(kv)
		ef.Set(p)
		set = true
		break
	}
	if !set {
		return false
	}
	m := reflect.MakeMap(f.Type())
	m.SetMapIndex(kv, elem)
	f.Set(m)
	return true
}

func pathOf(f reflect.StructField) string { return f.Tag.Get("path") }

// childEntry 按名查子节点，穿透 choice/case（ygot 结构体拍平 choice/case，
// schema 树保留层级）。
func childEntry(e *yang.Entry, name string) *yang.Entry {
	if name == "" || e == nil || e.Dir == nil {
		return nil
	}
	if c, ok := e.Dir[name]; ok {
		return c
	}
	for _, c := range e.Dir {
		if c.IsChoice() || c.IsCase() {
			if got := childEntry(c, name); got != nil {
				return got
			}
		}
	}
	return nil
}
