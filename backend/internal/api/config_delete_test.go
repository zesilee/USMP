package api

import (
	"sync"
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/internal/cache"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/openconfig/ygot/ygot"
)

const (
	delVlanPath = "/vlan:vlan/vlan:vlans"
	delIfmPath  = "/ifm:ifm/ifm:interfaces"
)

// BR-09：per-model key 解析——构造仅含 key 的单条目模型对象（供删除编码与 desired 移除）。
func TestParseDeleteTarget(t *testing.T) {
	t.Run("vlan 合法键", func(t *testing.T) {
		v, err := parseDeleteTarget(delVlanPath, "10")
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		vlans, ok := v.(*huawei.HuaweiVlan_Vlan_Vlans)
		if !ok || vlans.Vlan[10] == nil || vlans.Vlan[10].Id == nil || *vlans.Vlan[10].Id != 10 {
			t.Fatalf("parsed = %#v, want keyed vlan 10", v)
		}
	})
	t.Run("ifm 合法键（含斜杠接口名）", func(t *testing.T) {
		v, err := parseDeleteTarget(delIfmPath, "GigabitEthernet0/0/1")
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		ifaces, ok := v.(*huawei.HuaweiIfm_Ifm_Interfaces)
		if !ok || ifaces.Interface["GigabitEthernet0/0/1"] == nil {
			t.Fatalf("parsed = %#v, want keyed interface", v)
		}
	})
	cases := []struct {
		desc, path, key string
	}{
		{"vlan 非整数键", delVlanPath, "abc"},
		{"vlan 超范围键(0)", delVlanPath, "0"},
		{"vlan 超范围键(5000)", delVlanPath, "5000"},
		{"空键", delVlanPath, ""},
		{"ifm 空键", delIfmPath, ""},
		{"未知路径", "/route:route/tables", "1"},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			if _, err := parseDeleteTarget(c.path, c.key); err == nil {
				t.Errorf("err = nil, want error（不得触达设备）")
			}
		})
	}
}

func newDeleteTestStore() *manager.InMemoryConfigStore {
	return manager.NewInMemoryConfigStore(cache.NewTTLLRUCache(1000, 30*time.Minute, 0))
}

// BR-09：desired 键移除——per-model、幂等、不原地改（并发读安全，R09）。
func TestStoreConfigDeletedVlan(t *testing.T) {
	cs := newDeleteTestStore()
	seed := &huawei.HuaweiVlan_Vlan_Vlans{Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
		10: {Id: ygot.Uint16(10), Name: ygot.String("ten")},
		20: {Id: ygot.Uint16(20)},
	}}
	if err := cs.Set("1.1.1.1", delVlanPath, seed); err != nil {
		t.Fatal(err)
	}

	target, _ := parseDeleteTarget(delVlanPath, "10")
	if err := storeConfigDeleted(cs, "1.1.1.1", delVlanPath, target); err != nil {
		t.Fatalf("storeConfigDeleted: %v", err)
	}
	got, _ := cs.Get("1.1.1.1", delVlanPath)
	vlans := got.(*huawei.HuaweiVlan_Vlan_Vlans)
	if _, exists := vlans.Vlan[10]; exists {
		t.Error("vlan 10 still in desired")
	}
	if _, exists := vlans.Vlan[20]; !exists {
		t.Error("vlan 20 lost（不得误删同表其它键）")
	}
	// 不原地改：seed 对象保持原样（并发读旧快照安全）
	if _, exists := seed.Vlan[10]; !exists {
		t.Error("seed mutated in place（R09 违例）")
	}

	// 幂等：再次删除同键 no-op 不报错
	if err := storeConfigDeleted(cs, "1.1.1.1", delVlanPath, target); err != nil {
		t.Errorf("idempotent delete err = %v", err)
	}
	// desired 无该表时 no-op 不报错
	if err := storeConfigDeleted(cs, "9.9.9.9", delVlanPath, target); err != nil {
		t.Errorf("no-desired delete err = %v", err)
	}
}

func TestStoreConfigDeletedIfm(t *testing.T) {
	cs := newDeleteTestStore()
	seed := &huawei.HuaweiIfm_Ifm_Interfaces{Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{
		"GE0/0/1": {Name: ygot.String("GE0/0/1")},
		"GE0/0/2": {Name: ygot.String("GE0/0/2")},
	}}
	if err := cs.Set("1.1.1.1", delIfmPath, seed); err != nil {
		t.Fatal(err)
	}
	target, err := parseDeleteTarget(delIfmPath, "GE0/0/1")
	if err != nil {
		t.Fatal(err)
	}
	if err := storeConfigDeleted(cs, "1.1.1.1", delIfmPath, target); err != nil {
		t.Fatalf("storeConfigDeleted: %v", err)
	}
	got, _ := cs.Get("1.1.1.1", delIfmPath)
	ifaces := got.(*huawei.HuaweiIfm_Ifm_Interfaces)
	if _, exists := ifaces.Interface["GE0/0/1"]; exists {
		t.Error("GE0/0/1 still in desired")
	}
	if _, exists := ifaces.Interface["GE0/0/2"]; !exists {
		t.Error("GE0/0/2 lost")
	}
}

// R09：删除与合并写并发交错——同临界区串行化，无丢更新无竞态（-race）。
func TestStoreConfigDeletedConcurrentWithMerge(t *testing.T) {
	cs := newDeleteTestStore()
	var wg sync.WaitGroup
	for i := 1; i <= 8; i++ {
		wg.Add(2)
		id := uint16(i)
		go func() {
			defer wg.Done()
			inc := &huawei.HuaweiVlan_Vlan_Vlans{Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
				id: {Id: ygot.Uint16(id)},
			}}
			_ = storeConfigMerged(cs, "1.1.1.1", delVlanPath, inc)
		}()
		go func() {
			defer wg.Done()
			target := &huawei.HuaweiVlan_Vlan_Vlans{Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
				id + 100: {Id: ygot.Uint16(id + 100)},
			}}
			_ = storeConfigDeleted(cs, "1.1.1.1", delVlanPath, target)
		}()
	}
	wg.Wait()
	got, _ := cs.Get("1.1.1.1", delVlanPath)
	vlans, ok := got.(*huawei.HuaweiVlan_Vlan_Vlans)
	if !ok {
		t.Fatalf("stored type %T", got)
	}
	// 8 个合并键全部存活（删除只动 100+ 键，无丢更新）
	for i := uint16(1); i <= 8; i++ {
		if _, exists := vlans.Vlan[i]; !exists {
			t.Errorf("vlan %d lost under concurrency", i)
		}
	}
}
