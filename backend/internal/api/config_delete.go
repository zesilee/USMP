package api

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
	"github.com/openconfig/ygot/ygot"
)

// parseDeleteTarget 按 path 路由模型分支，把行主键解析成「仅含 key 叶的单条目模型
// 对象」（BR-09）：同一对象既作删除编码目标（DP-07 marshalDeleteChange）也作 desired
// 键移除依据。路径匹配与 convertConfig/ygotRegistry 同规（关键字包含式）。
// 非法 key / 未知路径返回错误——调用方 400，不得触达设备。
func parseDeleteTarget(path, key string) (interface{}, error) {
	if key == "" {
		return nil, fmt.Errorf("missing entry key")
	}
	switch {
	case strings.Contains(path, "ifm:ifm") && strings.Contains(path, "interfaces"):
		return &huawei.HuaweiIfm_Ifm_Interfaces{
			Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{
				key: {Name: ygot.String(key)},
			},
		}, nil
	case strings.Contains(path, "vlan:") && strings.Contains(path, "vlan"):
		id, err := strconv.ParseUint(key, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("vlan key %q is not an integer: %w", key, err)
		}
		// 与 validateConfig 同域约束：1-4094（0/4095+ 为保留/非法）。
		if id < 1 || id > 4094 {
			return nil, fmt.Errorf("VLAN ID %d 超出有效范围 [1, 4094]", id)
		}
		return &huawei.HuaweiVlan_Vlan_Vlans{
			Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
				uint16(id): {Id: ygot.Uint16(uint16(id))},
			},
		}, nil
	}
	return nil, fmt.Errorf("path %q has no delete-capable model branch", path)
}

// deleteGate 按模型门禁校验删除请求（BR-10）：目标节点（或其包裹容器的单 list 子节点）
// operation-exclude 含 delete、或节点 readonly（config false）→ 拒绝，与前端按钮门禁
// 互为防御。schema 查不到该路径 → 放行（降级，R08——门禁失效不应封死合法删除，
// 模型分支路由与设备是最终权威）。运行时路径按段剥模块前缀映射 schema 路径
// （/vlan:vlan/vlan:vlans → /vlan/vlans，与前端 configPathFor 互逆）。
func deleteGate(s schema.Schema, runtimePath string) error {
	if s == nil {
		return nil
	}
	segs := strings.Split(strings.Trim(runtimePath, "/"), "/")
	for i, seg := range segs {
		if j := strings.Index(seg, ":"); j >= 0 {
			segs[i] = seg[j+1:]
		}
	}
	node, ok := s.Path("/" + strings.Join(segs, "/"))
	if !ok {
		return nil // schema 未覆盖：放行
	}
	if node.ReadOnly() {
		return fmt.Errorf("路径 %s 为设备状态数据（config false），不可删除", runtimePath)
	}
	excluded := func(n schema.Node) bool {
		switch t := n.(type) {
		case schema.ListNode:
			for _, op := range t.OperationExcludes() {
				if op == "delete" {
					return true
				}
			}
		case schema.ContainerNode:
			for _, op := range t.OperationExcludes() {
				if op == "delete" {
					return true
				}
			}
		}
		return false
	}
	if excluded(node) {
		return fmt.Errorf("模型禁止删除该节点（operation-exclude 含 delete）：%s", runtimePath)
	}
	// 包裹容器路径（group 裹单 list 的常见形态）：取其唯一 list 子节点一并判定。
	if c, ok := node.(schema.ContainerNode); ok {
		var lists []schema.Node
		for _, ch := range c.Children() {
			if _, isList := ch.(schema.ListNode); isList {
				lists = append(lists, ch)
			}
		}
		if len(lists) == 1 {
			if lists[0].ReadOnly() {
				return fmt.Errorf("路径 %s 为设备状态数据（config false），不可删除", runtimePath)
			}
			if excluded(lists[0]) {
				return fmt.Errorf("模型禁止删除该节点（operation-exclude 含 delete）：%s", runtimePath)
			}
		}
	}
	return nil
}

// storeConfigDeleted 从已存 desired 中移除 target 携带的键（BR-09）：与合并写共用
// configMergeMu 临界区（防丢更新，R09），构造新对象不原地改（并发读旧快照安全，与
// mergeConfig 同规）。desired 不存在或键不存在时为幂等 no-op——删除意图以设备为准。
func storeConfigDeleted(cs reconcile.ConfigStore, ip, path string, target interface{}) error {
	configMergeMu.Lock()
	defer configMergeMu.Unlock()

	existing, err := cs.Get(ip, path)
	if err != nil || existing == nil {
		return nil // 无 desired：no-op
	}

	switch tgt := target.(type) {
	case *huawei.HuaweiVlan_Vlan_Vlans:
		ex, ok := existing.(*huawei.HuaweiVlan_Vlan_Vlans)
		if !ok || ex == nil {
			return nil
		}
		next := &huawei.HuaweiVlan_Vlan_Vlans{Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{}}
		for k, v := range ex.Vlan {
			if _, drop := tgt.Vlan[k]; drop {
				continue
			}
			next.Vlan[k] = v
		}
		return cs.Set(ip, path, next)
	case *huawei.HuaweiIfm_Ifm_Interfaces:
		ex, ok := existing.(*huawei.HuaweiIfm_Ifm_Interfaces)
		if !ok || ex == nil {
			return nil
		}
		next := &huawei.HuaweiIfm_Ifm_Interfaces{Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{}}
		for k, v := range ex.Interface {
			if _, drop := tgt.Interface[k]; drop {
				continue
			}
			next.Interface[k] = v
		}
		return cs.Set(ip, path, next)
	}
	return fmt.Errorf("unsupported delete target %T", target)
}

// summarizeDeleted 生成删除审计摘要：模型键列表（诚实、简短）。
func summarizeDeleted(target interface{}) string {
	switch t := target.(type) {
	case *huawei.HuaweiVlan_Vlan_Vlans:
		keys := make([]string, 0, len(t.Vlan))
		for k := range t.Vlan {
			keys = append(keys, strconv.Itoa(int(k)))
		}
		sort.Strings(keys)
		return "delete vlan " + strings.Join(keys, ",")
	case *huawei.HuaweiIfm_Ifm_Interfaces:
		keys := make([]string, 0, len(t.Interface))
		for k := range t.Interface {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return "delete interface " + strings.Join(keys, ",")
	}
	return "delete (unknown)"
}
