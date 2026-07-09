package driver

import (
	"sync"
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/xmlcodec"
)

// fake ygot shapes（driver 包保持零业务依赖：不 import 生成物）。
type fakeVlans struct {
	Vlan map[uint16]*fakeVlan `path:"vlan"`
}

func (*fakeVlans) IsYANGGoStruct() {}

type fakeVlan struct {
	Id *uint16 `path:"id"`
}

func (*fakeVlan) IsYANGGoStruct() {}

type otherStruct struct{}

func (*otherStruct) IsYANGGoStruct() {}

func xmlDescriptor() Descriptor {
	return Descriptor{
		Vendor: "huawei", Module: "vlan",
		NewStruct: func() ygot.GoStruct { return &fakeVlans{} },
		XML: &xmlcodec.Spec{
			Namespace: "urn:fake",
			Schema: func() *yang.Entry {
				return &yang.Entry{Name: "vlans", Dir: map[string]*yang.Entry{
					"vlan": {Name: "vlan", Key: "id", ListAttr: &yang.ListAttr{}},
				}}
			},
		},
	}
}

// DR-01: 按 GoStruct 类型（容器型 + 内层 list map 型）查得携带 XML 编解码
// 数据的描述符；未命中 ok=false（R08 降级）。
func TestRegistry_XMLEncoderForValue(t *testing.T) {
	r := NewRegistry()
	r.Register(xmlDescriptor())
	// 无 XML 数据的描述符不得参与匹配。
	r.Register(Descriptor{Vendor: "huawei", Module: "system",
		NewStruct: func() ygot.GoStruct { return &otherStruct{} }})

	t.Run("container type match", func(t *testing.T) {
		d, ok := r.XMLEncoderForValue(&fakeVlans{})
		if !ok || d.Module != "vlan" || d.XML == nil {
			t.Fatalf("container type should match, got %+v ok=%v", d, ok)
		}
	})
	t.Run("inner list map type match", func(t *testing.T) {
		id := uint16(10)
		m := map[uint16]*fakeVlan{10: {Id: &id}}
		d, ok := r.XMLEncoderForValue(m)
		if !ok || d.Module != "vlan" {
			t.Fatalf("inner map type should match（diff 引擎会发内层 map，IFM 漏发 bug 根因）, got ok=%v", ok)
		}
	})
	t.Run("miss returns false", func(t *testing.T) {
		if _, ok := r.XMLEncoderForValue(&otherStruct{}); ok {
			t.Fatal("descriptor without XML spec must not match")
		}
		if _, ok := r.XMLEncoderForValue("just a string"); ok {
			t.Fatal("non-GoStruct must not match")
		}
		if _, ok := r.XMLEncoderForValue(nil); ok {
			t.Fatal("nil must not match")
		}
	})
}

// WrapXMLValue：内层 map 形态包装回容器；容器形态原样返回；异型报错。
func TestDescriptor_WrapXMLValue(t *testing.T) {
	d := xmlDescriptor()
	t.Run("container passthrough", func(t *testing.T) {
		v := &fakeVlans{}
		got, err := d.WrapXMLValue(v)
		if err != nil || got != ygot.GoStruct(v) {
			t.Fatalf("want passthrough, got %v err %v", got, err)
		}
	})
	t.Run("inner map wrapped", func(t *testing.T) {
		id := uint16(7)
		m := map[uint16]*fakeVlan{7: {Id: &id}}
		got, err := d.WrapXMLValue(m)
		if err != nil {
			t.Fatal(err)
		}
		c, ok := got.(*fakeVlans)
		if !ok || c.Vlan[7] == nil || *c.Vlan[7].Id != 7 {
			t.Fatalf("map not wrapped into container: %+v", got)
		}
	})
	t.Run("mismatched value", func(t *testing.T) {
		if _, err := d.WrapXMLValue(42); err == nil {
			t.Fatal("want error for unrelated value")
		}
	})
}

// R09: 并发 Register 与 XMLEncoderForValue 无数据竞态（-race 验证）。
func TestRegistry_XMLConcurrent(t *testing.T) {
	r := NewRegistry()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			r.Register(xmlDescriptor())
		}()
		go func() {
			defer wg.Done()
			r.XMLEncoderForValue(&fakeVlans{})
		}()
	}
	wg.Wait()
}
