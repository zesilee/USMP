package source

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

// 回归：FileSource 的 watcher 错误必须落日志。
//
// 病灶（code-todo-backlog B2）：原实现 `case err := <-s.watcher.Errors:` 里只有
// 一句 `_ = err`，注释写着「Log error but continue」但没有 log——文件监听退化
// （句柄耗尽、挂载点消失等）后事件源静默失效，desired 不再被触发对账，现象是
// 「改了文件没反应」且无任何线索。继续监听没错，但错误不能丢。

// syncBuffer 是并发安全的日志接收缓冲：run 在独立协程里写日志，用例在主协程读，
// 裸 bytes.Buffer 会被 -race 判定数据竞态（R09）。
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func captureLogConcurrent(t *testing.T) *syncBuffer {
	t.Helper()
	b := &syncBuffer{}
	prevOut, prevFlags := log.Writer(), log.Flags()
	log.SetOutput(b)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
	})
	return b
}

// runningFileSource 起一个只挂在手搓 watcher 上的 FileSource：不碰真实文件系统，
// 直接把错误推进 Errors 通道，复刻 fsnotify 上报监听故障的形态。
// 不走 Start()/Stop()，因为手搓的 Watcher 没有内部句柄，Close 会崩。
func runningFileSource(t *testing.T) (*FileSource, chan error, context.CancelFunc) {
	t.Helper()
	w := &fsnotify.Watcher{
		Events: make(chan fsnotify.Event),
		Errors: make(chan error, 1),
	}
	s := NewFileSource("/does/not/matter.json", "10.0.0.1", "/vlan:vlan")
	s.watcher = w
	s.done = make(chan struct{})
	s.wg.Add()

	ctx, cancel := context.WithCancel(context.Background())
	go s.run(ctx)

	t.Cleanup(func() {
		cancel()
		s.wg.Wait()
	})
	return s, w.Errors, cancel
}

// waitForLog 轮询等待日志出现，避免靠 sleep 猜时序（弱机上会假失败）。
func waitForLog(t *testing.T, b *syncBuffer, want string) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if out := b.String(); strings.Contains(out, want) {
			return out
		}
		time.Sleep(5 * time.Millisecond)
	}
	return b.String()
}

func TestFileSourceLogsWatcherError(t *testing.T) {
	logged := captureLogConcurrent(t)
	_, errCh, _ := runningFileSource(t)

	errCh <- errors.New("inotify watch limit reached")

	out := waitForLog(t, logged, "inotify watch limit reached")
	if !strings.Contains(out, "inotify watch limit reached") {
		t.Errorf("watcher 错误必须落日志，实际日志为 %q", out)
	}
	if !strings.Contains(out, "file-source:") {
		t.Errorf("日志需带 file-source 子系统前缀（对齐本仓 log.Printf 约定），实际 %q", out)
	}
}

// 上报错误后监听协程必须存活——降级语义是「记一笔继续看」，不是「就此退出」。
func TestFileSourceKeepsWatchingAfterError(t *testing.T) {
	logged := captureLogConcurrent(t)
	_, errCh, _ := runningFileSource(t)

	errCh <- errors.New("first failure")
	waitForLog(t, logged, "first failure")

	// 通道容量 1：第二条能塞进去，说明第一条已被消费、循环还在跑。
	select {
	case errCh <- errors.New("second failure"):
	case <-time.After(2 * time.Second):
		t.Fatal("首个错误后监听协程已退出，降级语义应为继续监听（R08）")
	}

	if out := waitForLog(t, logged, "second failure"); !strings.Contains(out, "second failure") {
		t.Errorf("后续 watcher 错误同样应落日志，实际 %q", out)
	}
}
