package client

import (
	"context"
	"testing"

	netsim "github.com/leezesi/usmp/backend/simulator/netconfsim"
)

// CN-04 端到端（tasks 2.3，T02）：sim 按路径注入 unknown-element → client.Get
// 拿到可归因的结构化错误（BadElement 走通整条线缆）→ 标记入集 → 断线重连清空。
func TestUnsupportedPathLearning_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim := startSim(t)
	sc := netsim.NewScenarioConfig()
	sc.UnknownElementPaths = []string{"ifm/interfaces"}
	sim.SetScenario(sc)
	c := newSimClient(t, sim)

	path := "ifm:ifm/ifm:interfaces"
	_, err := c.Get(context.Background(), path)
	if err == nil {
		t.Fatal("注入路径读取应失败")
	}
	if !UnknownElementForPath(path, err) {
		t.Fatalf("错误应可归因为 unknown-element（BadElement 须存活整条链路）: %v", err)
	}

	// API 层学习动作（此处手工模拟）：标记 → 快速失败判定命中。
	c.MarkUnsupportedPath(path)
	if !c.IsUnsupportedPath(path) {
		t.Fatal("标记后应命中")
	}

	// 其他路径不受注入影响（sim 精确性 + 会话未判死）。
	if _, err := c.Get(context.Background(), "ifm:ifm/ifm:global"); err != nil {
		t.Fatalf("未注入路径应正常: %v", err)
	}

	// 断线重连 → 不支持集清空（设备升级重学语义）。
	c.markDisconnected()
	if _, err := c.Get(context.Background(), "ifm:ifm/ifm:global"); err != nil {
		t.Fatalf("重连后读取应成功: %v", err)
	}
	if c.IsUnsupportedPath(path) {
		t.Fatal("重连后不支持集应清空")
	}
}
