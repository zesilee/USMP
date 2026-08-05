package netconfcore

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeServer 在 net.Pipe 对端扮演 NETCONF 服务端：EOM 交换 hello 后按协商
// 封帧收发。handler 收到完整 <rpc> 帧、返回应答帧内容。
type fakeServer struct {
	conn    net.Conn
	caps    []string
	handler func(req []byte) []byte
	done    chan struct{}
	errs    chan error
}

var msgIDRe = regexp.MustCompile(`message-id="(\d+)"`)

// echoOK 提取请求 message-id 回 <ok/>。
func echoOK(req []byte) []byte {
	m := msgIDRe.FindSubmatch(req)
	return []byte(fmt.Sprintf(
		`<rpc-reply xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="%s"><ok/></rpc-reply>`, m[1]))
}

func startFakeServer(t *testing.T, caps []string, handler func([]byte) []byte) (net.Conn, *fakeServer) {
	t.Helper()
	clientSide, serverSide := net.Pipe()
	fs := &fakeServer{conn: serverSide, caps: caps, handler: handler,
		done: make(chan struct{}), errs: make(chan error, 8)}
	go fs.run()
	t.Cleanup(func() {
		_ = serverSide.Close()
		_ = clientSide.Close()
		<-fs.done
		close(fs.errs)
		for err := range fs.errs {
			t.Errorf("fakeServer: %v", err)
		}
	})
	return clientSide, fs
}

func (fs *fakeServer) run() {
	defer close(fs.done)
	// hello 恒走 EOM
	eomW := NewEOMWriter(fs.conn)
	eomR := NewEOMReader(fs.conn)
	var capXML strings.Builder
	for _, c := range fs.caps {
		capXML.WriteString("<capability>" + c + "</capability>")
	}
	hello := `<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"><capabilities>` +
		capXML.String() + `</capabilities><session-id>99</session-id></hello>`
	if err := eomW.WriteFrame([]byte(hello)); err != nil {
		return
	}
	if _, err := eomR.ReadFrame(); err != nil { // client hello
		return
	}
	// 协商后封帧
	var r FrameReader = eomR
	var w FrameWriter = eomW
	chunked := false
	for _, c := range fs.caps {
		if c == capBase11 {
			chunked = true
		}
	}
	if chunked {
		r, w = NewChunkedReader(fs.conn), NewChunkedWriter(fs.conn)
	}
	for {
		req, err := r.ReadFrame()
		if err != nil {
			return // 客户端关连接
		}
		if strings.Contains(string(req), "<close-session/>") {
			_ = w.WriteFrame(echoOK(req))
			return
		}
		if fs.handler == nil {
			_ = w.WriteFrame(echoOK(req))
			continue
		}
		if err := w.WriteFrame(fs.handler(req)); err != nil {
			return
		}
	}
}

func newTestSession(t *testing.T, caps []string, handler func([]byte) []byte) *Session {
	t.Helper()
	conn, _ := startFakeServer(t, caps, handler)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s, err := NewSession(ctx, conn)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	return s
}

var (
	caps10 = []string{capBase10}
	caps11 = []string{capBase10, capBase11}
)

// ── 会话建立 ────────────────────────────────────────────────────

func TestSessionOpenNegotiate(t *testing.T) {
	tests := []struct {
		name string
		caps []string
		want FramingVersion
	}{
		{"sim 风格仅 1.0 → EOM", caps10, FramingEOM},
		{"华为风格 1.1 → chunked", caps11, FramingChunked},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestSession(t, tt.caps, nil)
			if s.SessionID() != 99 {
				t.Fatalf("SessionID = %d, want 99", s.SessionID())
			}
			if s.Framing() != tt.want {
				t.Fatalf("Framing = %v, want %v", s.Framing(), tt.want)
			}
			if len(s.Capabilities()) != len(tt.caps) {
				t.Fatalf("Capabilities = %v", s.Capabilities())
			}
		})
	}
}

