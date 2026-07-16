package device

import (
	"fmt"
	"sync"
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
)

// DS-06 建连解析统一 helper：已注册走库、未注册/无库统一 AUTO+空凭据兜底（R08）。
func TestResolveConn(t *testing.T) {
	registered := client.DeviceConnectionInfo{
		IP: "10.0.0.1", Port: 830, Username: "op", Password: "s3cret",
		Protocol: client.ProtocolNETCONF, Vendor: "huawei",
	}
	withDevice := NewStore()
	withDevice.Put("10.0.0.1", registered)
	emptyVendor := client.DeviceConnectionInfo{IP: "10.0.0.2", Port: 830, Protocol: client.ProtocolGNMI}
	withDevice.Put("10.0.0.2", emptyVendor)

	tests := []struct {
		name           string
		store          Store
		deviceID       string
		wantInfo       client.DeviceConnectionInfo
		wantRegistered bool
	}{
		{
			name:  "已注册返回库中完整连接信息",
			store: withDevice, deviceID: "10.0.0.1",
			wantInfo: registered, wantRegistered: true,
		},
		{
			name:  "Vendor 零值原样透传（缺省语义在消费方）",
			store: withDevice, deviceID: "10.0.0.2",
			wantInfo: emptyVendor, wantRegistered: true,
		},
		{
			name:  "未注册兜底 AUTO+空凭据",
			store: withDevice, deviceID: "10.9.9.9",
			wantInfo:       client.DeviceConnectionInfo{IP: "10.9.9.9", Protocol: client.ProtocolAUTO},
			wantRegistered: false,
		},
		{
			name:  "nil store 兜底 AUTO+空凭据（legacy path）",
			store: nil, deviceID: "10.0.0.1",
			wantInfo:       client.DeviceConnectionInfo{IP: "10.0.0.1", Protocol: client.ProtocolAUTO},
			wantRegistered: false,
		},
		{
			name:  "空 deviceID 兜底不 panic（边界）",
			store: withDevice, deviceID: "",
			wantInfo:       client.DeviceConnectionInfo{IP: "", Protocol: client.ProtocolAUTO},
			wantRegistered: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, ok := ResolveConn(tt.store, tt.deviceID)
			if ok != tt.wantRegistered {
				t.Fatalf("registered = %v, want %v", ok, tt.wantRegistered)
			}
			if info != tt.wantInfo {
				t.Fatalf("info = %+v, want %+v", info, tt.wantInfo)
			}
		})
	}
}

// R09：并发解析与注册/注销无数据竞态（-race）。
func TestResolveConnConcurrent(t *testing.T) {
	s := NewStore()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			ip := fmt.Sprintf("10.1.0.%d", n)
			for j := 0; j < 100; j++ {
				s.Put(ip, client.DeviceConnectionInfo{IP: ip, Protocol: client.ProtocolNETCONF})
				s.Delete(ip)
			}
		}(i)
		go func(n int) {
			defer wg.Done()
			ip := fmt.Sprintf("10.1.0.%d", n)
			for j := 0; j < 100; j++ {
				info, _ := ResolveConn(s, ip)
				if info.IP != ip {
					t.Errorf("info.IP = %q, want %q", info.IP, ip)
					return
				}
			}
		}(i)
	}
	wg.Wait()
}
