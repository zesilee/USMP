package driver

import (
	"fmt"
	"sync"
	"testing"
)

// DR-04: 厂商支持性查询——大小写无关命中、未注册 false、不 panic。
func TestRegistry_VendorSupported(t *testing.T) {
	r := NewRegistry()
	r.Register(testDescriptor("huawei", "vlan", "vlan"))
	r.Register(testDescriptor("huawei", "ifm", "ifm"))

	cases := []struct {
		name   string
		vendor string
		want   bool
	}{
		{"精确匹配", "huawei", true},
		{"大小写无关-首字母大写", "Huawei", true},
		{"大小写无关-全大写", "HUAWEI", true},
		{"未注册厂商", "nokia", false},
		{"枚举遗留厂商无描述符", "cisco", false},
		{"空串", "", false},
		{"前后缀不算命中", "huawei2", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := r.VendorSupported(tc.vendor); got != tc.want {
				t.Fatalf("VendorSupported(%q) = %v, want %v", tc.vendor, got, tc.want)
			}
		})
	}
}

// DR-04: 空注册表查询不 panic（R08）。
func TestRegistry_VendorSupportedEmpty(t *testing.T) {
	r := NewRegistry()
	if r.VendorSupported("huawei") {
		t.Fatal("空注册表应返回 false")
	}
}

// DR-04: Register 与 VendorSupported 并发无数据竞态（R09，-race 兜底）。
func TestRegistry_VendorSupportedConcurrent(t *testing.T) {
	r := NewRegistry()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			r.Register(testDescriptor(fmt.Sprintf("vendor%d", i), "m", "m"))
		}(i)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = r.VendorSupported("huawei")
			}
		}()
	}
	wg.Wait()
	if !r.VendorSupported("VENDOR3") {
		t.Fatal("并发注册后应可大小写无关查得 vendor3")
	}
}

// DR-04: 包级 facade 走默认注册表（生产接线 init() 已注册 huawei 描述符）。
func TestVendorSupportedFacade(t *testing.T) {
	if VendorSupported("no-such-vendor") {
		t.Fatal("默认注册表不应支持 no-such-vendor")
	}
}
