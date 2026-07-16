package intent

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeCacheStarter struct {
	err     error
	started chan struct{}
}

func (f *fakeCacheStarter) Start(ctx context.Context) error {
	close(f.started)
	if f.err != nil {
		return f.err
	}
	<-ctx.Done()
	return ctx.Err()
}

// StartCache(nil) 是无集群降级路径（Register 返回 nil cache）：必须立即返回、不 panic（R08）。
func TestStartCache_NilNoop(t *testing.T) {
	done := make(chan struct{})
	go func() {
		StartCache(context.Background(), nil)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("StartCache(nil) 应立即返回")
	}
}

// cache.Start 返回错误时只记日志不崩溃（R08）。
func TestStartCache_StartErrorLogged(t *testing.T) {
	f := &fakeCacheStarter{err: errors.New("boom"), started: make(chan struct{})}
	done := make(chan struct{})
	go func() {
		StartCache(context.Background(), f)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Start 出错后 StartCache 应返回")
	}
}

// 正常路径：阻塞运行直到 ctx 取消。
func TestStartCache_BlocksUntilCancel(t *testing.T) {
	f := &fakeCacheStarter{started: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		StartCache(ctx, f)
		close(done)
	}()
	<-f.started
	select {
	case <-done:
		t.Fatal("ctx 未取消前 StartCache 不应返回")
	case <-time.After(50 * time.Millisecond):
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ctx 取消后 StartCache 应返回")
	}
}
