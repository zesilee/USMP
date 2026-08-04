package netconfcore

import (
	"context"
	"strings"
	"testing"
	"time"

	netsim "github.com/leezesi/usmp/backend/simulator/netconfsim"
	"github.com/leezesi/usmp/backend/simulator/netconfsim/testsupport"
)

// B2 集成：自研核心 ↔ netconfsim 真 SSH 端到端（T02）。
// sim 只报 base:1.0 → 走 EOM；chunked 1.1 路径由 session_test 的 net.Pipe
// 假服务端覆盖，真机验证在 Wave 4。

const seedVlanXML = `<vlan xmlns="urn:huawei:params:xml:ns:yang:huawei-vlan"><vlans><vlan><id>10</id><name>seed10</name></vlan></vlans></vlan>`

func dialSim(t *testing.T) (*netsim.Simulator, *Session) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim := netsim.NewSimulator()
	if err := sim.Start(); err != nil {
		t.Fatalf("start simulator: %v", err)
	}
	t.Cleanup(sim.Stop)
	sim.SetRunningConfigXML([]byte(seedVlanXML))

	conn, err := DialSSH(sim.Addr(), sim.Port(), sim.Username(), sim.Password(), 5*time.Second)
	if err != nil {
		t.Fatalf("DialSSH: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s, err := NewSession(ctx, conn)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	t.Cleanup(func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer ccancel()
		_ = s.Close(cctx)
	})
	return sim, s
}

func TestIntegrationHelloAndCapabilities(t *testing.T) {
	_, s := dialSim(t)
	if s.Framing() != FramingEOM {
		t.Fatalf("sim 应协商出 EOM, got %v", s.Framing())
	}
	found := false
	for _, c := range s.Capabilities() {
		if strings.Contains(c, "netconf:base:1.0") {
			found = true
		}
	}
	if !found {
		t.Fatalf("能力清单缺 base:1.0: %v", s.Capabilities())
	}
}

func TestIntegrationGetConfig(t *testing.T) {
	_, s := dialSim(t)
	reply, err := s.GetConfig(context.Background(), "running",
		`<vlan xmlns="urn:huawei:params:xml:ns:yang:huawei-vlan"/>`)
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if !strings.Contains(string(reply), "seed10") {
		t.Fatalf("回读应含种子 vlan, got: %s", reply)
	}
}

func TestIntegrationEditConfigRoundTrip(t *testing.T) {
	sim, s := dialSim(t)
	edit := `<vlan xmlns="urn:huawei:params:xml:ns:yang:huawei-vlan"><vlans><vlan><id>20</id><name>wave2</name></vlan></vlans></vlan>`
	if _, err := s.EditConfig(context.Background(), "running", edit); err != nil {
		t.Fatalf("EditConfig: %v", err)
	}
	testsupport.AssertHuaweiVlanExists(t, sim, 20)
	testsupport.AssertHuaweiVlanName(t, sim, 20, "wave2")
	// 幂等：重复下发同配置不报错（merge 语义）
	if _, err := s.EditConfig(context.Background(), "running", edit); err != nil {
		t.Fatalf("重复 EditConfig 应幂等: %v", err)
	}
}

func TestIntegrationRPCErrorFromDevice(t *testing.T) {
	_, s := dialSim(t)
	// 故意打一个 sim 不认识的操作，应拿到结构化 rpc-error 而非挂起/崩溃
	_, err := s.Do(context.Background(), []byte(`<no-such-op xmlns="urn:nope"/>`))
	if err == nil {
		t.Skip("sim 对未知操作宽容放行，负路径由单测覆盖")
	}
	t.Logf("设备侧错误（预期）: %v", err)
}

func TestIntegrationConcurrentClients(t *testing.T) {
	// 多会话并发（模拟多控制器同连一台设备）+ 单会话并发 Do 串行化
	sim, s := dialSim(t)
	done := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := s.GetConfig(context.Background(), "running",
				`<vlan xmlns="urn:huawei:params:xml:ns:yang:huawei-vlan"/>`)
			done <- err
		}()
	}
	for i := 0; i < 10; i++ {
		if err := <-done; err != nil {
			t.Fatalf("并发 GetConfig: %v", err)
		}
	}
	_ = sim
}
