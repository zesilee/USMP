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

// chunk 级观测（真机排障二段）：完整帧迟迟收不齐时（设备慢 vs 解析卡死无法
// 区分），开关开启下 chunked 读端须逐 chunk 打点（尺寸+累计），读帧失败时打出
// 已累计字节数——没有这层，卡在半帧的会话在日志里完全隐形。
func TestWireDebugLogsChunkProgress(t *testing.T) {
	t.Setenv("USMP_NETCONF_WIRE_DEBUG", "1")
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(prev)

	// 两个 chunk（5B + 3B）+ 正常结束。
	r := NewChunkedReader(strings.NewReader("\n#5\nhello\n#3\nxyz\n##\n"))
	frame, err := r.ReadFrame()
	if err != nil || string(frame) != "helloxyz" {
		t.Fatalf("ReadFrame: %q, %v", frame, err)
	}
	out := buf.String()
	if !strings.Contains(out, "chunk 5B") || !strings.Contains(out, "chunk 3B") {
		t.Errorf("chunk 进度日志缺失: %s", out)
	}

	// 断流半帧：错误路径须打出已累计字节。
	buf.Reset()
	r2 := NewChunkedReader(strings.NewReader("\n#5\nhel"))
	if _, err := r2.ReadFrame(); err == nil {
		t.Fatal("断流须报错")
	}
	if !strings.Contains(buf.String(), "partial") {
		t.Errorf("断流路径缺已累计字节日志: %s", buf.String())
	}
}

func TestWireDebugChunkSilentWhenOff(t *testing.T) {
	t.Setenv("USMP_NETCONF_WIRE_DEBUG", "")
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(prev)
	r := NewChunkedReader(strings.NewReader("\n#5\nhello\n##\n"))
	if _, err := r.ReadFrame(); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("开关关闭时 chunk 层不得输出: %s", buf.String())
	}
}
