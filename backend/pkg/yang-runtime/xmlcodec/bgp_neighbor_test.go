package xmlcodec

import (
	"reflect"
	"strings"
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
)

// BN-01/02/03：公网 BGP 基础邻居（instance[_public_]/bgp/base-process/peers/peer）
// 经既有 ni 描述符 + XC-06 per-node namespace 端到端编解码。peer 是 huawei-bgp augment
// 深层子树（instance/bgp/base-process/peers/peer/afs/af），验证：深层嵌套 + list-under-
// list + 枚举 key（af-type）+ 深处 per-node namespace。正确性靠 encode namespace 真值断言。

const peerConfigTrueLeaves = 41 // 26 直属 + timer/graceful-restart/bfd-parameter 15（schema 实测锁定）

// 2a 纳入的 peer 基础子容器（其余 fake-as-parameter/egress-engineer-parameter/
// local-graceful-restart/afs 与 config-false 状态容器不属基础邻居完备面）。
var peerScopeContainers = map[string]bool{
	"timer": true, "graceful-restart": true, "bfd-parameter": true,
}

type peerAlias = huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance_Bgp_BaseProcess_Peers_Peer

// peerSchema 定位 peer 的 schema entry。
func peerSchema(t *testing.T) *yang.Entry {
	t.Helper()
	root := huawei.SchemaTree["HuaweiNetworkInstance_NetworkInstance"]
	e := root.Dir["instances"].Dir["instance"].Dir["bgp"].Dir["base-process"].Dir["peers"].Dir["peer"]
	if e == nil {
		t.Fatal("peer schema 未解析")
	}
	return e
}

// populatePeerConfigTrue 给 peer 直属 config-true 标量 + 2a 基础子容器
// (timer/graceful-restart/bfd-parameter) 的 config-true 标量赋唯一值；afs/其他子容器/
// config-false 跳过。返回赋值 leaf 数。
func populatePeerConfigTrue(t *testing.T, sv reflect.Value, e *yang.Entry, n *int) {
	t.Helper()
	st := sv.Type()
	for i := 0; i < st.NumField(); i++ {
		f := st.Field(i)
		tag := pathTag(f)
		if tag == "" {
			continue
		}
		var child *yang.Entry
		if e != nil {
			child = e.Dir[tag]
		}
		cfg := child == nil || child.Config != yang.TSFalse
		fv := sv.Field(i)
		switch {
		case fv.Kind() == reflect.Ptr && fv.Type().Elem().Kind() == reflect.Struct:
			if !cfg || !peerScopeContainers[tag] {
				continue // 仅递归 2a 基础子容器
			}
			fv.Set(reflect.New(fv.Type().Elem()))
			populatePeerConfigTrue(t, fv.Elem(), child, n)
		case fv.Kind() == reflect.Ptr:
			if !cfg {
				continue
			}
			setScalarLeaf(fv, *n)
			*n++
		case fv.Kind() == reflect.Int64 && fv.Type().Implements(goEnumType):
			if !cfg {
				continue
			}
			fv.SetInt(1)
			*n++
		}
	}
}

// wrapPeer 把一个 peer 包进完整 /ni:network-instance 结构（instance[_public_]）。
func wrapPeer(addr string, p *peerAlias) *huawei.HuaweiNetworkInstance_NetworkInstance {
	return &huawei.HuaweiNetworkInstance_NetworkInstance{
		Instances: &huawei.HuaweiNetworkInstance_NetworkInstance_Instances{
			Instance: map[string]*huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance{
				"_public_": {
					Name: ygot.String("_public_"),
					Bgp: &huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance_Bgp{
						BaseProcess: &huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance_Bgp_BaseProcess{
							Peers: &huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance_Bgp_BaseProcess_Peers{
								Peer: map[string]*peerAlias{addr: p},
							},
						},
					},
				},
			},
		},
	}
}

