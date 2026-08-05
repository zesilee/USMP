package netconfsim

import (
	"strings"
	"testing"
)

// CN-04 地基（tasks 1.1）：按路径注入 unknown-element——复刻华为真机对
// running 配置 schema 不存在节点的 313 形态拒绝（error-tag=unknown-element +
// bad-element），让「设备软件版本无此节点」从此有 B2 防线。

func newUnknownElemServer(t *testing.T, paths ...string) *sshServer {
	t.Helper()
	sc := NewScenarioConfig()
	sc.UnknownElementPaths = paths
	s := &sshServer{store: newTreeDatastore(), scenario: sc}
	if err := s.store.SetRunning([]byte(stateIfmRunning)); err != nil {
		t.Fatalf("SetRunning: %v", err)
	}
	return s
}

func assertUnknownElementReply(t *testing.T, reply, badElement string) {
	t.Helper()
	for _, want := range []string{
		"<error-tag>unknown-element</error-tag>",
		"<bad-element>" + badElement + "</bad-element>",
		"<error-severity>error</error-severity>",
	} {
		if !strings.Contains(reply, want) {
			t.Fatalf("reply missing %q:\n%.500s", want, reply)
		}
	}
	if strings.Contains(reply, "<data>") {
		t.Fatalf("error reply must not carry <data>:\n%.500s", reply)
	}
}

// get-config filter 命中注入路径 → 华为 313 形态 rpc-error。
func TestUnknownElementInjectionGetConfig(t *testing.T) {
	s := newUnknownElemServer(t, "ifm/interfaces")
	reply := s.handleRequest(`<rpc message-id="27" ` + rpcNS + `><get-config><source><running/></source>` +
		`<filter type="subtree"><ifm xmlns="urn:huawei:yang:huawei-ifm"><interfaces/></ifm></filter></get-config></rpc>`)
	assertUnknownElementReply(t, reply, "interfaces")
	if !strings.Contains(reply, `message-id="27"`) {
		t.Fatalf("reply must echo message-id: %.300s", reply)
	}
}

// <get>（状态通道）同样受注入约束。
func TestUnknownElementInjectionGet(t *testing.T) {
	s := newUnknownElemServer(t, "ifm/interfaces")
	reply := s.handleRequest(`<rpc message-id="28" ` + rpcNS + `><get>` +
		`<filter type="subtree"><ifm xmlns="urn:huawei:yang:huawei-ifm"><interfaces/></ifm></filter></get></rpc>`)
	assertUnknownElementReply(t, reply, "interfaces")
}

// edit-config 配置体命中注入路径 → 同款拒绝（写路径 B2 防线）。
func TestUnknownElementInjectionEditConfig(t *testing.T) {
	s := newUnknownElemServer(t, "ifm/interfaces")
	reply := s.handleRequest(`<rpc message-id="29" ` + rpcNS + `><edit-config><target><running/></target>` +
		`<config><ifm xmlns="urn:huawei:yang:huawei-ifm"><interfaces><interface><name>X</name></interface></interfaces></ifm></config></edit-config></rpc>`)
	assertUnknownElementReply(t, reply, "interfaces")
}

// 未命中路径不受影响：其他子树照常返回数据（注入必须精确，不能误伤）。
func TestUnknownElementInjectionMissUnaffected(t *testing.T) {
	s := newUnknownElemServer(t, "devm/cards")
	reply := s.handleRequest(`<rpc message-id="30" ` + rpcNS + `><get-config><source><running/></source>` +
		`<filter type="subtree"><ifm xmlns="urn:huawei:yang:huawei-ifm"><interfaces/></ifm></filter></get-config></rpc>`)
	if strings.Contains(reply, "unknown-element") {
		t.Fatalf("non-matching path must not be rejected: %.400s", reply)
	}
	if !strings.Contains(reply, "200GE0/1/0") {
		t.Fatalf("normal data expected for non-matching path: %.400s", reply)
	}
}

// 无注入 → 全部正常（缺省行为零变化）。
func TestUnknownElementInjectionDefaultOff(t *testing.T) {
	s := newUnknownElemServer(t)
	reply := s.handleRequest(`<rpc message-id="31" ` + rpcNS + `><get-config><source><running/></source>` +
		`<filter type="subtree"><ifm xmlns="urn:huawei:yang:huawei-ifm"><interfaces/></ifm></filter></get-config></rpc>`)
	if strings.Contains(reply, "rpc-error") {
		t.Fatalf("default must be no injection: %.400s", reply)
	}
}

// 仅首段匹配（父容器存在、子节点才是注入点）不误拒：请求 <ifm/> 整树时，
// 注入 ifm/interfaces 意味着设备"认识 ifm 但没有 interfaces"——真机对仅含
// 父容器的 filter 不报错（返回它有的部分），sim 对齐该语义。
func TestUnknownElementInjectionParentOnlyFilterPasses(t *testing.T) {
	s := newUnknownElemServer(t, "ifm/interfaces")
	reply := s.handleRequest(`<rpc message-id="32" ` + rpcNS + `><get-config><source><running/></source>` +
		`<filter type="subtree"><ifm xmlns="urn:huawei:yang:huawei-ifm"/></filter></get-config></rpc>`)
	if strings.Contains(reply, "unknown-element") {
		t.Fatalf("parent-only filter must not be rejected: %.400s", reply)
	}
}