func TestSessionOpenBadHello(t *testing.T) {
	clientSide, serverSide := net.Pipe()
	defer clientSide.Close()
	go func() {
		w := NewEOMWriter(serverSide)
		_ = w.WriteFrame([]byte(`<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"><capabilities><capability>x</capability></capabilities><session-id>1</session-id></hello>`))
		r := NewEOMReader(serverSide)
		_, _ = r.ReadFrame()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := NewSession(ctx, clientSide); err == nil {
		t.Fatal("无 base 能力应建会话失败")
	}
}

// ── RPC 引擎 ────────────────────────────────────────────────────

func TestSessionDoRoundTrip(t *testing.T) {
	for _, caps := range [][]string{caps10, caps11} {
		s := newTestSession(t, caps, func(req []byte) []byte {
			if !strings.Contains(string(req), "<get-x/>") {
				return echoOK(req)
			}
			m := msgIDRe.FindSubmatch(req)
			return []byte(fmt.Sprintf(
				`<rpc-reply message-id="%s"><data><y>1</y></data></rpc-reply>`, m[1]))
		})
		reply, err := s.Do(context.Background(), []byte("<get-x/>"))
		if err != nil {
			t.Fatalf("Do: %v", err)
		}
		if !strings.Contains(string(reply), "<y>1</y>") {
			t.Fatalf("reply = %s", reply)
		}
	}
}

func TestSessionMessageIDMismatchKillsSession(t *testing.T) {
	s := newTestSession(t, caps10, func(req []byte) []byte {
		return []byte(`<rpc-reply message-id="777"><ok/></rpc-reply>`)
	})
	if _, err := s.Do(context.Background(), []byte("<x/>")); err == nil {
		t.Fatal("message-id 错配应报错")
	}
	// 错配后会话不可信，必须拒绝后续请求（快速失败，交给上层重连）
	if _, err := s.Do(context.Background(), []byte("<x/>")); !errors.Is(err, ErrSessionDead) {
		t.Fatalf("死会话应 ErrSessionDead, got %v", err)
	}
}

func TestSessionRPCError(t *testing.T) {
	s := newTestSession(t, caps10, func(req []byte) []byte {
		if !strings.Contains(string(req), "<x/>") {
			return echoOK(req) // 第二条正常请求回 ok，验证会话未被业务错误判死
		}
		m := msgIDRe.FindSubmatch(req)
		return []byte(fmt.Sprintf(`<rpc-reply message-id="%s">
  <rpc-error>
    <error-type>application</error-type>
    <error-tag>data-missing</error-tag>
    <error-severity>error</error-severity>
    <error-message>vlan 999 not found</error-message>
  </rpc-error>
</rpc-reply>`, m[1]))
	})
	_, err := s.Do(context.Background(), []byte("<x/>"))
	if err == nil {
		t.Fatal("rpc-error(severity=error) 应返回错误")
	}
	var re *RPCReplyError
	if !errors.As(err, &re) {
		t.Fatalf("应为 *RPCReplyError, got %T: %v", err, err)
	}
	if re.Errors[0].Tag != "data-missing" || !strings.Contains(re.Errors[0].Message, "vlan 999") {
		t.Fatalf("错误细节 = %+v", re.Errors[0])
	}
	// rpc-error 是业务错误不是传输错误：会话仍可用
	if _, err := s.Do(context.Background(), []byte("<ok-please/>")); err != nil {
		t.Fatalf("业务错误后会话应仍可用: %v", err)
	}
}

func TestSessionWarningOnlyNotError(t *testing.T) {
	s := newTestSession(t, caps10, func(req []byte) []byte {
		m := msgIDRe.FindSubmatch(req)
		return []byte(fmt.Sprintf(`<rpc-reply message-id="%s"><rpc-error><error-severity>warning</error-severity><error-message>deprecated</error-message></rpc-error><ok/></rpc-reply>`, m[1]))
	})
	if _, err := s.Do(context.Background(), []byte("<x/>")); err != nil {
		t.Fatalf("仅 warning 不应视为失败: %v", err)
	}
}

func TestSessionConcurrentDoSerialized(t *testing.T) {
	// 并发 Do 必须串行化且各自拿到自己的应答（根治 scrapligo messageID 竞态）
	var mu sync.Mutex
	seen := map[string]bool{}
	s := newTestSession(t, caps11, func(req []byte) []byte {
		m := msgIDRe.FindSubmatch(req)
		mu.Lock()
		if seen[string(m[1])] {
			mu.Unlock()
			return []byte(`<rpc-reply message-id="0"><rpc-error><error-severity>error</error-severity><error-message>duplicate id</error-message></rpc-error></rpc-reply>`)
		}
		seen[string(m[1])] = true
		mu.Unlock()
		return echoOK(req)
	})
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Do(context.Background(), []byte("<x/>")); err != nil {
				t.Errorf("并发 Do: %v", err)
			}
		}()
	}
	wg.Wait()
}

