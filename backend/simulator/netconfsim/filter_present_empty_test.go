package netconfsim

import (
	"strings"
	"testing"
)

// 真机保真回归（T07 配套）：RFC6241 subtree 语义下「filter 元素存在但为空」
// = 什么都不选（须回空 <data/>）；「无 filter 元素」才是全量读。sim 曾把两者
// 都当全量，掩盖了客户端发空过滤器的 bug（真机回空、sim 回全量、测试全绿）。
func TestExtractFilterDistinguishesPresentEmpty(t *testing.T) {
	tests := []struct {
		name        string
		msg         string
		wantInner   string
		wantPresent bool
	}{
		{"无 filter=全量", `<rpc><get-config><source><running/></source></get-config></rpc>`, "", false},
		{"自闭合空 filter=存在但空", `<rpc><get-config><source><running/></source><filter type="subtree"/></get-config></rpc>`, "", true},
		{"XPath select 自闭合（客户端历史形态）", `<rpc><get-config><source><running/></source><filter xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" select="/ifm:ifm/ifm:interfaces"/></get-config></rpc>`, "", true},
		{"有内容 filter", `<rpc><get-config><source><running/></source><filter type="subtree"><ifm xmlns="urn:huawei:yang:huawei-ifm"/></filter></get-config></rpc>`, `<ifm xmlns="urn:huawei:yang:huawei-ifm"/>`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inner, present := extractFilter(tt.msg)
			if inner != tt.wantInner || present != tt.wantPresent {
				t.Errorf("extractFilter() = (%q, %v), want (%q, %v)", inner, present, tt.wantInner, tt.wantPresent)
			}
		})
	}
}

// 端到端（sim 服务面）：空过滤器的 get-config 必须回空 <data/>，不得整树兜底；
// 有效 subtree 过滤器则只回匹配子树（不再无条件全量）。
func TestGetConfigFilterSemantics(t *testing.T) {
	s := &sshServer{store: newTreeDatastore(), scenario: NewScenarioConfig()}
	if err := s.store.SetRunning([]byte(DemoSeedConfig)); err != nil {
		t.Fatalf("seed: %v", err)
	}

	t.Run("空 filter → 空 data（真机行为）", func(t *testing.T) {
		reply := s.handleRequest(`<rpc message-id="7" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"><get-config><source><running/></source><filter xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" select="/ifm:ifm/ifm:interfaces"/></get-config></rpc>`)
		if strings.Contains(reply, "<interface>") || strings.Contains(reply, "<ifm") {
			t.Fatalf("空 filter 不得返回配置内容，got: %.300s", reply)
		}
	})

	t.Run("有效 subtree filter → 匹配子树", func(t *testing.T) {
		reply := s.handleRequest(`<rpc message-id="8" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"><get-config><source><running/></source><filter type="subtree"><ifm xmlns="urn:huawei:yang:huawei-ifm"><interfaces/></ifm></filter></get-config></rpc>`)
		if !strings.Contains(reply, "<interface>") {
			t.Fatalf("有效 filter 应返回接口子树，got: %.300s", reply)
		}
	})

	t.Run("无 filter → 全量（既有行为不回退）", func(t *testing.T) {
		reply := s.handleRequest(`<rpc message-id="9" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"><get-config><source><running/></source></get-config></rpc>`)
		if !strings.Contains(reply, "<interface>") {
			t.Fatalf("无 filter 应全量返回，got: %.300s", reply)
		}
	})
}
