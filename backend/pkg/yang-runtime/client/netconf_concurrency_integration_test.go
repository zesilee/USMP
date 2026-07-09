package client

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	netsim "github.com/leezesi/usmp/backend/simulator/netconfsim"
)

// 回归测试：并发 RPC 竞态 + 死连接不自愈（生产表现为「新建 interface 失败，
// GET /config 持续 500 EOF，reconcile ifm 路径 error: EOF，直到重启后端」）。
//
// 根因 1：NETCONFClient.Get/Set 仅持读锁，允许多个 goroutine 并发调用同一条
// scrapligo Driver。scrapligo 非并发安全——buildPayload 的 messageID++ 无锁
// （数据竞态 → 重复 message-id → 响应被错领/丢失 → 60s op-timeout 挂起），
// Channel.Write 无锁（帧字节交错 → 模拟器/设备侧 NETCONF 帧解析卡死）。
// 前端模块控制台并行拉取 7 个 ifm 子树即触发。
//
// 根因 2：传输层死亡（EOF/超时/对端关闭）后 connected 标志仍为 true，
// ClientPool.Get 的 IsConnected() 检查形同虚设，死客户端被永久复用，
// 之后所有请求瞬间 EOF。

const testIfmRunningXML = `<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm"><interfaces><interface><name>GE0/0/1</name><type>93</type><class>1</class><mtu>1500</mtu></interface></interfaces></ifm>`

func startSim(t *testing.T) *netsim.Simulator {
	t.Helper()
	sim := netsim.NewSimulator()
	if err := sim.Start(); err != nil {
		t.Fatalf("start simulator: %v", err)
	}
	t.Cleanup(sim.Stop)
	sim.SetRunningConfigXML([]byte(testIfmRunningXML))
	return sim
}

func newSimClient(t *testing.T, sim *netsim.Simulator) *NETCONFClient {
	t.Helper()
	c, err := NewNETCONFClient(DeviceConnectionInfo{
		IP:       sim.Addr(),
		Port:     sim.Port(),
		Username: sim.Username(),
		Password: sim.Password(),
		Protocol: ProtocolNETCONF,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("connect to simulator: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// TestNETCONFClient_ConcurrentGet_SingleConnection：多 goroutine 并发 Get 同一
// 客户端（复刻前端并行拉多个 YANG 子树），要求全部成功且响应携带预期数据。
// 需配合 -race 运行：串行化缺失时 scrapligo messageID++ 会被竞态检测器命中，
// 帧交错则表现为超时/EOF/空响应导致断言失败。
func TestNETCONFClient_ConcurrentGet_SingleConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	sim := startSim(t)
	c := newSimClient(t, sim)

	paths := []string{
		"/ifm:ifm/ifm:interfaces",
		"/ifm:ifm/ifm:global",
		"/ifm:ifm/ifm:damp",
		"/vlan:vlan/vlan:vlans",
		"/system:system",
	}
	const rounds = 4

	var wg sync.WaitGroup
	errCh := make(chan error, len(paths)*rounds)
	for _, p := range paths {
		for i := 0; i < rounds; i++ {
			wg.Add(1)
			go func(path string, round int) {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()
				res, err := c.Get(ctx, path)
				if err != nil {
					errCh <- fmt.Errorf("get %s round %d: %w", path, round, err)
					return
				}
				if !strings.Contains(fmt.Sprintf("%s", res.Data), "GE0/0/1") {
					errCh <- fmt.Errorf("get %s round %d: response missing seeded interface, got %.200s", path, round, res.Data)
				}
			}(p, i)
		}
	}

	waitDone := make(chan struct{})
	go func() { wg.Wait(); close(waitDone) }()
	select {
	case <-waitDone:
	case <-time.After(45 * time.Second):
		t.Fatal("concurrent gets deadlocked (NETCONF framing corrupted / responses lost)")
	}
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}
}

// TestNETCONFClient_ReconnectAfterConnectionLoss：底层 NETCONF 会话死亡后
// （设备重启、网络闪断、scrapligo 超时后关闭通道），客户端必须能自愈重连，
// 而不是永久返回 EOF。
//
// 注入方式：直接优雅关闭底层 driver 但保留 connected=true——这正是生产中
// 「传输层已死、客户端仍自认在线」的状态。不用「杀模拟器再重启」的方式注入，
// 因为 scrapligo v1.4.0 的 channel reader 在连接被对端强杀时有内部数据竞态，
// 会让本测试在 -race 下因第三方竞态误报失败。
func TestNETCONFClient_ReconnectAfterConnectionLoss(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	sim := startSim(t)
	c := newSimClient(t, sim)

	ctx := context.Background()
	if _, err := c.Get(ctx, "/ifm:ifm/ifm:interfaces"); err != nil {
		t.Fatalf("initial get must succeed: %v", err)
	}

	// 制造「死连接但自认在线」：关掉 driver，不改 connected/driver 字段。
	c.mu.Lock()
	deadDriver := c.driver
	c.mu.Unlock()
	if deadDriver == nil {
		t.Fatal("expected live driver after successful get")
	}
	if err := deadDriver.Close(); err != nil {
		t.Fatalf("close underlying driver: %v", err)
	}

	// 修复前：connected 恒为 true、driver 不重建，这里每次调用都瞬间失败
	// （生产表现为持续 500 EOF 直到重启后端）。修复后：Get 识别传输层错误
	// → markDisconnected → 重连 → 单次重试成功。
	var lastErr error
	for i := 0; i < 3; i++ {
		res, err := c.Get(ctx, "/ifm:ifm/ifm:interfaces")
		if err == nil && strings.Contains(fmt.Sprintf("%s", res.Data), "GE0/0/1") {
			return // 自愈成功
		}
		lastErr = err
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("client never recovered after connection loss, last error: %v", lastErr)
}
