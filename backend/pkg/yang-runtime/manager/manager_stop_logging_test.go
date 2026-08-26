package manager

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
)

// 回归：Manager.Stop 关停连接池失败时，错误必须落到日志。
//
// 病灶（code-todo-backlog B1）：原实现是 `if err := CloseAll(); err != nil {}` 空块，
// 注释写着「Log but continue shutdown」但块内一行 log 都没有——关停期连接池故障
// 完全不可观测。R08 要求异常降级处理，「继续关停」没错，但错误不能人间蒸发。

// stopFailPool 是只让 CloseAll 失败的 ClientPool 桩，其余方法不参与本用例。
type stopFailPool struct {
	err    error
	closed int
}

func (p *stopFailPool) Get(client.DeviceConnectionInfo) (client.Client, error) {
	return nil, errors.New("not used in this test")
}
func (p *stopFailPool) Release(string) {}
func (p *stopFailPool) CloseAll() error {
	p.closed++
	return p.err
}
func (p *stopFailPool) Stats() client.PoolStats { return client.PoolStats{} }

// startedManager 造一个「已启动」的 Manager：Stop 在 started=false 时直接返回，
// 而 cancel 只在 Start 里赋值，故此处补齐这两项，避开真实设备连接。
func startedManager(t *testing.T, pool client.ClientPool) *DefaultManager {
	t.Helper()
	m := New()
	m.clientPool = pool
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.started = true
	return m
}

// captureLog 把标准库 log 的输出接到内存缓冲区，返回取值函数。
// log 输出是进程级全局状态，本包用例不并行，用完即还原。
func captureLog(t *testing.T) func() string {
	t.Helper()
	var buf bytes.Buffer
	prevOut, prevFlags := log.Writer(), log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
	})
	return buf.String
}

func TestStopLogsClientPoolCloseFailure(t *testing.T) {
	logged := captureLog(t)
	pool := &stopFailPool{err: errors.New("ssh transport already dead")}

	m := startedManager(t, pool)
	if err := m.Stop(); err != nil {
		t.Fatalf("Stop 不应因关连接失败而报错（R08 继续关停）: %v", err)
	}

	if pool.closed != 1 {
		t.Fatalf("CloseAll 应被调用 1 次，实际 %d 次", pool.closed)
	}

	out := logged()
	if !strings.Contains(out, "ssh transport already dead") {
		t.Errorf("连接池关停失败必须落日志，实际日志为 %q", out)
	}
	if !strings.Contains(out, "manager:") {
		t.Errorf("日志需带 manager 子系统前缀（对齐本仓 log.Printf 约定），实际 %q", out)
	}
}

// 关停成功时不该产生噪声日志——否则每次正常关停都刷一行无用错误。
func TestStopSilentWhenClientPoolClosesCleanly(t *testing.T) {
	logged := captureLog(t)
	pool := &stopFailPool{err: nil}

	m := startedManager(t, pool)
	if err := m.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if out := logged(); strings.Contains(out, "manager:") {
		t.Errorf("关停无异常时不应打 manager 日志，实际 %q", out)
	}
}
