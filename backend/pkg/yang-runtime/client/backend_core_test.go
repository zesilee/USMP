package client

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// netconfcore 芯专项行为测试（端到端读写、断连自愈、Kill 非阻塞）。
// 历史：Wave 3 双路径期本文件配合 USMP_NETCONF_IMPL 开关使用；scrapligo
// 移除（NC-01）后 core 即唯一引擎，整套 client 测试天然全跑在其上。

func newCoreSimClient(t *testing.T) *NETCONFClient {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim := startSim(t)
	return newSimClient(t, sim)
}

func TestCoreImplGetConfigEndToEnd(t *testing.T) {
	c := newCoreSimClient(t)
	res, err := c.Get(context.Background(), "/ifm:ifm/ifm:interfaces")
	if err != nil {
		t.Fatalf("core Get: %v", err)
	}
	if !strings.Contains(fmt.Sprintf("%s", res.Data), "GE0/0/1") {
		t.Fatalf("core Get 应回读种子接口, got %.300s", res.Data)
	}
}

func TestCoreImplStateGetEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim := startSim(t)
	if err := sim.SetStateDataXML([]byte(testIfmStateXML)); err != nil {
		t.Fatalf("SetStateDataXML: %v", err)
	}
	c := newSimClient(t, sim)
	res, err := c.Get(context.Background(), "/ifm:ifm/ifm:interfaces", WithStateData())
	if err != nil {
		t.Fatalf("core state Get: %v", err)
	}
	if !strings.Contains(fmt.Sprintf("%s", res.Data), "oper-status") {
		t.Fatalf("core state Get 缺状态叶, got %.300s", res.Data)
	}
}

func TestCoreImplSelfHealAfterConnectionLoss(t *testing.T) {
	c := newCoreSimClient(t)
	ctx := context.Background()
	if _, err := c.Get(ctx, "/ifm:ifm/ifm:interfaces"); err != nil {
		t.Fatalf("initial get: %v", err)
	}
	// 弄死连接但不改 connected（复用现网自愈回归的手法）
	c.mu.Lock()
	dead := c.backend
	c.mu.Unlock()
	dead.Kill()

	var lastErr error
	for i := 0; i < 3; i++ {
		res, err := c.Get(ctx, "/ifm:ifm/ifm:interfaces")
		if err == nil && strings.Contains(fmt.Sprintf("%s", res.Data), "GE0/0/1") {
			return // 自愈成功
		}
		lastErr = err
	}
	t.Fatalf("core 路径断连后未自愈: %v", lastErr)
}

func TestCoreImplKillNonBlocking(t *testing.T) {
	c := newCoreSimClient(t)
	if _, err := c.Get(context.Background(), "/ifm:ifm/ifm:interfaces"); err != nil {
		t.Fatalf("initial get: %v", err)
	}
	done := make(chan struct{})
	go func() {
		c.markDisconnected()
		c.markDisconnected() // 幂等
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("core Kill/markDisconnected 不得阻塞")
	}
	if c.IsConnected() {
		t.Fatal("markDisconnected 后应为断连态")
	}
}
