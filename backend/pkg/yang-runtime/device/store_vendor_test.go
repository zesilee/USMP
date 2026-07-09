package device

import (
	"sync"
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
)

// DS-01: Vendor 随连接信息完整透传（SND 厂商注册表的数据通路，P5-1）。
func TestStore_VendorPassthrough(t *testing.T) {
	s := NewStore()
	s.Put("10.0.0.1", client.DeviceConnectionInfo{IP: "10.0.0.1", Vendor: "huawei"})

	got, ok := s.Get("10.0.0.1")
	if !ok || got.Vendor != "huawei" {
		t.Fatalf("Vendor 应随连接信息透传, got %+v ok=%v", got, ok)
	}
}

// DS-01 边界: Vendor 零值（存量数据）原样存取，缺省语义由消费方解读（R08）。
func TestStore_VendorZeroValueRoundTrip(t *testing.T) {
	s := NewStore()
	s.Put("10.0.0.2", client.DeviceConnectionInfo{IP: "10.0.0.2"})

	got, ok := s.Get("10.0.0.2")
	if !ok || got.Vendor != "" {
		t.Fatalf("Vendor 零值应原样往返, got %q", got.Vendor)
	}
}

// R09: 含 Vendor 的并发读写无竞态（-race 锁定）。
func TestStore_VendorConcurrentAccess(t *testing.T) {
	s := NewStore()
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			s.Put("x", client.DeviceConnectionInfo{IP: "x", Vendor: "huawei"})
		}()
		go func() {
			defer wg.Done()
			if info, ok := s.Get("x"); ok && info.Vendor != "" && info.Vendor != "huawei" {
				t.Error("并发下 Vendor 值撕裂")
			}
		}()
	}
	wg.Wait()
}
