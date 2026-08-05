// session.go — NETCONF 会话与 RPC 引擎（自研替代 Wave 2）。
//
// 设计要点（对照 scrapligo 已知缺陷，勿复刻）：
//   - message-id 递增与网络读写全程持锁：并发 Do 串行化，根治编号竞态与帧交错；
//   - 任何传输级异常/超时/编号错配 → 会话判死（ErrSessionDead）快速拒绝，
//     绝不复用状态未知的连接，重连交上层 ClientPool；
//   - 超时/关闭一律先关底层连接解锁在途读写（看门狗模式），无死锁、无泄漏协程；
//   - rpc-error(severity=error) 是业务错误：返回结构化 *RPCReplyError，会话仍可用。
package netconfcore

import (
	"bufio"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
)

// ErrSessionDead 会话已失效（传输异常/超时/协议错乱后置位），须重连。
var ErrSessionDead = errors.New("netconfcore: 会话已失效（需重连）")

// RPCError 一条 <rpc-error> 的结构化字段。
type RPCError struct {
	Type     string `xml:"error-type"`
	Tag      string `xml:"error-tag"`
	Severity string `xml:"error-severity"`
	Message  string `xml:"error-message"`
	// BadElement error-info>bad-element（华为 unknown-element/313 形态携带的
	// 被拒节点名），供上层按请求路径归因（CN-04）。缺失为空串。
	BadElement string `xml:"error-info>bad-element"`
}

// RPCReplyError 应答中 severity=error 的错误集合（业务错误，非传输错误）。
type RPCReplyError struct {
	Errors []RPCError
}

func (e *RPCReplyError) Error() string {
	if len(e.Errors) == 0 {
		return "netconfcore: rpc-error"
	}
	first := e.Errors[0]
	return fmt.Sprintf("netconfcore: rpc-error [%s/%s]: %s（共 %d 条）",
		first.Tag, first.Severity, first.Message, len(e.Errors))
}

// Session 一条已完成 hello 协商的 NETCONF 会话。并发安全：Do/Close 可被多
// goroutine 调用，内部串行化（NETCONF 会话语义本就是有序请求-应答流）。
type Session struct {
	mu        sync.Mutex
	conn      io.ReadWriteCloser
	reader    FrameReader
	writer    FrameWriter
	framing   FramingVersion
	sessionID uint64
	caps      []string
	msgID     uint64
	dead      error // 非 nil = 会话判死原因
	closed    bool
}

// NewSession 在任意字节流上建立会话：交换 hello（恒 EOM）、协商封帧。
// ctx 约束整个握手；超时/失败会关闭 conn。SSH 连接由 DialSSH 提供，测试可
// 直接用 net.Pipe 注入假服务端。
func NewSession(ctx context.Context, conn io.ReadWriteCloser) (*Session, error) {
	br := bufio.NewReader(conn)
	eomR := NewEOMReader(br)
	eomW := NewEOMWriter(conn)

	// 双向 hello 并发进行：RFC6241 双方连上即发，各自读对端。分开两个
	// goroutine 才能兼容无缓冲传输（net.Pipe）与任意服务端收发顺序。
	writeErr := make(chan error, 1)
	type readRes struct {
		raw []byte
		err error
	}
	readCh := make(chan readRes, 1)
	go func() { writeErr <- eomW.WriteFrame(BuildClientHello()) }()
	go func() {
		raw, err := eomR.ReadFrame()
		readCh <- readRes{raw, err}
	}()

	var serverRaw []byte
	for i := 0; i < 2; i++ {
		select {
		case <-ctx.Done():
			_ = conn.Close()
			return nil, fmt.Errorf("netconfcore: hello 握手超时: %w", ctx.Err())
		case err := <-writeErr:
			if err != nil {
				_ = conn.Close()
				return nil, fmt.Errorf("netconfcore: 发送 client hello: %w", err)
			}
		case r := <-readCh:
			if r.err != nil {
				_ = conn.Close()
				return nil, fmt.Errorf("netconfcore: 读取 server hello: %w", r.err)
			}
			serverRaw = r.raw
		}
	}

	hello, err := ParseServerHello(serverRaw)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	framing, err := NegotiateFraming(hello.Capabilities)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	wireLogf("session %d established, framing=%v, server-caps=%d", hello.SessionID, framing, len(hello.Capabilities))
	s := &Session{
		conn:      conn,
		framing:   framing,
		sessionID: hello.SessionID,
		caps:      hello.Capabilities,
	}
	if framing == FramingChunked {
		s.reader = NewChunkedReader(br) // 共享 bufio，hello 预读字节不丢
		s.writer = NewChunkedWriter(conn)
	} else {
		s.reader = eomR
		s.writer = eomW
	}
	return s, nil
}

