package intent

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/openconfig/ygot/ygot"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
)

// Cleaner removes intent-claimed config from devices（BIO-05 finalizer 删除 /
// BIO-06 收缩差集共用的 DELETE 命令通道）。Implemented by TxCoordinator; faked
// in unit tests. Returns per-device errors (empty map = all clean).
type Cleaner interface {
	Cleanup(ctx context.Context, claims []Claim) map[string]error
}

// claimKeyRe extracts the entry key from a claim path
// (…/vlan[id=100] / …/interface[name=GE0/0/1]).
var claimKeyRe = regexp.MustCompile(`\[(id|name)=([^\]]+)\]$`)

// claimKey returns the key kind ("id"/"name") and value of a claim path.
func claimKey(path string) (kind, value string, ok bool) {
	m := claimKeyRe.FindStringSubmatch(path)
	if m == nil {
		return "", "", false
	}
	return m[1], m[2], true
}

// Cleanup implements Cleaner on the TxCoordinator: per device it issues the
// DELETE command channel pushes for every claim (command semantics like BR-09,
// non-transactional — deletion retries are idempotent).
func (t *TxCoordinator) Cleanup(ctx context.Context, claims []Claim) map[string]error {
	byDev := map[string][]Claim{}
	var devs []string
	for _, c := range claims {
		if _, ok := byDev[c.Device]; !ok {
			devs = append(devs, c.Device)
		}
		byDev[c.Device] = append(byDev[c.Device], c)
	}
	sort.Strings(devs)

	unlock := t.lockAll(devs)
	defer unlock()

	failures := map[string]error{}
	for _, d := range devs {
		changes, err := cleanupChanges(byDev[d])
		if err != nil {
			failures[d] = err
			continue
		}
		if len(changes) == 0 {
			continue
		}
		c, err := t.pool.Get(t.resolveConn(d))
		if err != nil {
			failures[d] = fmt.Errorf("connect: %w", err)
			continue
		}
		res, err := c.Set(ctx, changes, client.WithCommit(true))
		if err == nil {
			continue
		}
		// 幂等清理：keyed delete 命中已不存在的条目返回 data-missing（RFC 6241）
		// ——重试路径的预期形态，视为已清理；其余错误如实上报。
		if res != nil && onlyDataMissing(res) {
			continue
		}
		failures[d] = err
	}
	return failures
}

// onlyDataMissing reports whether every failed change in res is a DELETE that
// hit data-missing (already clean).
func onlyDataMissing(res *client.SetResult) bool {
	sawFailure := false
	for _, cr := range res.Changes {
		if cr.Success {
			continue
		}
		sawFailure = true
		if cr.Change.Type != client.DeleteChange || cr.Error == nil ||
			!strings.Contains(cr.Error.Error(), "data-missing") {
			return false
		}
	}
	return sawFailure
}

// cleanupChanges builds the per-device DELETE command changes for claims:
//   - vlan 条目：keyed delete（DP-07 marshalDeleteChange 通道）；
//   - ifm 端口：删 l2-attribute 子树（raw XML operation="remove"）——真机内置
//     接口条目删不掉（PR#145 教训），意图只认领了 L2 链，清理也只清 L2 链。
func cleanupChanges(claims []Claim) ([]client.Change, error) {
	var out []client.Change
	for _, c := range claims {
		kind, value, ok := claimKey(c.Path)
		if !ok {
			return nil, fmt.Errorf("claim path %q has no parsable key", c.Path)
		}
		switch {
		case c.Module == "vlan" && kind == "id":
			id, err := strconv.ParseUint(value, 10, 16)
			if err != nil {
				return nil, fmt.Errorf("claim %q: vlan id %q: %w", c.Path, value, err)
			}
			out = append(out, client.Change{
				Type: client.DeleteChange,
				Path: VlanPath,
				OldValue: &huawei.HuaweiVlan_Vlan_Vlans{
					Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{
						uint16(id): {Id: ygot.Uint16(uint16(id))},
					},
				},
			})
		case c.Module == "ifm" && kind == "name":
			out = append(out, client.Change{
				Type:     client.AddChange,
				Path:     IfmPath,
				NewValue: l2RemoveXML(value),
			})
		default:
			return nil, fmt.Errorf("claim %q: no cleanup channel for module %s key %s", c.Path, c.Module, kind)
		}
	}
	return out, nil
}

// l2RemoveXML builds the raw edit-config payload that removes the intent-owned
// l2-attribute subtree of one interface. Namespaces mirror the push encoder
// exactly (ethernet chain inherits the huawei-ifm namespace) so the device/sim
// key-matches the same stored nodes; operation="remove" keeps retries
// idempotent (no data-missing on the second pass).
func l2RemoveXML(port string) string {
	return fmt.Sprintf(`<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm"><interfaces><interface><name>%s</name><ethernet><main-interface><l2-attribute xmlns:nc="urn:ietf:params:xml:ns:netconf:base:1.0" nc:operation="remove"/></main-interface></ethernet></interface></interfaces></ifm>`, xmlEscape(port))
}

func xmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

// removeClaimsFromDesired scrubs intent-claimed entries out of the desired
// ConfigStore（必须先于/伴随设备清理：否则周期对账会按 desired 把刚删的配置
// 推回去）。Missing/mismatched desired entries are skipped (R08).
func removeClaimsFromDesired(cs reconcile.ConfigStore, claims []Claim) {
	if cs == nil {
		return
	}
	for _, c := range claims {
		kind, value, ok := claimKey(c.Path)
		if !ok {
			continue
		}
		switch {
		case c.Module == "vlan" && kind == "id":
			existing, _ := cs.Get(c.Device, VlanPath)
			e, ok := existing.(*huawei.HuaweiVlan_Vlan_Vlans)
			if !ok || e == nil {
				continue
			}
			id, err := strconv.ParseUint(value, 10, 16)
			if err != nil {
				continue
			}
			out := &huawei.HuaweiVlan_Vlan_Vlans{Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{}}
			for k, v := range e.Vlan {
				if k != uint16(id) {
					out.Vlan[k] = v
				}
			}
			_ = cs.Set(c.Device, VlanPath, out)
		case c.Module == "ifm" && kind == "name":
			existing, _ := cs.Get(c.Device, IfmPath)
			e, ok := existing.(*huawei.HuaweiIfm_Ifm_Interfaces)
			if !ok || e == nil {
				continue
			}
			out := &huawei.HuaweiIfm_Ifm_Interfaces{Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{}}
			for k, v := range e.Interface {
				if k != value {
					out.Interface[k] = v
					continue
				}
				if v == nil {
					continue
				}
				stripped := *v
				stripped.Ethernet = nil
				// 意图只认领 L2 链：剥掉后条目仅剩 key 则整条移除，否则保留手工字段。
				if stripped.Name != nil && onlyName(&stripped) {
					continue
				}
				out.Interface[k] = &stripped
			}
			_ = cs.Set(c.Device, IfmPath, out)
		}
	}
}

// onlyName reports whether the interface entry carries nothing beyond its key.
func onlyName(i *huawei.HuaweiIfm_Ifm_Interfaces_Interface) bool {
	probe := *i
	probe.Name = nil
	empty := huawei.HuaweiIfm_Ifm_Interfaces_Interface{}
	return fmt.Sprintf("%+v", probe) == fmt.Sprintf("%+v", empty)
}
