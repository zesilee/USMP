package netconfsim

import (
	"errors"
	"strings"
	"testing"
)

// NS-08 server 层：<get> RPC 分发、subtree filter、get-config 状态隔离、故障注入。

const rpcNS = `xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"`

func newGetTestServer(t *testing.T) *sshServer {
	t.Helper()
	s := &sshServer{store: newTreeDatastore(), scenario: NewScenarioConfig()}
	if err := s.store.SetRunning([]byte(stateIfmRunning)); err != nil {
		t.Fatalf("SetRunning: %v", err)
	}
	if err := s.store.SetState([]byte(stateIfmOverlay)); err != nil {
		t.Fatalf("SetState: %v", err)
	}
	return s
}

func TestClassifyRPCGet(t *testing.T) {
	cases := []struct {
		name string
		msg  string
		want rpcKind
	}{
		{"bare get", `<rpc message-id="1" ` + rpcNS + `><get/></rpc>`, rpcGet},
		{"get with filter", `<rpc message-id="1" ` + rpcNS + `><get><filter type="subtree"><ifm/></filter></get></rpc>`, rpcGet},
		{"get-config still get-config", `<rpc message-id="1" ` + rpcNS + `><get-config><source><running/></source></get-config></rpc>`, rpcGetConfig},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyRPC(tc.msg); got != tc.want {
				t.Fatalf("classifyRPC(%s) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

// <get> 无 filter：返回配置+状态合并全树。
func TestHandleGetReturnsMergedTree(t *testing.T) {
	s := newGetTestServer(t)
	reply := s.handleRequest(`<rpc message-id="7" ` + rpcNS + `><get/></rpc>`)
	if !strings.Contains(reply, `message-id="7"`) || !strings.Contains(reply, "<data>") {
		t.Fatalf("malformed get reply: %.300s", reply)
	}
	if !strings.Contains(reply, "<oper-status>1</oper-status>") {
		t.Fatalf("get reply missing merged state leaf: %.500s", reply)
	}
	if !strings.Contains(reply, "<mtu>9216</mtu>") {
		t.Fatalf("get reply missing config leaf: %.500s", reply)
	}
}

// <get> 套 subtree filter：只返回命中条目，状态子树随之合并。
func TestHandleGetAppliesSubtreeFilter(t *testing.T) {
	s := newGetTestServer(t)
	reply := s.handleRequest(`<rpc message-id="8" ` + rpcNS + `><get><filter type="subtree">` +
		`<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm"><interfaces><interface><name>200GE0/1/0</name></interface></interfaces></ifm>` +
		`</filter></get></rpc>`)
	if !strings.Contains(reply, "<dynamic>") {
		t.Fatalf("filtered get missing state subtree: %.500s", reply)
	}
	if strings.Contains(reply, "200GE0/1/1") {
		t.Fatalf("filter leaked non-matching entry: %.500s", reply)
	}
}

// get-config 恒不含状态数据（NS-08 隔离场景，server 层）。
func TestHandleGetConfigExcludesStateData(t *testing.T) {
	s := newGetTestServer(t)
	reply := s.handleRequest(`<rpc message-id="9" ` + rpcNS + `><get-config><source><running/></source></get-config></rpc>`)
	if strings.Contains(reply, "dynamic") {
		t.Fatalf("get-config reply leaked state data: %.500s", reply)
	}
	if !strings.Contains(reply, "<mtu>9216</mtu>") {
		t.Fatalf("get-config reply missing config: %.500s", reply)
	}
}

// ErrorOnRPC["get"] 注入：返回 rpc-error。
func TestHandleGetErrorInjection(t *testing.T) {
	s := newGetTestServer(t)
	s.scenario.ErrorOnRPC["get"] = errors.New("device meltdown")
	reply := s.handleRequest(`<rpc message-id="10" ` + rpcNS + `><get/></rpc>`)
	if !strings.Contains(reply, "rpc-error") || !strings.Contains(reply, "device meltdown") {
		t.Fatalf("expected injected rpc-error, got: %.300s", reply)
	}
}
