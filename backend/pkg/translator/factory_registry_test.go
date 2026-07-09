package translator

import (
	"sync"
	"testing"
)

// TE-01: huawei 翻译器经实现文件 init() 编译期自注册——进程启动即可得，
// 不依赖 GetTranslator 内的延迟初始化（该 once.Do 硬注册应被移除）。
func TestVendorRegistry_HuaweiSelfRegisteredViaInit(t *testing.T) {
	if !IsVendorSupported(VendorHuawei) {
		t.Fatal("huawei 应经 init() 编译期自注册，IsVendorSupported 在任何 GetTranslator 调用前即为 true")
	}
	tr, err := GetTranslator(VendorHuawei)
	if err != nil || tr == nil {
		t.Fatalf("GetTranslator(huawei) 应可得: %v", err)
	}
	if tr.Vendor() != VendorHuawei {
		t.Fatalf("翻译器厂商标识应为 huawei，got %s", tr.Vendor())
	}
}

// TE-01: 未注册厂商（枚举存在但无实现）明确报错，不 panic（R08）。
func TestVendorRegistry_UnregisteredVendorError(t *testing.T) {
	if IsVendorSupported(VendorCisco) {
		t.Skip("cisco 已有注册实现，本用例前提不成立")
	}
	tr, err := GetTranslator(VendorCisco)
	if err == nil || tr != nil {
		t.Fatal("未注册厂商应返回明确错误")
	}
}

// TE-01/R09: RegisterTranslator 与读路径并发调用无数据竞态（-race 锁定）。
func TestVendorRegistry_ConcurrentRegisterAndLookup(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			RegisterTranslator(VendorHuawei, NewHuaweiTranslator())
		}()
		go func() {
			defer wg.Done()
			_ = IsVendorSupported(VendorHuawei)
			_, _ = GetTranslator(VendorHuawei)
			_ = SupportedVendors()
		}()
	}
	wg.Wait()
}