// BN-01 完备性 + 往返 + namespace 真值：peer 全 config-true 标量（+基础子容器）赋值→
// 经 ni Spec 编码（XC-06 namespace）→解码→DeepEqual，且计数锁定。
func TestBN01_PeerAllConfigTrue_RoundtripAndNamespace(t *testing.T) {
	pe := peerSchema(t)
	p := &peerAlias{}
	n := 0
	populatePeerConfigTrue(t, reflect.ValueOf(p).Elem(), pe, &n)
	if p.Address == nil {
		t.Fatal("address（key）未赋值")
	}
	// 计数锁定：26 直属 config-true 标量 + timer/graceful-restart/bfd-parameter 的标量。
	// 首次运行后按实测锁定常量，防「全属性可配」漏字段。
	const wantPeerConfigTrue = peerConfigTrueLeaves
	if n != wantPeerConfigTrue {
		t.Fatalf("peer config-true 标量覆盖 = %d，期望 %d（模型变更？须复审）", n, wantPeerConfigTrue)
	}

	ni := wrapPeer(*p.Address, p)
	xml, err := Encode(niSpecWithNS(), ni)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	// per-node namespace 真值：<bgp> 带 huawei-bgp namespace，peer 继承、ni 原生不另发
	if !strings.Contains(xml, `<bgp xmlns="urn:huawei:yang:huawei-bgp">`) {
		t.Errorf("bgp 子树缺 huawei-bgp namespace\n%s", xml)
	}
	if strings.Contains(xml, `<peer xmlns=`) || strings.Contains(xml, `<address xmlns=`) || strings.Contains(xml, `<remote-as xmlns=`) {
		t.Errorf("peer 子节点不应另发 xmlns（继承 <bgp>）\n%s", xml)
	}

	got := &huawei.HuaweiNetworkInstance_NetworkInstance{}
	if err := Decode(niSpecWithNS(), []byte(xml), got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !reflect.DeepEqual(ni, got) {
		t.Fatalf("peer 往返不等价（字段级丢失）\nXML: %s", xml)
	}
}

// XC-07 YANG empty 类型（ygot YANGEmpty，非指针 bool）：presence-only——true 发
// 自闭合 <tag/>、false 不发；解码元素存在即 true。peer/bfd-parameter/compatible 是
// 首个走此路径的驱动字段（vlan/ifm/bgp/ni 无 empty 类型）。
func TestXC07_EmptyType_PresenceOnly(t *testing.T) {
	withEmpty := &peerAlias{
		Address:  ygot.String("10.0.0.9"),
		RemoteAs: ygot.String("100"),
		BfdParameter: &huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance_Bgp_BaseProcess_Peers_Peer_BfdParameter{
			Compatible: huawei.YANGEmpty(true),
		},
	}
	xml, err := Encode(niSpecWithNS(), wrapPeer("10.0.0.9", withEmpty))
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !strings.Contains(xml, "<compatible/>") {
		t.Errorf("empty 类型 true 应发自闭合 <compatible/>\n%s", xml)
	}
	got := &huawei.HuaweiNetworkInstance_NetworkInstance{}
	if err := Decode(niSpecWithNS(), []byte(xml), got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	gp := got.Instances.Instance["_public_"].Bgp.BaseProcess.Peers.Peer["10.0.0.9"]
	if gp.BfdParameter == nil || gp.BfdParameter.Compatible != huawei.YANGEmpty(true) {
		t.Fatalf("empty 类型未解码为 present/true: %#v", gp.BfdParameter)
	}

	// false（缺省）：不发
	noEmpty := &peerAlias{
		Address:      ygot.String("10.0.0.10"),
		RemoteAs:     ygot.String("100"),
		BfdParameter: &huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance_Bgp_BaseProcess_Peers_Peer_BfdParameter{Compatible: huawei.YANGEmpty(false)},
	}
	xml2, err := Encode(niSpecWithNS(), wrapPeer("10.0.0.10", noEmpty))
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if strings.Contains(xml2, "<compatible") {
		t.Errorf("empty 类型 false 不应发\n%s", xml2)
	}
}

// BN-02 af-type 枚举 key：afs/af（key=af-type，枚举）list-under-list 编解码往返。
func TestBN02_AfTypeEnumKey_Roundtrip(t *testing.T) {
	afType := huawei.E_HuaweiBgp_AfType(1) // 非 UNSET 的某地址族
	p := &peerAlias{
		Address:  ygot.String("10.0.0.1"),
		RemoteAs: ygot.String("100"),
		Afs: &huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance_Bgp_BaseProcess_Peers_Peer_Afs{
			Af: map[huawei.E_HuaweiBgp_AfType]*huawei.HuaweiNetworkInstance_NetworkInstance_Instances_Instance_Bgp_BaseProcess_Peers_Peer_Afs_Af{
				afType: {Type: afType},
			},
		},
	}
	ni := wrapPeer("10.0.0.1", p)
	xml, err := Encode(niSpecWithNS(), ni)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !strings.Contains(xml, "<afs>") || !strings.Contains(xml, "<af>") {
		t.Errorf("afs/af 未编码\n%s", xml)
	}
	got := &huawei.HuaweiNetworkInstance_NetworkInstance{}
	if err := Decode(niSpecWithNS(), []byte(xml), got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	gp := got.Instances.Instance["_public_"].Bgp.BaseProcess.Peers.Peer["10.0.0.1"]
	if gp.Afs == nil || gp.Afs.Af[afType] == nil {
		t.Fatalf("af-type 枚举 key 往返丢失: %#v", gp.Afs)
	}
}
