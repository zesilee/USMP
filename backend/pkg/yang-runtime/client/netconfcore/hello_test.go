package netconfcore

import (
	"strings"
	"testing"
)

// 真实世界样本：华为 CE 风格 server hello（带 xmlns 前缀与私有能力）
const huaweiHello = `<?xml version="1.0" encoding="UTF-8"?>
<nc:hello xmlns:nc="urn:ietf:params:xml:ns:netconf:base:1.0">
  <nc:capabilities>
    <nc:capability>urn:ietf:params:netconf:base:1.0</nc:capability>
    <nc:capability>urn:ietf:params:netconf:base:1.1</nc:capability>
    <nc:capability>urn:ietf:params:netconf:capability:candidate:1.0</nc:capability>
    <nc:capability>urn:ietf:params:netconf:capability:confirmed-commit:1.1</nc:capability>
    <nc:capability>http://www.huawei.com/netconf/capability/sync/1.0</nc:capability>
  </nc:capabilities>
  <nc:session-id>27</nc:session-id>
</nc:hello>`

// netconfsim 风格：无前缀、仅 base:1.0
const simHello = `<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <capabilities>
    <capability>urn:ietf:params:netconf:base:1.0</capability>
    <capability>urn:ietf:params:netconf:capability:candidate:1.0</capability>
  </capabilities>
  <session-id>1</session-id>
</hello>`

func TestParseServerHello(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantSID   uint64
		wantCaps  []string
		wantError bool
	}{
		{"华为带前缀样本", huaweiHello, 27,
			[]string{"urn:ietf:params:netconf:base:1.1", "http://www.huawei.com/netconf/capability/sync/1.0"}, false},
		{"sim 无前缀样本", simHello, 1,
			[]string{"urn:ietf:params:netconf:base:1.0"}, false},
		{"缺 session-id 必须报错", // RFC6241: server hello 必含 session-id
			`<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"><capabilities><capability>urn:ietf:params:netconf:base:1.0</capability></capabilities></hello>`,
			0, nil, true},
		{"空 capabilities 必须报错",
			`<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"><capabilities></capabilities><session-id>3</session-id></hello>`,
			0, nil, true},
		{"非法 XML", `<hello><capa`, 0, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := ParseServerHello([]byte(tt.raw))
			if tt.wantError {
				if err == nil {
					t.Fatal("应报错")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseServerHello: %v", err)
			}
			if h.SessionID != tt.wantSID {
				t.Fatalf("SessionID = %d, want %d", h.SessionID, tt.wantSID)
			}
			for _, want := range tt.wantCaps {
				found := false
				for _, c := range h.Capabilities {
					if c == want {
						found = true
					}
				}
				if !found {
					t.Fatalf("缺能力 %q，实际 %v", want, h.Capabilities)
				}
			}
		})
	}
}

func TestNegotiateFraming(t *testing.T) {
	tests := []struct {
		name      string
		caps      []string
		want      FramingVersion
		wantError bool
	}{
		{"双方支持 1.1 → chunked", []string{"urn:ietf:params:netconf:base:1.0", "urn:ietf:params:netconf:base:1.1"}, FramingChunked, false},
		{"仅 1.1 → chunked", []string{"urn:ietf:params:netconf:base:1.1"}, FramingChunked, false},
		{"仅 1.0 → EOM（netconfsim 现状）", []string{"urn:ietf:params:netconf:base:1.0"}, FramingEOM, false},
		{"无 base 能力 → 报错", []string{"urn:ietf:params:netconf:capability:candidate:1.0"}, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NegotiateFraming(tt.caps)
			if tt.wantError {
				if err == nil {
					t.Fatal("应报错")
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("= %v, %v; want %v", got, err, tt.want)
			}
		})
	}
}

func TestBuildClientHello(t *testing.T) {
	out := string(BuildClientHello())
	// 客户端 hello：必须同时声明 1.0/1.1（协商交给服务端能力交集）、不得含 session-id
	for _, want := range []string{
		"urn:ietf:params:netconf:base:1.0",
		"urn:ietf:params:netconf:base:1.1",
		"<hello xmlns=\"urn:ietf:params:xml:ns:netconf:base:1.0\">",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("client hello 缺 %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "session-id") {
		t.Fatal("client hello 不得携带 session-id")
	}
}