func TestSessionContextTimeoutKillsSession(t *testing.T) {
	block := make(chan struct{})
	s := newTestSession(t, caps10, nil)
	// 单独造一个不回话的服务端：handler 挂住 → 客户端 ctx 超时
	clientSide, serverSide := net.Pipe()
	t.Cleanup(func() { clientSide.Close(); serverSide.Close(); close(block) })
	go func() {
		w := NewEOMWriter(serverSide)
		_ = w.WriteFrame([]byte(`<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"><capabilities><capability>` + capBase10 + `</capability></capabilities><session-id>7</session-id></hello>`))
		r := NewEOMReader(serverSide)
		_, _ = r.ReadFrame() // client hello
		_, _ = r.ReadFrame() // rpc（不回话）
		<-block
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	s2, err := NewSession(ctx, clientSide)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	shortCtx, cancel2 := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel2()
	start := time.Now()
	_, err = s2.Do(shortCtx, []byte("<never-replied/>"))
	if err == nil || time.Since(start) > 2*time.Second {
		t.Fatalf("应在 ctx 超时内失败, err=%v 耗时=%v", err, time.Since(start))
	}
	// 超时后传输层状态未知 → 会话必须判死，快速拒绝而非挂起（根治死连接复用）
	if _, err := s2.Do(context.Background(), []byte("<x/>")); !errors.Is(err, ErrSessionDead) {
		t.Fatalf("超时后应 ErrSessionDead, got %v", err)
	}
	_ = s // keep first session alive till cleanup
}

func TestSessionAbortNonBlocking(t *testing.T) {
	s := newTestSession(t, caps10, nil)
	done := make(chan struct{})
	go func() { s.Abort(); s.Abort(); close(done) }() // 幂等且不取锁
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Abort 不得阻塞")
	}
	if _, err := s.Do(context.Background(), []byte("<x/>")); err == nil {
		t.Fatal("Abort 后 Do 应失败")
	}
	if _, err := s.Do(context.Background(), []byte("<x/>")); !errors.Is(err, ErrSessionDead) {
		t.Fatalf("Abort 后会话应判死, got %v", err)
	}
}

func TestSessionCloseIdempotent(t *testing.T) {
	s := newTestSession(t, caps10, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := s.Close(ctx); err != nil {
		t.Fatalf("二次 Close 应幂等: %v", err)
	}
	if _, err := s.Do(context.Background(), []byte("<x/>")); !errors.Is(err, ErrSessionDead) {
		t.Fatalf("关闭后 Do 应 ErrSessionDead, got %v", err)
	}
}

// CN-04 归因地基：华为 313 形态 unknown-element 的 bad-element 须被结构化
// 提取（error-info>bad-element），供上层按请求路径归因。
func TestSessionRPCErrorBadElement(t *testing.T) {
	s := newTestSession(t, caps10, func(req []byte) []byte {
		m := msgIDRe.FindSubmatch(req)
		return []byte(fmt.Sprintf(`<rpc-reply message-id="%s" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <rpc-error>
    <error-type>application</error-type>
    <error-tag>unknown-element</error-tag>
    <error-severity>error</error-severity>
    <error-message xml:lang="en">Unexpected element: cards.</error-message>
    <error-info xmlns:nc-ext="urn:huawei:yang:huawei-ietf-netconf-ext">
      <bad-element>cards</bad-element>
      <nc-ext:error-info-code>313</nc-ext:error-info-code>
    </error-info>
  </rpc-error>
</rpc-reply>`, m[1]))
	})
	_, err := s.Do(context.Background(), []byte("<x/>"))
	var re *RPCReplyError
	if !errors.As(err, &re) {
		t.Fatalf("应为 *RPCReplyError, got %T: %v", err, err)
	}
	if re.Errors[0].Tag != "unknown-element" || re.Errors[0].BadElement != "cards" {
		t.Fatalf("bad-element 未提取: %+v", re.Errors[0])
	}
}
