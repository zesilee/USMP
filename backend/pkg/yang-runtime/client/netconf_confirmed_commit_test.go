package client

import (
	"context"
	"errors"
	"testing"
	"time"

	netsim "github.com/leezesi/usmp/backend/simulator/netconfsim"
)

// DP-08 — confirmed-commit 原语：CommitConfirmed 发送带 <confirmed/> 的 commit，
// ConfirmCommit 发确认 commit 转正；超时未确认由设备侧自动回滚（回读可验证）；
// 设备未宣告 :confirmed-commit capability 时明确报错且不发送 RPC。

const txVlanXML = `<vlan xmlns="urn:huawei:params:xml:ns:yang:huawei-vlan"><vlans><vlan><id>100</id><name>tx-vlan</name></vlan></vlans></vlan>`

// supportsConfirmedCommit 能力判定：兼容 :1.0/:1.1 URN，不被无关 capability 误命中。
func TestSupportsConfirmedCommit(t *testing.T) {
	cases := []struct {
		name string
		caps []string
		want bool
	}{
		{"v1.1", []string{"urn:ietf:params:netconf:capability:confirmed-commit:1.1"}, true},
		{"v1.0", []string{"urn:ietf:params:netconf:capability:confirmed-commit:1.0"}, true},
		{"absent", []string{"urn:ietf:params:netconf:base:1.0", "urn:ietf:params:netconf:capability:candidate:1.0"}, false},
		{"empty", nil, false},
		{"unrelated-substring", []string{"urn:example:yang:confirmed-commit-tricks"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := supportsConfirmedCommit(tc.caps); got != tc.want {
				t.Errorf("supportsConfirmedCommit(%v) = %v, want %v", tc.caps, got, tc.want)
			}
		})
	}
}

func prepareCandidateVlan(t *testing.T, c *NETCONFClient) {
	t.Helper()
	res, err := c.Set(context.Background(), []Change{{
		Type:     AddChange,
		Path:     "/vlan:vlan/vlan:vlans",
		NewValue: txVlanXML,
	}}, WithCommit(false))
	if err != nil || !res.Success {
		t.Fatalf("prepare candidate: err=%v res=%+v", err, res)
	}
}

func waitVlanGone(t *testing.T, sim *netsim.Simulator, timeout time.Duration) map[uint16]string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var vlans map[uint16]string
	for time.Now().Before(deadline) {
		vlans = sim.RunningHuaweiVLANs()
		if _, ok := vlans[100]; !ok {
			return vlans
		}
		time.Sleep(50 * time.Millisecond)
	}
	return vlans
}

// 场景 1：confirmed-commit 后确认转正——配置在原确认窗口过后仍在 running。
func TestCommitConfirmed_ConfirmKeepsConfig_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim := startSim(t)
	c := newSimClient(t, sim)
	ctx := context.Background()

	prepareCandidateVlan(t, c)
	if err := c.CommitConfirmed(ctx, 1*time.Second); err != nil {
		t.Fatalf("CommitConfirmed: %v", err)
	}
	if _, ok := sim.RunningHuaweiVLANs()[100]; !ok {
		t.Fatal("vlan 100 should be in running during confirm window")
	}
	if err := c.ConfirmCommit(ctx); err != nil {
		t.Fatalf("ConfirmCommit: %v", err)
	}
	time.Sleep(1300 * time.Millisecond)
	if _, ok := sim.RunningHuaweiVLANs()[100]; !ok {
		t.Fatal("vlan 100 rolled back despite confirming commit")
	}
}

// 场景 2：超时未确认自动回滚——running 回读不到该配置。
func TestCommitConfirmed_TimeoutRollsBack_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim := startSim(t)
	c := newSimClient(t, sim)

	prepareCandidateVlan(t, c)
	if err := c.CommitConfirmed(context.Background(), 1*time.Second); err != nil {
		t.Fatalf("CommitConfirmed: %v", err)
	}
	if _, ok := sim.RunningHuaweiVLANs()[100]; !ok {
		t.Fatal("vlan 100 should be in running during confirm window")
	}
	vlans := waitVlanGone(t, sim, 3*time.Second)
	if _, ok := vlans[100]; ok {
		t.Fatalf("vlan 100 should roll back after confirm timeout, running vlans: %v", vlans)
	}
}

// 场景 3：能力缺失——明确报错（可 errors.Is 判定供上层降级），不产生任何提交。
func TestCommitConfirmed_CapabilityMissing_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim := netsim.NewSimulator()
	sc := netsim.NewScenarioConfig()
	sc.DisableConfirmedCommit = true
	sim.SetScenario(sc)
	if err := sim.Start(); err != nil {
		t.Fatalf("start simulator: %v", err)
	}
	t.Cleanup(sim.Stop)
	c := newSimClient(t, sim)

	prepareCandidateVlan(t, c)
	err := c.CommitConfirmed(context.Background(), 1*time.Second)
	if err == nil {
		t.Fatal("CommitConfirmed should fail when capability is missing")
	}
	if !errors.Is(err, ErrConfirmedCommitUnsupported) {
		t.Fatalf("want ErrConfirmedCommitUnsupported, got %v", err)
	}
	if _, ok := sim.RunningHuaweiVLANs()[100]; ok {
		t.Fatal("no config should reach running when capability is missing")
	}
}
