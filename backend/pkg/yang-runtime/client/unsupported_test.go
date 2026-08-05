package client

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client/netconfcore"
)

// CN-04（tasks 2.1）：节点级不支持集——unknown-element 归因判定（保守：
// bad-element 须与请求路径段名匹配才算）+ 集合增/查/清 + 并发 race。

func ueErr(tag, severity, badElement string) error {
	return &netconfcore.RPCReplyError{Errors: []netconfcore.RPCError{{
		Type: "application", Tag: tag, Severity: severity, BadElement: badElement,
		Message: fmt.Sprintf("Unexpected element: %s.", badElement),
	}}}
}

func TestUnknownElementForPath(t *testing.T) {
	cases := []struct {
		name string
		path string
		err  error
		want bool
	}{
		{"bad-element 命中末段", "devm:devm/devm:cards", ueErr("unknown-element", "error", "cards"), true},
		{"bad-element 命中带谓词段", "ifm:ifm/ifm:interfaces/ifm:interface[name='GE0/0/1']", ueErr("unknown-element", "error", "interface"), true},
		{"bad-element 命中根段", "devm:devm/devm:cards", ueErr("unknown-element", "error", "devm"), true},
		{"tag=bad-element 同判", "devm:devm/devm:cards", ueErr("bad-element", "error", "cards"), true},
		{"bad-element 与路径不匹配→不归因", "devm:devm/devm:cards", ueErr("unknown-element", "error", "ports"), false},
		{"其他 error-tag 不归因", "devm:devm/devm:cards", ueErr("operation-failed", "error", "cards"), false},
		{"severity=warning 不归因", "devm:devm/devm:cards", ueErr("unknown-element", "warning", "cards"), false},
		{"bad-element 缺失不归因", "devm:devm/devm:cards", ueErr("unknown-element", "error", ""), false},
		{"非 rpc-error 不归因", "devm:devm/devm:cards", errors.New("dial tcp: timeout"), false},
		{"nil 错误不归因", "devm:devm/devm:cards", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := UnknownElementForPath(tc.path, tc.err); got != tc.want {
				t.Fatalf("UnknownElementForPath(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestNodeSupportSetOps(t *testing.T) {
	c := &NETCONFClient{}
	if c.IsUnsupportedPath("devm:devm/devm:cards") {
		t.Fatal("空集不得命中")
	}
	c.MarkUnsupportedPath("devm:devm/devm:cards")
	if !c.IsUnsupportedPath("devm:devm/devm:cards") {
		t.Fatal("标记后应命中")
	}
	// 首尾斜杠归一化：同一路径不同写法视为同一条（API *path 带首斜杠、
	// 模块根前缀不带，两端形态必须互通）
	if !c.IsUnsupportedPath("devm:devm/devm:cards/") {
		t.Fatal("尾斜杠应归一化命中")
	}
	if !c.IsUnsupportedPath("/devm:devm/devm:cards") {
		t.Fatal("首斜杠应归一化命中")
	}
	if c.IsUnsupportedPath("devm:devm/devm:ports") {
		t.Fatal("未标记路径不得命中")
	}
	c.ClearUnsupportedPath("devm:devm/devm:cards")
	if c.IsUnsupportedPath("devm:devm/devm:cards") {
		t.Fatal("清除后不得命中")
	}
}

func TestUnsupportedPathsUnder(t *testing.T) {
	c := &NETCONFClient{}
	c.MarkUnsupportedPath("devm:devm/devm:cards")
	c.MarkUnsupportedPath("devm:devm/devm:schedule-reboot")
	c.MarkUnsupportedPath("ifm:ifm/ifm:interfaces")
	got := c.UnsupportedPathsUnder("devm:devm")
	if len(got) != 2 {
		t.Fatalf("devm 前缀应命中 2 条, got %v", got)
	}
	for _, p := range got {
		if p != "devm:devm/devm:cards" && p != "devm:devm/devm:schedule-reboot" {
			t.Fatalf("意外条目 %q", p)
		}
	}
	// 前缀须按段边界匹配：devm:devm 不得命中 devm:devm2/...
	c.MarkUnsupportedPath("devm:devm2/devm:x")
	if got := c.UnsupportedPathsUnder("devm:devm"); len(got) != 2 {
		t.Fatalf("段边界匹配失败, got %v", got)
	}
}

func TestNodeSupportConcurrency(t *testing.T) {
	c := &NETCONFClient{}
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(3)
		go func(n int) {
			defer wg.Done()
			c.MarkUnsupportedPath(fmt.Sprintf("m:m/m:c%d", n))
		}(i)
		go func(n int) {
			defer wg.Done()
			c.IsUnsupportedPath(fmt.Sprintf("m:m/m:c%d", n))
		}(i)
		go func() {
			defer wg.Done()
			c.UnsupportedPathsUnder("m:m")
		}()
	}
	wg.Wait()
}

// 重连清空（CN-04）：connect() 重建 backend 时不支持集必须清零重学。
func TestNodeSupportClearedOnReconnect(t *testing.T) {
	c := &NETCONFClient{}
	c.MarkUnsupportedPath("devm:devm/devm:cards")
	c.resetNodeSupport()
	if c.IsUnsupportedPath("devm:devm/devm:cards") {
		t.Fatal("重连后不支持集应清空")
	}
}
