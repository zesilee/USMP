package netconfcore

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

// 真机排障配套（T05）：USMP_NETCONF_WIRE_DEBUG=1 时，发出的 <rpc> 原文与收到的
// <rpc-reply> 原文（含长度）必须进日志——真机「空回复/超时」类问题靠肉眼比对
// 线上报文定位（2026-08-05 CE 读通道排障即此场景）。缺省关闭：不产生任何 wire 行。
func TestWireDebugLogsSendRecv(t *testing.T) {
	t.Setenv("USMP_NETCONF_WIRE_DEBUG", "1")
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(prev)

	wireLog("send", []byte("<rpc><get/></rpc>"))
	wireLog("recv", []byte("<rpc-reply><data/></rpc-reply>"))

	out := buf.String()
	if !strings.Contains(out, "netconf-wire send 17B") {
		t.Errorf("send 行缺失或缺长度: %s", out)
	}
	if !strings.Contains(out, "<rpc><get/></rpc>") {
		t.Errorf("send 原文缺失: %s", out)
	}
	if !strings.Contains(out, "netconf-wire recv") || !strings.Contains(out, "<data/>") {
		t.Errorf("recv 行缺失: %s", out)
	}
}

func TestWireDebugOffByDefault(t *testing.T) {
	t.Setenv("USMP_NETCONF_WIRE_DEBUG", "")
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(prev)

	wireLog("send", []byte("<rpc/>"))
	if buf.Len() != 0 {
		t.Errorf("开关关闭时不得输出 wire 日志: %s", buf.String())
	}
}

// 超长报文截断（保留头尾，日志不被兆级回读撑爆）。
func TestWireDebugTruncatesLargePayload(t *testing.T) {
	t.Setenv("USMP_NETCONF_WIRE_DEBUG", "1")
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(prev)

	big := "<rpc-reply>" + strings.Repeat("x", 10000) + "TAIL</rpc-reply>"
	wireLog("recv", []byte(big))
	out := buf.String()
	if len(out) > 5000 {
		t.Errorf("日志行未截断（%d 字节）", len(out))
	}
	if !strings.Contains(out, "TAIL</rpc-reply>") {
		t.Errorf("截断须保留尾部（框架结束符可见性）: %.200s…", out)
	}
	if !strings.Contains(out, "…") {
		t.Errorf("截断标记缺失")
	}
}