// SessionID 服务端分配的会话号。
func (s *Session) SessionID() uint64 { return s.sessionID }

// Framing 协商出的封帧格式。
func (s *Session) Framing() FramingVersion { return s.framing }

// Capabilities 服务端能力清单（副本）。
func (s *Session) Capabilities() []string {
	out := make([]string, len(s.caps))
	copy(out, s.caps)
	return out
}

// rpcReply 应答骨架：message-id 校验 + rpc-error 提取（原文另行透传）。
type rpcReply struct {
	MessageID string     `xml:"message-id,attr"`
	Errors    []RPCError `xml:"rpc-error"`
}

// Do 发送一条 RPC（body 为 <rpc> 内层操作元素）并等待应答，返回完整
// <rpc-reply> 原文。ctx 到期即判死会话并切断连接（快速失败语义）。
func (s *Session) Do(ctx context.Context, body []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dead != nil {
		return nil, s.dead
	}
	if s.closed {
		return nil, fmt.Errorf("%w: 已关闭", ErrSessionDead)
	}
	s.msgID++
	id := s.msgID
	envelope := fmt.Sprintf(
		`<?xml version="1.0" encoding="UTF-8"?><rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="%d">%s</rpc>`,
		id, body)

	wireLog("send", []byte(envelope))
	raw, err := s.exchangeLocked(ctx, []byte(envelope))
	if err != nil {
		wireLogf("recv-error: %v", err)
		return nil, err
	}
	wireLog("recv", raw)

	var reply rpcReply
	if err := xml.Unmarshal(raw, &reply); err != nil {
		s.kill(fmt.Errorf("应答非法 XML: %w", err))
		return nil, s.dead
	}
	if reply.MessageID != strconv.FormatUint(id, 10) {
		// 编号错配 = 应答流错位，之后每条应答都可能是别人的：必须判死
		s.kill(fmt.Errorf("message-id 错配（发 %d 收 %q）", id, reply.MessageID))
		return nil, s.dead
	}
	var hard []RPCError
	for _, e := range reply.Errors {
		if e.Severity != "warning" {
			hard = append(hard, e)
		}
	}
	if len(hard) > 0 {
		return raw, &RPCReplyError{Errors: hard}
	}
	return raw, nil
}

// exchangeLocked 持锁状态下写请求读应答，ctx 看门狗超时即切断连接。
// 后台协程因连接关闭而解锁退出（chan 带缓冲），无泄漏。
func (s *Session) exchangeLocked(ctx context.Context, frame []byte) ([]byte, error) {
	type res struct {
		raw []byte
		err error
	}
	ch := make(chan res, 1)
	go func() {
		if err := s.writer.WriteFrame(frame); err != nil {
			ch <- res{nil, fmt.Errorf("写请求: %w", err)}
			return
		}
		raw, err := s.reader.ReadFrame()
		if err != nil {
			err = fmt.Errorf("读应答: %w", err)
		}
		ch <- res{raw, err}
	}()
	select {
	case <-ctx.Done():
		s.kill(fmt.Errorf("RPC 超时/取消: %w", ctx.Err()))
		return nil, ctx.Err()
	case r := <-ch:
		if r.err != nil {
			s.kill(r.err)
			return nil, s.dead
		}
		return r.raw, nil
	}
}

// kill 判死会话并切断连接（幂等；调用方须持锁）。
func (s *Session) kill(cause error) {
	if s.dead == nil {
		s.dead = fmt.Errorf("%w: %v", ErrSessionDead, cause)
	}
	_ = s.conn.Close()
}

// Abort 非阻塞强切传输层（不发 close-session、不取会话锁）：在途与后续操作
// 将以传输错误失败并自行判死会话。供上层「标记死连接」路径使用——那里绝不能
// 因等待在途操作而阻塞（对照 scrapligo 死连接 Close 死锁的根治）。幂等。
func (s *Session) Abort() {
	_ = s.conn.Close()
}

// Close 优雅关闭：best-effort 发 <close-session>（ctx 限时），随后必然切断
// 连接。幂等，可与 Do 并发调用。
func (s *Session) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.dead == nil {
		envelope := fmt.Sprintf(
			`<?xml version="1.0" encoding="UTF-8"?><rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="%d"><close-session/></rpc>`,
			s.msgID+1)
		s.msgID++
		// 应答内容不校验：目的只是让设备干净收会话，超时就硬切
		_, _ = s.exchangeLocked(ctx, []byte(envelope))
	}
	return s.conn.Close()
}
