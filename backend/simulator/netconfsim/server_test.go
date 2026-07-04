package netconfsim

import (
	"encoding/xml"
	"strings"
	"testing"
)

// T4.1 — the server hello must advertise base:1.0 plus the :candidate and
// :writable-running capabilities the platform relies on. base:1.1 is out of
// scope (T0.3): the simulator only speaks 1.0 EOM framing.
func TestServerHelloAdvertisesCapabilities(t *testing.T) {
	h := buildHello(1, nil)
	got := make(map[string]bool)
	for _, c := range h.Capabilities.Capabilities {
		got[c.URN] = true
	}
	for _, want := range []string{
		"urn:ietf:params:netconf:base:1.0",
		"urn:ietf:params:netconf:capability:candidate:1.0",
		"urn:ietf:params:netconf:capability:writable-running:1.0",
	} {
		if !got[want] {
			t.Errorf("hello missing capability %q (have %v)", want, got)
		}
	}
	if h.SessionID != 1 {
		t.Errorf("session-id = %d, want 1", h.SessionID)
	}
	// base:1.1 must NOT be advertised (out of scope).
	if got["urn:ietf:params:netconf:base:1.1"] {
		t.Error("hello should not advertise base:1.1")
	}
}

// The hello must still serialize to well-formed XML in the base:1.0 namespace so
// the SSH handshake framing is unchanged.
func TestServerHelloEncodesToXML(t *testing.T) {
	out, err := xml.Marshal(buildHello(1, nil))
	if err != nil {
		t.Fatalf("marshal hello: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, "hello") || !strings.Contains(s, "urn:ietf:params:xml:ns:netconf:base:1.0") {
		t.Fatalf("hello XML missing root/namespace: %s", s)
	}
	if !strings.Contains(s, "capability:candidate:1.0") {
		t.Fatalf("hello XML missing candidate capability: %s", s)
	}
}

// T4.1 — classifyRPC replaces strings.Contains dispatch with encoding/xml
// decoding. It must map each supported RPC to its kind and fall back to
// rpcUnknown for unsupported or malformed messages.
func TestClassifyRPC(t *testing.T) {
	const ns = `xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"`
	cases := []struct {
		name string
		msg  string
		want rpcKind
	}{
		{
			name: "get-config",
			msg:  `<rpc ` + ns + ` message-id="1"><get-config><source><running/></source></get-config></rpc>`,
			want: rpcGetConfig,
		},
		{
			name: "edit-config",
			msg:  `<rpc ` + ns + ` message-id="2"><edit-config><target><candidate/></target><config/></edit-config></rpc>`,
			want: rpcEditConfig,
		},
		{
			name: "commit",
			msg:  `<rpc ` + ns + ` message-id="3"><commit/></rpc>`,
			want: rpcCommit,
		},
		{
			name: "discard-changes",
			msg:  `<rpc ` + ns + ` message-id="4"><discard-changes/></rpc>`,
			want: rpcDiscardChanges,
		},
		{
			name: "unsupported lock falls back",
			msg:  `<rpc ` + ns + ` message-id="5"><lock><target><running/></target></lock></rpc>`,
			want: rpcUnknown,
		},
		{
			name: "malformed xml falls back",
			msg:  `<rpc ` + ns + `><get-config>`,
			want: rpcUnknown,
		},
		{
			name: "not an rpc envelope falls back",
			msg:  `<hello><capabilities/></hello>`,
			want: rpcUnknown,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyRPC(tc.msg); got != tc.want {
				t.Errorf("classifyRPC(%s) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

// Dispatch must not be fooled by the RPC keyword appearing inside element text
// or a comment — the failure mode of the old strings.Contains approach.
func TestClassifyRPCIgnoresKeywordInContent(t *testing.T) {
	const ns = `xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"`
	// A get-config whose filter text mentions "edit-config" must still classify
	// as get-config, not edit-config.
	msg := `<rpc ` + ns + ` message-id="9"><get-config><source><running/></source>` +
		`<filter><desc>edit-config sample</desc></filter></get-config></rpc>`
	if got := classifyRPC(msg); got != rpcGetConfig {
		t.Fatalf("classifyRPC = %v, want rpcGetConfig (keyword-in-content must not mislead)", got)
	}
}
