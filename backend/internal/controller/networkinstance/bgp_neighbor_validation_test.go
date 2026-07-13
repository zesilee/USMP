package networkinstance

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/openconfig/ygot/ygot"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/driver"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/xmlcodec"
	netsim "github.com/leezesi/usmp/backend/simulator/netconfsim"
)

// BN-03 边界：peer.description（length 1..255、pattern 不含 '?'）由 ygot ΛValidate 强制。
// 记录事实：remote-as 虽 YANG mandatory，但 ygot ΛValidate **不强制** list-entry mandatory
// leaf（设备侧/API 层须另兜底）——实测校准，防臆断。
func TestBN_PeerBoundary_Description(t *testing.T) {
	cases := []struct {
		name    string
		desc    string
		wantErr bool
	}{
		{"valid", "a normal peer", false},
		{"max-255", strings.Repeat("d", 255), false},
		{"over-255", strings.Repeat("d", 256), true},
		{"question-mark", "bad?desc", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ni := niWithPublicPeers(peer("10.0.0.1", "100", tc.desc))
			err := ni.ΛValidate()
			if tc.wantErr && err == nil {
				t.Fatalf("期望校验失败：desc len=%d", len(tc.desc))
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("期望校验通过，实际: %v", err)
			}
		})
	}
}

// remote-as 缺失：ygot ΛValidate 不强制（事实锁定，非本模块可修——需设备/API 层兜底）。
func TestBN_PeerRemoteAsNotYgotEnforced(t *testing.T) {
	ni := niWithPublicPeers(&peerT{Address: ygot.String("10.0.0.1")})
	if err := ni.ΛValidate(); err != nil {
		t.Fatalf("记录事实：ygot ΛValidate 当前不强制 remote-as mandatory，若某版本开始强制则更新此断言: %v", err)
	}
}

// BN-04 负路径（防越序）：只 set 基础邻居字段时，策略属性/状态容器不出现在下发报文。
func TestBN_PeerNegative_NoPolicyOrStateInEncode(t *testing.T) {
	enc, ok := driver.EncoderFor("/ni:network-instance")
	if !ok || enc.XML == nil {
		t.Fatal("ni 描述符缺失")
	}
	p := peer("10.0.0.1", "100", "basic")
	xml, err := xmlcodec.Encode(enc.XML, niWithPublicPeers(p))
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	// 2b 策略属性与 config-false 状态：本期不 set → 不应出现
	for _, bad := range []string{
		"route-policy", "route-filter", "<acl", "tunnel-policy",
		"peer-groups", "dynamic-peer-prefixes", "egress-engineer",
		"-state>", "bfd-session-state",
	} {
		if strings.Contains(xml, bad) {
			t.Errorf("越序/状态字段 %q 不应出现在下发报文\n%s", bad, xml)
		}
	}
	// 基础字段在
	for _, want := range []string{"<address>10.0.0.1</address>", "<remote-as>100</remote-as>", "<description>basic</description>"} {
		if !strings.Contains(xml, want) {
			t.Errorf("基础字段缺 %q\n%s", want, xml)
		}
	}
}

// BN-03 并发：多协程并发对含 peers 的 network-instance reconcile 无竞态无 panic。
func TestReconciler_Integration_BgpNeighborConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim := netsim.NewSimulator()
	if err := sim.Start(); err != nil {
		t.Fatalf("start simulator: %v", err)
	}
	defer sim.Stop()

	r, cs, _, deviceID, pool := newHarness(t, sim)
	defer pool.CloseAll()
	ctx := context.Background()
	req := reconcile.Request{DeviceID: deviceID, Path: NetworkInstancePath}
	_ = cs.Set(deviceID, NetworkInstancePath, niWithPublicPeers(
		peer("10.0.0.1", "100", "a"), peer("10.0.0.2", "200", "b")))

	const n = 8
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) { defer wg.Done(); errs[idx] = r.Reconcile(ctx, req).Error }(i)
	}
	wg.Wait()
	for i, e := range errs {
		if e != nil {
			t.Errorf("并发 reconcile[%d]: %v", i, e)
		}
	}
}
