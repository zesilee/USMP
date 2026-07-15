package intent

import (
	"context"
	"log"

	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
)

// TxResult is the per-device outcome of a cross-device push (BIO-03).
type TxResult struct {
	Device string
	// Err is non-nil when the device did not end up with the config applied.
	Err error
	// NonTransactional marks a device that lacked :confirmed-commit and was
	// pushed with a plain commit (DP-08 降级，呈现为非事务下发).
	NonTransactional bool
}

// Pusher executes the cross-device push for an expansion. Implemented by
// TxCoordinator; faked in unit tests.
type Pusher interface {
	Push(ctx context.Context, frags []Fragment) map[string]TxResult
}

// writeDesired merges the expansion fragments into the desired ConfigStore and
// triggers native reconciliation — called ONLY after the transaction succeeded
// (a failed transaction must not leak desired, or periodic reconcile would
// bypass 2PC and push it anyway, BIO-03).
func writeDesired(cs reconcile.ConfigStore, trigger func(deviceID, path string) bool, frags []Fragment) {
	for _, f := range frags {
		existing, _ := cs.Get(f.Device, f.Path)
		var merged interface{} = f.Config
		if existing != nil {
			merged = mergeFragment(existing, f.Config)
		}
		if err := cs.Set(f.Device, f.Path, merged); err != nil {
			log.Printf("intent: write desired %s %s: %v", f.Device, f.Path, err)
			continue
		}
		if trigger != nil {
			trigger(f.Device, f.Path)
		}
	}
}

// mergeFragment unions an intent fragment into the existing desired config so
// intent-managed entries never clobber manually-configured siblings（合并防
// 抹除，矩阵 A3 / VLAN 交付教训）. Same-key entries: intent leaves overlay the
// existing entry (manual fields survive). Unknown existing types fall back to
// the fragment (logged, R08).
func mergeFragment(existing, frag interface{}) interface{} {
	switch f := frag.(type) {
	case *huawei.HuaweiVlan_Vlan_Vlans:
		e, ok := existing.(*huawei.HuaweiVlan_Vlan_Vlans)
		if !ok || e == nil {
			logMergeFallback(existing, frag)
			return frag
		}
		out := &huawei.HuaweiVlan_Vlan_Vlans{Vlan: map[uint16]*huawei.HuaweiVlan_Vlan_Vlans_Vlan{}}
		for id, v := range e.Vlan {
			out.Vlan[id] = v
		}
		for id, fv := range f.Vlan {
			if ev, ok := out.Vlan[id]; ok && ev != nil {
				merged := *ev
				if fv.Id != nil {
					merged.Id = fv.Id
				}
				if fv.Name != nil {
					merged.Name = fv.Name
				}
				out.Vlan[id] = &merged
			} else {
				out.Vlan[id] = fv
			}
		}
		return out
	case *huawei.HuaweiIfm_Ifm_Interfaces:
		e, ok := existing.(*huawei.HuaweiIfm_Ifm_Interfaces)
		if !ok || e == nil {
			logMergeFallback(existing, frag)
			return frag
		}
		out := &huawei.HuaweiIfm_Ifm_Interfaces{Interface: map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface{}}
		for name, i := range e.Interface {
			out.Interface[name] = i
		}
		for name, fi := range f.Interface {
			ei, ok := out.Interface[name]
			if !ok || ei == nil {
				out.Interface[name] = fi
				continue
			}
			merged := *ei
			// 仅覆盖意图管理的 L2 链条，保留手工字段（mtu/描述等）。
			if fi.Ethernet != nil {
				merged.Ethernet = fi.Ethernet
			}
			if fi.Name != nil {
				merged.Name = fi.Name
			}
			out.Interface[name] = &merged
		}
		return out
	default:
		logMergeFallback(existing, frag)
		return frag
	}
}

func logMergeFallback(existing, frag interface{}) {
	log.Printf("intent: mergeFragment fallback to fragment (existing %T, frag %T)", existing, frag)
}
