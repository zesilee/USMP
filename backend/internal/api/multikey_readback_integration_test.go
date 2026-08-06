package api

import (
	"context"
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	netsim "github.com/leezesi/usmp/backend/simulator/netconfsim"
)

// B2 端到端（回归：devm 多键列表空表）：模拟网元种子含多键列表
// physical-entitys/physical-entity（key: class+position+serial-number）→ NETCONF
// get-config 回读 → decodeRunningConfig 剥层——修复前解码报 multi-key lists
// unsupported 整树降级原始透传，前端零行；修复后须解出可渲染数据行且键叶真值正确。
func TestMultiKeyListReadback_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	sim := netsim.NewSimulator()
	if err := sim.Start(); err != nil {
		t.Fatalf("start sim: %v", err)
	}
	defer sim.Stop()

	sim.SetRunningConfigXML([]byte(`<devm xmlns="urn:huawei:yang:huawei-devm"><physical-entitys>` +
		`<physical-entity><class>chassis</class><position>1</position><serial-number>SN-A</serial-number><name>Chassis 1</name></physical-entity>` +
		`<physical-entity><class>port</class><position>1/0/1</position><serial-number>SN-B</serial-number><name>GE1/0/1</name></physical-entity>` +
		`</physical-entitys></devm>`))

	pool := client.NewDefaultClientPool(client.DefaultClientFactory(5 * time.Second))
	defer pool.CloseAll()
	cli, err := pool.Get(client.DeviceConnectionInfo{
		IP: sim.Addr(), Port: sim.Port(), Username: sim.Username(), Password: sim.Password(), Protocol: client.ProtocolNETCONF,
	})
	if err != nil {
		t.Fatalf("pool get: %v", err)
	}

	path := "/devm:devm/devm:physical-entitys"
	result, err := cli.Get(context.Background(), path, client.WithDatastore("running"))
	if err != nil {
		t.Fatalf("device get: %v", err)
	}

	out := decodeRunningConfig(path, result.Data)
	m, ok := out.(map[string]interface{})
	if !ok {
		t.Fatalf("decode degraded to %T（多键解码失败原始透传，前端零行回归）, want map", out)
	}
	rows, ok := m["physical-entity"].([]interface{})
	if !ok {
		t.Fatalf("physical-entity rows missing, top-level: %v", m)
	}
	if len(rows) != 2 {
		t.Fatalf("want 2 rows, got %d: %v", len(rows), rows)
	}
	seen := map[string]bool{}
	for _, r := range rows {
		row, ok := r.(map[string]interface{})
		if !ok {
			t.Fatalf("row is %T, want map", r)
		}
		for _, k := range []string{"class", "position", "serial-number", "name"} {
			if _, ok := row[k]; !ok {
				t.Errorf("row missing field %q: %v", k, row)
			}
		}
		if sn, _ := row["serial-number"].(string); sn != "" {
			seen[sn] = true
		}
	}
	if !seen["SN-A"] || !seen["SN-B"] {
		t.Errorf("key leaves not faithfully decoded, serial-numbers seen: %v", seen)
	}
}
