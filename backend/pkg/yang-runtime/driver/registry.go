// Package driver provides the compile-time device-driver descriptor registry
// (DR-01, SND 声明式化第一步)：每个 (vendor, module) 一条描述符，收敛此前散落在
// manager 路由与 config 编解码里的路径字符串硬编码。描述符经各接线包的 init()
// 注册（纯 Go 编译期，无运行时插件加载）；本包零业务依赖（仅 ygot 类型），
// 供 manager / api / 将来 client 消费而不成环。
//
// 本期描述符刻意最小（谓词 + 控制器名 token + 编解码闭包）；①声明式数据驱动
// 终态在此 struct 上扩展（路径/模板描述、能力元数据），不另起注册机制。
package driver

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/xmlcodec"
)

// Descriptor describes one device driver module: how config paths map to its
// reconcile controller and its codec closures. Match predicates are kept
// per-concern to reproduce the pre-registry dispatch semantics exactly
// (行为等价是本期硬约束；谓词的声明式化属后续 change)。
type Descriptor struct {
	// Vendor is the driver vendor label (lowercase, e.g. "huawei").
	Vendor string
	// Module is the YANG module identity (e.g. "vlan").
	Module string

	// MatchRoute reports whether a config path belongs to this module for
	// reconcile routing (DR-02). ControllerToken is the substring identifying
	// the module's controller name at registration.
	MatchRoute      func(path string) bool
	ControllerToken string

	// MatchDecode + DecodeXML: NETCONF XML readback → ygot GoStruct (DR-03 读).
	MatchDecode func(path string) bool
	DecodeXML   func(raw []byte) (ygot.GoStruct, error)

	// MatchEncode + NewStruct + Unmarshal: RFC7951 JSON → ygot GoStruct (DR-03 写).
	MatchEncode func(path string) bool
	NewStruct   func() ygot.GoStruct
	Unmarshal   func([]byte, ygot.GoStruct, ...ytypes.UnmarshalOpt) error

	// EncodeAnchor 是 NewStruct 容器的规范配置路径（DR-05，如 "/vlan:vlan/vlan:vlans"）。
	// config-api 写路径据此把「以请求 path 为根的 RFC7951 子树」机械包裹成锚点相对
	// JSON 后根级解码；SND 谓词声明式化的第一块数据（后续 MatchEncode 可由其派生）。
	EncodeAnchor string

	// XML carries the generic NETCONF XML codec data for this module
	// (yang-xml-codec XC-01/02/03)：namespace + SchemaTree 入口。nil 表示该
	// 模块不走通用 XML 引擎（如 system 无 XML 通道），调用方保持既有降级。
	XML *xmlcodec.Spec
}

// WrapXMLValue normalizes a change value to the descriptor's container
// GoStruct: containers pass through, inner list-map values (diff 引擎产出的
// map[K]*Entry 形态) are wrapped into a fresh container. Values of any other
// type are an explicit error (R08).
func (d Descriptor) WrapXMLValue(v interface{}) (ygot.GoStruct, error) {
	if d.NewStruct == nil {
		return nil, fmt.Errorf("driver: descriptor %s/%s has no NewStruct", d.Vendor, d.Module)
	}
	container := d.NewStruct()
	if reflect.TypeOf(v) == reflect.TypeOf(container) {
		return v.(ygot.GoStruct), nil
	}
	if err := xmlcodec.WrapListMap(container, v); err != nil {
		return nil, fmt.Errorf("driver %s/%s: %w", d.Vendor, d.Module, err)
	}
	return container, nil
}

// matchesXMLValue reports whether v is this descriptor's container type or
// its inner list-map type.
func (d Descriptor) matchesXMLValue(v interface{}) bool {
	if d.XML == nil || d.NewStruct == nil || v == nil {
		return false
	}
	container := d.NewStruct()
	tv := reflect.TypeOf(v)
	if tv == reflect.TypeOf(container) {
		return true
	}
	mt, err := xmlcodec.ListMapType(container)
	return err == nil && tv == mt
}

