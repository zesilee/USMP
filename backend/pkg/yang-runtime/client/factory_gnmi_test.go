package client

import (
	"strings"
	"testing"
	"time"
)

// DP-02（retire-idle-scaffolds）：gNMI 空壳已删——GNMI/AUTO+9339 必须显式
// 「尚未实现（规划能力）」错误，SHALL NOT 返回伪装成功的 client。

func TestFactory_GNMIExplicitlyUnimplemented(t *testing.T) {
	factory := DefaultClientFactory(5 * time.Second)
	cases := []struct {
		name string
		info DeviceConnectionInfo
	}{
		{"Protocol=GNMI", DeviceConnectionInfo{IP: "10.0.0.1", Port: 9339, Protocol: ProtocolGNMI}},
		{"AUTO+9339", DeviceConnectionInfo{IP: "10.0.0.1", Port: 9339, Protocol: ProtocolAUTO}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cli, err := factory(c.info)
			if err == nil || cli != nil {
				t.Fatalf("want explicit unimplemented error, got cli=%v err=%v", cli, err)
			}
			if !strings.Contains(err.Error(), "gNMI") || !strings.Contains(err.Error(), "未实现") {
				t.Fatalf("error should name gNMI 未实现（规划能力）: %v", err)
			}
		})
	}
}

// AUTO 非 9339 端口行为不变：仍路由到 NETCONF（无真设备时表现为 NETCONF 拨号错，
// 证明分派落点正确即可）。
func TestFactory_AUTONonGNMIPortsUnchanged(t *testing.T) {
	factory := DefaultClientFactory(1 * time.Second)
	for _, port := range []int{0, 830, 2830} {
		cli, err := factory(DeviceConnectionInfo{IP: "10.255.255.1", Port: port, Protocol: ProtocolAUTO})
		if err != nil && !strings.Contains(err.Error(), "NETCONF") {
			t.Fatalf("port %d: should route to NETCONF, got err=%v", port, err)
		}
		if err == nil && cli == nil {
			t.Fatalf("port %d: nil client without error", port)
		}
	}
}

// 未知协议错误保持。
func TestFactory_UnknownProtocolRejected(t *testing.T) {
	factory := DefaultClientFactory(5 * time.Second)
	if _, err := factory(DeviceConnectionInfo{IP: "10.0.0.1", Protocol: Protocol("telnet")}); err == nil {
		t.Fatal("unknown protocol should error")
	}
}
