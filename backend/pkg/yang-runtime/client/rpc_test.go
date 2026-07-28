package client

import (
	"strings"
	"testing"
)

// DP-10：input → <rpc> payload 编码（命名空间 + 输入叶 + XML 转义）。
func TestBuildRPCPayload(t *testing.T) {
	got := buildRPCPayload("urn:huawei:ifm", "reset-if-counters-by-name",
		[]RPCInput{{Name: "if-name", Value: "200GE0/1/0"}})

	for _, want := range []string{
		`<reset-if-counters-by-name xmlns="urn:huawei:ifm">`,
		`<if-name>200GE0/1/0</if-name>`,
		`</reset-if-counters-by-name>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("payload 缺 %q\n got: %s", want, got)
		}
	}
}

// 输入值含 XML 特殊字符必须转义（防注入/损坏 payload）。
func TestBuildRPCPayload_Escapes(t *testing.T) {
	got := buildRPCPayload("ns", "op", []RPCInput{{Name: "x", Value: "a<b>&c"}})
	if strings.Contains(got, "a<b>&c") {
		t.Errorf("特殊字符未转义: %s", got)
	}
	if !strings.Contains(got, "&lt;") || !strings.Contains(got, "&amp;") {
		t.Errorf("应含转义实体: %s", got)
	}
}

// 无 input 的 rpc payload。
func TestBuildRPCPayload_NoInput(t *testing.T) {
	got := buildRPCPayload("ns", "ping-op", nil)
	if !strings.Contains(got, `<ping-op xmlns="ns">`) || !strings.Contains(got, `</ping-op>`) {
		t.Errorf("无输入 payload 不符: %s", got)
	}
}

// DP-10：<rpc-reply> 解析——ok / 数据 / rpc-error。
func TestParseRPCReply(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantOK  bool
		wantErr bool
	}{
		{"ok", `<rpc-reply><ok/></rpc-reply>`, true, false},
		{"ok-selfclose", `<ok></ok>`, true, false},
		{"error", `<rpc-reply><rpc-error><error-message>no such if</error-message></rpc-error></rpc-reply>`, false, true},
		{"data", `<rpc-reply><data><counters>0</counters></data></rpc-reply>`, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := parseRPCReply(c.in)
			if res.OK != c.wantOK {
				t.Errorf("OK = %v, want %v", res.OK, c.wantOK)
			}
			if (res.Error != nil) != c.wantErr {
				t.Errorf("Error = %v, wantErr %v", res.Error, c.wantErr)
			}
			if c.wantErr && res.Error != nil && !strings.Contains(res.Error.Error(), "no such if") {
				t.Errorf("rpc-error 应带 error-message: %v", res.Error)
			}
		})
	}
}
