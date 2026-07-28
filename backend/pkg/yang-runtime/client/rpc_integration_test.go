package client

import (
	"context"
	"fmt"
	"strings"
	"testing"

	netsim "github.com/leezesi/usmp/backend/simulator/netconfsim"
)

const ifmNS = "urn:huawei:params:xml:ns:yang:huawei-ifm"

// DP-10/NS-09 端到端：client.ExecuteRPC → 模拟网元识别 custom rpc、记录、返回 ok/
// rpc-error。覆盖成功 + 设备拒绝负路径（T02）。
func TestExecuteRPC_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim := startSim(t)
	c := newSimClient(t, sim)

	// 成功：执行 reset-if-counters-by-name(if-name=GE0/0/1) → sim 记录 → ok。
	res, err := c.ExecuteRPC(context.Background(), ifmNS,
		"reset-if-counters-by-name", []RPCInput{{Name: "if-name", Value: "GE0/0/1"}})
	if err != nil {
		t.Fatalf("ExecuteRPC: %v", err)
	}
	if !res.OK {
		t.Errorf("应返回 OK, got %+v (reply=%s)", res, res.Reply)
	}

	// 模拟网元记录了该调用（op + input 正确）。
	recs := sim.RecordedRPCs()
	if len(recs) != 1 {
		t.Fatalf("sim 应记录 1 次 rpc, got %d: %+v", len(recs), recs)
	}
	if recs[0].Op != "reset-if-counters-by-name" || recs[0].Inputs["if-name"] != "GE0/0/1" {
		t.Errorf("记录不符: %+v", recs[0])
	}

	// 负路径：设备拒绝 → rpc-error → 调用方拿到错误。
	sc := netsim.NewScenarioConfig()
	sc.ErrorOnRPC["restart-if"] = fmt.Errorf("interface is busy")
	sim.SetScenario(sc)

	res2, err2 := c.ExecuteRPC(context.Background(), ifmNS,
		"restart-if", []RPCInput{{Name: "if-name", Value: "GE0/0/1"}})
	if err2 == nil {
		t.Error("设备拒绝应返回错误")
	}
	if res2 == nil || res2.OK {
		t.Errorf("拒绝时不应 OK: %+v", res2)
	}
	if res2 != nil && res2.Error != nil && !strings.Contains(res2.Error.Error(), "busy") {
		t.Errorf("错误应含设备原因 'busy': %v", res2.Error)
	}
}
