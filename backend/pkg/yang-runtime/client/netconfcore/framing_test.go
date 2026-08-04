package netconfcore

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
)

// slowReader 每次 Read 只吐 n 字节，模拟 TCP 拆包（帧头/定界符被切开的场景）。
type slowReader struct {
	data []byte
	n    int
	pos  int
}

func (r *slowReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	end := r.pos + r.n
	if end > len(r.data) {
		end = len(r.data)
	}
	c := copy(p, r.data[r.pos:end])
	r.pos += c
	return c, nil
}

// ── EOM 封帧（NETCONF 1.0，]]>]]> 定界）──────────────────────────

func TestEOMRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		msgs []string
	}{
		{"单帧", []string{"<hello/>"}},
		{"背靠背多帧", []string{"<rpc>1</rpc>", "<rpc>2</rpc>", "<rpc>3</rpc>"}},
		{"含定界符前缀的载荷", []string{"<a>]]&gt;]] not-delim ]]</a>"}},
		// 回归：载荷以 ] / ]] 结尾时与定界符首字符粘连（"…]"+"]]>]]>"="…]]]>]]>"），
		// 朴素前缀匹配会丢字节或错切帧
		{"载荷以单 ] 结尾", []string{"x]"}},
		{"载荷以 ]] 结尾", []string{"x]]"}},
		{"载荷含 ]]] 串", []string{"a]]]b"}},
		{"载荷仅为 ]", []string{"]"}},
		{"中文与多字节", []string{"<desc>华为交换机-测试</desc>"}},
		{"大帧 1MB", []string{strings.Repeat("x", 1<<20)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := NewEOMWriter(&buf)
			for _, m := range tt.msgs {
				if err := w.WriteFrame([]byte(m)); err != nil {
					t.Fatalf("WriteFrame: %v", err)
				}
			}
			r := NewEOMReader(&buf)
			for i, want := range tt.msgs {
				got, err := r.ReadFrame()
				if err != nil {
					t.Fatalf("ReadFrame[%d]: %v", i, err)
				}
				if string(got) != want {
					t.Fatalf("帧[%d] = %q, want %q", i, got, want)
				}
			}
			if _, err := r.ReadFrame(); !errors.Is(err, io.EOF) {
				t.Fatalf("流尽后应 EOF, got %v", err)
			}
		})
	}
}

func TestEOMDelimiterSplitAcrossReads(t *testing.T) {
	// 定界符 ]]>]]> 被逐字节拆开到达，仍须正确切帧
	raw := "<rpc-reply>ok</rpc-reply>]]>]]><next/>]]>]]>"
	for n := 1; n <= 7; n++ {
		r := NewEOMReader(&slowReader{data: []byte(raw), n: n})
		got1, err := r.ReadFrame()
		if err != nil || string(got1) != "<rpc-reply>ok</rpc-reply>" {
			t.Fatalf("n=%d 帧1 = %q err=%v", n, got1, err)
		}
		got2, err := r.ReadFrame()
		if err != nil || string(got2) != "<next/>" {
			t.Fatalf("n=%d 帧2 = %q err=%v", n, got2, err)
		}
	}
}

func TestEOMTruncatedStream(t *testing.T) {
	// 无定界符即断流：必须报错（ErrUnexpectedEOF），不能把半帧当整帧
	r := NewEOMReader(strings.NewReader("<rpc>half"))
	if _, err := r.ReadFrame(); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("半帧应报 ErrUnexpectedEOF, got %v", err)
	}
}

// withSmallFrameLimit 把帧上限临时压到 1KB，毫秒级验证切断路径（真实上限 64MB
// 灌数据在 -race 下要几十秒）。测试串行执行，改写包变量安全。
func withSmallFrameLimit(t *testing.T) {
	t.Helper()
	old := maxFrameSize
	maxFrameSize = 1 << 10
	t.Cleanup(func() { maxFrameSize = old })
}

func TestEOMFrameSizeLimit(t *testing.T) {
	// 超过上限的帧必须报错切断，防恶意/异常设备把内存打爆
	withSmallFrameLimit(t)
	huge := strings.Repeat("y", maxFrameSize+1)
	var buf bytes.Buffer
	buf.WriteString(huge)
	buf.WriteString(eomDelimiter)
	if _, err := NewEOMReader(&buf).ReadFrame(); err == nil {
		t.Fatal("超限帧应报错")
	}
}

// ── Chunked 封帧（NETCONF 1.1，RFC6242 §4.2）─────────────────────

func TestChunkedRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		msgs []string
	}{
		{"单帧", []string{"<hello/>"}},
		{"背靠背多帧", []string{"<rpc>1</rpc>", "<rpc>2</rpc>"}},
		{"载荷含伪 chunk 头", []string{"<a>\n#5\nfake\n##\n</a>"}},
		{"中文", []string{"<d>华为</d>"}},
		{"大帧 1MB", []string{strings.Repeat("z", 1<<20)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := NewChunkedWriter(&buf)
			for _, m := range tt.msgs {
				if err := w.WriteFrame([]byte(m)); err != nil {
					t.Fatalf("WriteFrame: %v", err)
				}
			}
			r := NewChunkedReader(&buf)
			for i, want := range tt.msgs {
				got, err := r.ReadFrame()
				if err != nil {
					t.Fatalf("ReadFrame[%d]: %v", i, err)
				}
				if string(got) != want {
					t.Fatalf("帧[%d] = %q, want %q", i, got, want)
				}
			}
		})
	}
}

func TestChunkedMultiChunkReassembly(t *testing.T) {
	// 设备把一条消息拆多个 chunk 发（大回包典型行为），须重组为一帧
	raw := "\n#4\n<rpc\n#8\n-reply/>\n##\n"
	got, err := NewChunkedReader(strings.NewReader(raw)).ReadFrame()
	if err != nil || string(got) != "<rpc-reply/>" {
		t.Fatalf("重组 = %q err=%v", got, err)
	}
}

func TestChunkedSplitAcrossReads(t *testing.T) {
	// chunk 头与数据被 TCP 逐字节拆开
	raw := "\n#12\n<rpc-reply/>\n##\n\n#5\n<ok/>\n##\n"
	for n := 1; n <= 5; n++ {
		r := NewChunkedReader(&slowReader{data: []byte(raw), n: n})
		got1, err := r.ReadFrame()
		if err != nil || string(got1) != "<rpc-reply/>" {
			t.Fatalf("n=%d 帧1 = %q err=%v", n, got1, err)
		}
		got2, err := r.ReadFrame()
		if err != nil || string(got2) != "<ok/>" {
			t.Fatalf("n=%d 帧2 = %q err=%v", n, got2, err)
		}
	}
}

func TestChunkedMalformed(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"缺 LF-hash 前导", "#5\nhello\n##\n"},
		{"chunk-size 非数字", "\n#ab\nhello\n##\n"},
		{"chunk-size 前导零", "\n#05\nhello\n##\n"},
		{"chunk-size 为零", "\n#0\n\n##\n"},
		{"chunk-size 超 RFC 上限", "\n#99999999999\nx\n##\n"},
		{"数据不足即断流", "\n#10\nshort"},
		{"缺结束标记", "\n#5\nhello"},
		{"结束标记残缺", "\n#5\nhello\n#"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewChunkedReader(strings.NewReader(tt.raw)).ReadFrame(); err == nil {
				t.Fatalf("畸形输入应报错: %q", tt.raw)
			}
		})
	}
}

func TestChunkedTotalSizeLimit(t *testing.T) {
	// 多 chunk 累计超上限也必须切断（单 chunk 合法但总量打爆内存的攻击面）
	withSmallFrameLimit(t)
	var buf bytes.Buffer
	chunk := strings.Repeat("a", 256)
	for i := 0; i < maxFrameSize/256+2; i++ {
		buf.WriteString("\n#256\n")
		buf.WriteString(chunk)
	}
	buf.WriteString("\n##\n")
	if _, err := NewChunkedReader(&buf).ReadFrame(); err == nil {
		t.Fatal("累计超限应报错")
	}
}

func TestChunkedSingleChunkOverLimit(t *testing.T) {
	// 单个 chunk-size 直接超上限：读数据前就应切断
	withSmallFrameLimit(t)
	raw := "\n#2048\n" + strings.Repeat("b", 2048) + "\n##\n"
	if _, err := NewChunkedReader(strings.NewReader(raw)).ReadFrame(); err == nil {
		t.Fatal("单 chunk 超限应报错")
	}
}

// ── 并发独立性（R09：实例间零共享状态，-race 验证）────────────────

func TestFramingConcurrentInstances(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var buf bytes.Buffer
			w := NewChunkedWriter(&buf)
			_ = w.WriteFrame([]byte("<rpc/>"))
			got, err := NewChunkedReader(&buf).ReadFrame()
			if err != nil || string(got) != "<rpc/>" {
				t.Errorf("并发实例 = %q err=%v", got, err)
			}
			var buf2 bytes.Buffer
			w2 := NewEOMWriter(&buf2)
			_ = w2.WriteFrame([]byte("<rpc/>"))
			got2, err := NewEOMReader(&buf2).ReadFrame()
			if err != nil || string(got2) != "<rpc/>" {
				t.Errorf("并发实例 EOM = %q err=%v", got2, err)
			}
		}()
	}
	wg.Wait()
}
