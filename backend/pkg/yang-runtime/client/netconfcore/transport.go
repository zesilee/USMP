// transport.go — SSH netconf subsystem 传输（RFC 6242 §3）。
//
// Session 只依赖 io.ReadWriteCloser；本文件是生产供给方（x/crypto/ssh，
// 已有依赖，Go 1.22 兼容）。免严格 HostKey 对齐现网 scrapligo
// WithAuthNoStrictKey 行为（设备管理网内网段，密钥指纹管理另立后续任务）。
package netconfcore

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// DialSSH 建立 SSH 连接并打开 netconf subsystem，返回可直接交给 NewSession
// 的字节流。timeout 仅约束 TCP+SSH 握手（会话级操作超时由 ctx 管）。
func DialSSH(host string, port int, user, pass string, timeout time.Duration) (io.ReadWriteCloser, error) {
	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(pass)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // 对齐现网免严格 HostKey
		Timeout:         timeout,
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, fmt.Errorf("netconfcore: SSH 连接 %s: %w", addr, err)
	}
	sess, err := client.NewSession()
	if err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("netconfcore: SSH 会话: %w", err)
	}
	stdin, err := sess.StdinPipe()
	if err != nil {
		_ = sess.Close()
		_ = client.Close()
		return nil, fmt.Errorf("netconfcore: stdin 管道: %w", err)
	}
	stdout, err := sess.StdoutPipe()
	if err != nil {
		_ = sess.Close()
		_ = client.Close()
		return nil, fmt.Errorf("netconfcore: stdout 管道: %w", err)
	}
	if err := sess.RequestSubsystem("netconf"); err != nil {
		_ = sess.Close()
		_ = client.Close()
		return nil, fmt.Errorf("netconfcore: 打开 netconf subsystem: %w", err)
	}
	return &sshConn{client: client, sess: sess, stdin: stdin, stdout: stdout}, nil
}

// sshConn 把 SSH session 的 stdin/stdout 收敛为单个 ReadWriteCloser。
// Close 幂等（sync.Once）：关 TCP 即解锁一切在途读写（Session 看门狗依赖此语义）。
type sshConn struct {
	client   *ssh.Client
	sess     *ssh.Session
	stdin    io.WriteCloser
	stdout   io.Reader
	closeOne sync.Once
	closeErr error
}

func (c *sshConn) Read(p []byte) (int, error)  { return c.stdout.Read(p) }
func (c *sshConn) Write(p []byte) (int, error) { return c.stdin.Write(p) }

func (c *sshConn) Close() error {
	c.closeOne.Do(func() {
		_ = c.sess.Close()
		c.closeErr = c.client.Close()
	})
	return c.closeErr
}