// Registry holds descriptors in registration order (first match wins — 对拍
// 既有 if-链语义)。注册发生在 init() 阶段、运行期只读；加锁防运行期注册与读
// 并发竞态（R09）。
type Registry struct {
	mu          sync.RWMutex
	descriptors []Descriptor
}

// NewRegistry creates an empty registry (unit tests / future multi-tenant use).
func NewRegistry() *Registry { return &Registry{} }

// Register appends a descriptor. Later registrations never shadow earlier ones.
func (r *Registry) Register(d Descriptor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.descriptors = append(r.descriptors, d)
}

// Route returns the first descriptor whose MatchRoute covers path (DR-02).
func (r *Registry) Route(path string) (Descriptor, bool) {
	return r.lookup(func(d Descriptor) bool { return d.MatchRoute != nil && d.MatchRoute(path) })
}

// DecoderFor returns the first descriptor able to decode XML readback at path.
func (r *Registry) DecoderFor(path string) (Descriptor, bool) {
	return r.lookup(func(d Descriptor) bool {
		return d.DecodeXML != nil && d.MatchDecode != nil && d.MatchDecode(path)
	})
}

// EncoderFor returns the first descriptor able to encode RFC7951 JSON at path.
func (r *Registry) EncoderFor(path string) (Descriptor, bool) {
	return r.lookup(func(d Descriptor) bool {
		return d.NewStruct != nil && d.Unmarshal != nil && d.MatchEncode != nil && d.MatchEncode(path)
	})
}

// XMLEncoderForValue returns the first descriptor whose container GoStruct
// type (or inner list-map type) matches v and that carries XML codec data.
// 按类型而非路径匹配：Change 的 Path 在 diff 引擎产出内层 map 时不可靠
// （当年 IFM 漏发 bug 根因），类型匹配是唯一稳定信号。
func (r *Registry) XMLEncoderForValue(v interface{}) (Descriptor, bool) {
	return r.lookup(func(d Descriptor) bool { return d.matchesXMLValue(v) })
}

// VendorSupported reports whether any registered descriptor belongs to the
// vendor (case-insensitive). It is the single source of truth for "厂商是否有
// 已注册驱动" (DR-04)，供 devices-api BR-04 厂商门禁消费。
func (r *Registry) VendorSupported(vendor string) bool {
	_, ok := r.lookup(func(d Descriptor) bool { return strings.EqualFold(d.Vendor, vendor) })
	return ok
}

func (r *Registry) lookup(pred func(Descriptor) bool) (Descriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, d := range r.descriptors {
		if pred(d) {
			return d, true
		}
	}
	return Descriptor{}, false
}

// defaultRegistry backs the package-level facade used by production wiring.
var defaultRegistry = NewRegistry()

// Register adds a descriptor to the default registry (init()-time wiring).
func Register(d Descriptor) { defaultRegistry.Register(d) }

// Route looks up the default registry for reconcile routing.
func Route(path string) (Descriptor, bool) { return defaultRegistry.Route(path) }

// DecoderFor looks up the default registry for XML readback decoding.
func DecoderFor(path string) (Descriptor, bool) { return defaultRegistry.DecoderFor(path) }

// EncoderFor looks up the default registry for RFC7951 encoding.
func EncoderFor(path string) (Descriptor, bool) { return defaultRegistry.EncoderFor(path) }

// XMLEncoderForValue looks up the default registry by change value type.
func XMLEncoderForValue(v interface{}) (Descriptor, bool) {
	return defaultRegistry.XMLEncoderForValue(v)
}

// VendorSupported reports whether the default registry has any driver
// descriptor for the vendor (case-insensitive, DR-04).
func VendorSupported(vendor string) bool { return defaultRegistry.VendorSupported(vendor) }
