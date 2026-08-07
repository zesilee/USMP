package api

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	beecontext "github.com/beego/beego/v2/server/web/context"
	"github.com/openconfig/ygot/ygot"

	"github.com/leezesi/usmp/backend/internal/intent"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/audit"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/diff"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/driver"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/xmlcodec"
)

// ChangesetHandler 承载攒批变更集的服务端契约（config-changeset）：
// 试运行预览（CS-01/02/03/05，纯计算不下发）与批量原子提交（CS-04）。
type ChangesetHandler struct {
	manager manager.Manager
	// fetch 与 ConfigHandler 同源的设备读闭包（可注入测试）。
	fetch func(ctx context.Context, ip, path string) (interface{}, error)
	// push 执行 candidate 两阶段原子下发（生产为 intent.TxCoordinator，
	// 可注入测试）。
	push intent.Pusher
	// support 节点级不支持集视图（BR-12 写门禁，与 ConfigHandler 同口径）。
	support func(ip string) nodeSupportView
}

// NewChangesetHandler 构造变更集 handler：设备读闭包与 ConfigHandler 同实现，
// 下发通道复用意图侧 2PC 协调器（共享 ClientPool/DeviceStore 与设备锁）。
func NewChangesetHandler(mgr manager.Manager) *ChangesetHandler {
	cfg := NewConfigHandler(mgr)
	return &ChangesetHandler{
		manager: mgr,
		fetch:   cfg.fetch,
		push:    intent.NewTxCoordinator(mgr.GetClientPool(), mgr.GetDeviceStore(), 0),
		support: cfg.support,
	}
}

// ChangesetEntryReq 是变更集单条目（前端 changeset store 的序列化形态）。
type ChangesetEntryReq struct {
	Op      string                 `json:"op"`      // create | update | delete
	Path    string                 `json:"path"`    // 请求路径（锚点或其子路径）
	Payload map[string]interface{} `json:"payload"` // create/update：以 path 为根的 RFC7951 子树
	Key     string                 `json:"key"`     // delete：list 主键值
	Cleared []string               `json:"cleared"` // update：字段级清除的叶名（CS-05）
}

// ChangesetReq 是 preview/commit 的请求体（单设备变更集）。
type ChangesetReq struct {
	Device  string              `json:"device"`
	Entries []ChangesetEntryReq `json:"entries"`
}

// DiffChangeDTO 是结构化 diff 树的单条变更。
type DiffChangeDTO struct {
	Type string      `json:"type"` // ADD | DELETE | MODIFY
	Path string      `json:"path"`
	Old  interface{} `json:"old,omitempty"`
	New  interface{} `json:"new,omitempty"`
}

// ChangesetSummary 聚合计数（对齐 NCE 图例：增加/修改/删除）。
type ChangesetSummary struct {
	Adds     int `json:"adds"`
	Deletes  int `json:"deletes"`
	Modifies int `json:"modifies"`
	Total    int `json:"total"`
}

// ChangesetPreviewEntry 是单条目的预览结果。
type ChangesetPreviewEntry struct {
	Op             string `json:"op"`
	Path           string `json:"path"`
	BaselineSource string `json:"baseline_source"` // desired | cache | device | none
	// ForwardXML/RollbackXML 为 edit-config 片段；无 XML 通道时为空且
	// Unsupported=true（CS-03 如实降级，绝不伪造报文）。
	ForwardXML        string          `json:"forward_xml,omitempty"`
	RollbackXML       string          `json:"rollback_xml,omitempty"`
	Unsupported       bool            `json:"unsupported,omitempty"`
	UnsupportedReason string          `json:"unsupported_reason,omitempty"`
	Diff              []DiffChangeDTO `json:"diff"`
}

// ChangesetPreviewData 是 POST /config/changeset/preview 的 data 负载。
type ChangesetPreviewData struct {
	Device  string                  `json:"device"`
	Entries []ChangesetPreviewEntry `json:"entries"`
	Summary ChangesetSummary        `json:"summary"`
}

// previewEntry 是解码后的条目工作单（decode 阶段产出，全部条目解码成功才计算）。
type previewEntry struct {
	req    ChangesetEntryReq
	target interface{} // 类型化目标（create/update=payload 解码；delete=键定位条目）
	anchor string
	desc   driver.Descriptor
}

// Preview 试运行预览（CS-01）：纯计算，不碰设备写通道、desired、缓存与审计。
//
// @Summary  变更集试运行预览（正向/回滚报文 + diff 树，不下发）
// @Tags     config
// @Accept   json
// @Produce  json
// @Param    changeset body ChangesetReq true "单设备变更集"
// @Success  200 {object} Response{data=ChangesetPreviewData} "预览结果"
// @Failure  400 {object} Response "变更集解析失败"
// @Router   /config/changeset/preview [post]
func (h *ChangesetHandler) Preview(c *beecontext.Context) {
	req, entries, ok := h.decodeChangeset(c)
	if !ok {
		return
	}

	data := ChangesetPreviewData{Device: req.Device, Entries: make([]ChangesetPreviewEntry, 0, len(entries))}
	engine := diff.NewDefaultDiffEngine()
	// 同锚点基线在单次请求内只取一次（多条目常落同一 list，避免重复实时回读）。
	memo := map[string]baselineResult{}
	for _, pe := range entries {
		if _, ok := memo[pe.anchor]; !ok {
			b, src := h.baseline(req.Device, pe)
			memo[pe.anchor] = baselineResult{value: b, source: src}
		}
		res := h.previewOne(pe, memo[pe.anchor], engine)
		for _, ch := range res.Diff {
			switch ch.Type {
			case "ADD":
				data.Summary.Adds++
			case "DELETE":
				data.Summary.Deletes++
			case "MODIFY":
				data.Summary.Modifies++
			}
			data.Summary.Total++
		}
		data.Entries = append(data.Entries, res)
	}
	Success(c, data, "Changeset preview computed")
}

// decodeChangeset 解析并解码整个变更集；任一条目非法即 400，不返回部分结果
// （CS-01 负路径）。
func (h *ChangesetHandler) decodeChangeset(c *beecontext.Context) (ChangesetReq, []previewEntry, bool) {
	var req ChangesetReq
	if err := bindJSON(c, &req); err != nil {
		Error(c, 400, "Invalid request: "+err.Error())
		return req, nil, false
	}
	if req.Device == "" {
		Error(c, 400, "缺少 device")
		return req, nil, false
	}
	if len(req.Entries) == 0 {
		Error(c, 400, "变更集为空")
		return req, nil, false
	}
	out := make([]previewEntry, 0, len(req.Entries))
	for i, e := range req.Entries {
		pe, err := decodeEntry(e)
		if err != nil {
			Error(c, 400, fmt.Sprintf("条目 %d (%s %s): %s", i, e.Op, e.Path, err.Error()))
			return req, nil, false
		}
		out = append(out, pe)
	}
	return req, out, true
}

// decodeEntry 把单条目解码为类型化工作单（op 校验 + RFC7951 解码 + 域约束）。
func decodeEntry(e ChangesetEntryReq) (previewEntry, error) {
	pe := previewEntry{req: e}
	switch e.Op {
	case "create", "update":
		if len(e.Payload) == 0 {
			return pe, fmt.Errorf("缺少 payload")
		}
		target, anchor, err := convertConfigAnchored(e.Path, e.Payload)
		if err != nil {
			return pe, err
		}
		if err := validateConfig(target); err != nil {
			return pe, fmt.Errorf("配置校验失败: %w", err)
		}
		pe.target, pe.anchor = target, anchor
	case "delete":
		if e.Key == "" {
			return pe, fmt.Errorf("delete 条目缺少 key")
		}
		target, err := parseDeleteTarget(e.Path, e.Key)
		if err != nil {
			return pe, err
		}
		d, ok := driver.EncoderFor(e.Path)
		if !ok {
			return pe, fmt.Errorf("路径 %q 未注册编码驱动", e.Path)
		}
		pe.target, pe.anchor = target, d.EncodeAnchor
	default:
		return pe, fmt.Errorf("未知 op %q（仅支持 create/update/delete）", e.Op)
	}
	d, ok := driver.EncoderFor(pe.anchor)
	if !ok {
		return pe, fmt.Errorf("锚点 %q 未注册编码驱动", pe.anchor)
	}
	pe.desc = d
	return pe, nil
}

// baselineResult 是单锚点的基线取值与来源（请求内 memo）。
type baselineResult struct {
	value  interface{}
	source string
}

// previewOne 计算单条目的预览：基线→diff→正向/回滚报文。
func (h *ChangesetHandler) previewOne(pe previewEntry, base baselineResult, engine *diff.DefaultDiffEngine) ChangesetPreviewEntry {
	res := ChangesetPreviewEntry{Op: pe.req.Op, Path: pe.req.Path, Diff: []DiffChangeDTO{}}

	res.BaselineSource = base.source
	baseSubset, baseCount := containerSubset(base.value, pe.target)

	// diff 树（引擎为反射比对，纯函数；delete 与 cleared 叶手工补齐——
	// 声明式 subset diff 刻意表达不了删除，见 config-delete-semantics）。
	switch pe.req.Op {
	case "delete":
		res.Diff = appendEntryDeleteDiff(res.Diff, pe, baseSubset, baseCount)
	default:
		if r, err := engine.Diff(pe.target, baseSubset, nil); err == nil && r != nil {
			for _, ch := range r.Changes {
				res.Diff = append(res.Diff, DiffChangeDTO{Type: ch.Type.String(), Path: ch.Path, Old: ch.OldValue, New: ch.NewValue})
			}
		}
		res.Diff = appendClearedDiff(res.Diff, pe, baseSubset)
	}

	// 报文预览：无 XML 通道如实降级（CS-03）。
	if pe.desc.XML == nil {
		res.Unsupported = true
		res.UnsupportedReason = fmt.Sprintf("模块 %s 无 XML 编码通道，不支持报文预览", pe.desc.Module)
		return res
	}
	fw, rb, err := h.encodePreviewXML(pe, baseSubset, baseCount)
	if err != nil {
		res.Unsupported = true
		res.UnsupportedReason = err.Error()
		return res
	}
	res.ForwardXML, res.RollbackXML = fw, rb
	return res
}

// baseline 取控制器目标态基线（CS-01 基线链）：desired → running cache →
// 实时回读；全部缺失返回 (nil, "none")（R08 降级，diff 视为全新增）。
func (h *ChangesetHandler) baseline(device string, pe previewEntry) (interface{}, string) {
	if v, err := h.manager.GetConfigStore().Get(device, pe.anchor); err == nil && v != nil {
		return v, "desired"
	}
	if raw, _, ok := h.manager.GetRunningCache().GetWithAge(runKey(device, pe.anchor)); ok {
		if conv := h.convertBaseline(pe.anchor, raw); conv != nil {
			return conv, "cache"
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if raw, err := h.fetch(ctx, device, pe.anchor); err == nil {
		if conv := h.convertBaseline(pe.anchor, raw); conv != nil {
			return conv, "device"
		}
	}
	return nil, "none"
}

// convertBaseline 把回读形态（RFC7951 map）转为类型化 GoStruct；非 map 或
// 解码失败返回 nil（上层降级到下一基线来源）。
func (h *ChangesetHandler) convertBaseline(anchor string, raw interface{}) interface{} {
	m, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}
	conv, _, err := convertConfigAnchored(anchor, m)
	if err != nil {
		return nil
	}
	return conv
}

// encodePreviewXML 生成单条目的正向与回滚报文（CS-01/02/05）。
func (h *ChangesetHandler) encodePreviewXML(pe previewEntry, baseSubset interface{}, baseCount int) (string, string, error) {
	gs, ok := pe.target.(ygot.GoStruct)
	if !ok {
		return "", "", fmt.Errorf("目标类型 %T 不是 GoStruct", pe.target)
	}
	switch pe.req.Op {
	case "delete":
		fw, err := client.EncodeChangeXML(client.Change{Type: client.DeleteChange, OldValue: pe.target})
		if err != nil {
			return "", "", err
		}
		rb := ""
		if baseCount > 0 {
			if rb, err = client.EncodeChangeXML(client.Change{Type: client.AddChange, NewValue: baseSubset}); err != nil {
				return "", "", err
			}
		}
		return fw, rb, nil
	default:
		fw, err := client.EncodeChangeXML(client.Change{Type: client.AddChange, NewValue: pe.target})
		if err != nil {
			return "", "", err
		}
		if len(pe.req.Cleared) > 0 {
			leafDel, lerr := encodeClearedLeaves(pe.desc, gs, pe.req.Cleared)
			if lerr != nil {
				return "", "", lerr
			}
			fw += "\n" + leafDel
		}
		var rb string
		if baseCount > 0 {
			// 基线有该条目：回滚 = 按基线值重建（含被改叶与被清叶的旧值）。
			if rb, err = client.EncodeChangeXML(client.Change{Type: client.AddChange, NewValue: baseSubset}); err != nil {
				return "", "", err
			}
		} else {
			// 基线无该条目（新增）：回滚 = 键定位删除。
			if rb, err = client.EncodeChangeXML(client.Change{Type: client.DeleteChange, OldValue: pe.target}); err != nil {
				return "", "", err
			}
		}
		return fw, rb, nil
	}
}

// encodeClearedLeaves 经 xmlcodec 叶级删除通道生成清除叶报文（CS-05）。
func encodeClearedLeaves(d driver.Descriptor, gs ygot.GoStruct, leaves []string) (string, error) {
	wrapped, err := d.WrapXMLValue(gs)
	if err != nil {
		return "", err
	}
	return xmlcodec.EncodeLeafDelete(d.XML, wrapped, leaves)
}

// appendEntryDeleteDiff 为 delete 条目补 DELETE 变更：基线有值时 Old 取基线
// 条目，否则仅键定位（旧值未知，如实不填）。
func appendEntryDeleteDiff(dst []DiffChangeDTO, pe previewEntry, baseSubset interface{}, baseCount int) []DiffChangeDTO {
	old := pe.target
	if baseCount > 0 {
		old = baseSubset
	}
	return append(dst, DiffChangeDTO{Type: "DELETE", Path: pe.anchor + "[key=" + pe.req.Key + "]", Old: flattenForDTO(old)})
}

// appendClearedDiff 为 cleared 叶补 DELETE 变更（Old=基线值；基线无值则该叶
// 本次本就不存在，跳过——与前端「基线无值清除仅置空」语义一致）。
func appendClearedDiff(dst []DiffChangeDTO, pe previewEntry, baseSubset interface{}) []DiffChangeDTO {
	if len(pe.req.Cleared) == 0 || baseSubset == nil {
		return dst
	}
	entries := listEntries(baseSubset)
	for _, leaf := range pe.req.Cleared {
		for key, ev := range entries {
			if val, ok := leafValueByTag(ev, leaf); ok {
				dst = append(dst, DiffChangeDTO{Type: "DELETE", Path: fmt.Sprintf("%s[key=%v]/%s", pe.anchor, key, leaf), Old: val})
			}
		}
	}
	return dst
}

// ChangesetCommitData 是 POST /config/changeset/commit 的 data 负载。
type ChangesetCommitData struct {
	Status  string `json:"status"` // COMMITTED
	Device  string `json:"device"`
	Entries int    `json:"entries"`
	// NonTransactional 标记设备缺 :confirmed-commit 降级普通 commit（DP-08）。
	NonTransactional bool          `json:"non_transactional,omitempty"`
	Reconciliation   ReconcileInfo `json:"reconciliation"`
}

// Commit 批量原子提交（CS-04）：单设备变更集经 candidate 两阶段整体下发，
// 任一失败设备回到提交前状态；desired/缓存/审计仅在设备 commit 成功后落地
// （先写 desired 会让周期对账绕过 2PC 把失败变更重新推上去）。
//
// @Summary  变更集批量原子提交（整体生效或整体回退）
// @Tags     config
// @Accept   json
// @Produce  json
// @Param    changeset body ChangesetReq true "单设备变更集"
// @Param    force query bool false "覆盖业务意图归属硬锁（force=true，审计留痕）"
// @Success  200 {object} Response{data=ChangesetCommitData} "已提交并触发对账"
// @Failure  400 {object} Response "变更集解析失败"
// @Failure  409 {object} Response{data=OwnershipRejection} "路径被业务意图认领（无 force 拒绝）"
// @Failure  502 {object} Response "设备下发失败（已整体回退）"
// @Router   /config/changeset/commit [post]
func (h *ChangesetHandler) Commit(c *beecontext.Context) {
	force := c.Input.Query("force") == "true"
	req, entries, ok := h.decodeChangeset(c)
	if !ok {
		return
	}

	// 归属硬锁（BR-11 口径，CS-04）：任一条目路径被认领且无 force → 整体 409。
	ownerSet := map[string]bool{}
	var owners []string
	for _, pe := range entries {
		for _, o := range intent.DefaultOwnership.Owners(req.Device, pe.req.Path) {
			if !ownerSet[o] {
				ownerSet[o] = true
				owners = append(owners, o)
			}
		}
	}
	if len(owners) > 0 && !force {
		rejectOwnedPath(c, owners)
		return
	}

	// 节点不支持写门禁（BR-12）：2PC 全有全无，任一条目命中即整体拒绝且不打
	// 设备（恢复通道=GET force_refresh 重试成功清标记）。
	if view := h.support(req.Device); view != nil {
		for _, pe := range entries {
			if view.IsUnsupportedPath(pe.req.Path) {
				rejectNodeUnsupported(c, pe.req.Path)
				return
			}
		}
	}

	frags, err := changesetFragments(req.Device, entries)
	if err != nil {
		Error(c, 400, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	res, resOK := h.push.Push(ctx, frags)[req.Device]
	if !resOK {
		// 下发通道未返回该设备结局——按失败处理，绝不当成功（R08）。
		Error(c, 502, "提交失败: 下发通道未返回设备 "+req.Device+" 的结局")
		return
	}
	if res.Err != nil {
		// 写通道学习（CN-04）：设备拒绝可归因为某条目路径的 unknown-element 时
		// 入集并结构化透出（下次同路径提交快速失败；2PC 已整体回退无半配）。
		if view := h.support(req.Device); view != nil {
			for _, pe := range entries {
				if client.UnknownElementForPath(pe.req.Path, res.Err) {
					view.MarkUnsupportedPath(pe.req.Path)
					rejectNodeUnsupported(c, pe.req.Path)
					return
				}
			}
		}
		// 整体回退已由 2PC 完成（candidate discard）；控制器侧零痕迹（R08/§9）。
		Error(c, 502, "提交失败（设备已整体回退）: "+res.Err.Error())
		return
	}

	// 设备 commit 成功——desired/缓存/审计/对账按序落地。
	anchors := map[string]bool{}
	for _, pe := range entries {
		anchors[pe.anchor] = true
		var werr error
		var summary string
		switch pe.req.Op {
		case "delete":
			werr = storeConfigDeleted(h.manager.GetConfigStore(), req.Device, pe.anchor, pe.target)
			summary = "delete " + summarizeDeleted(pe.target)
		default:
			werr = storeConfigMerged(h.manager.GetConfigStore(), req.Device, pe.anchor, pe.target)
			summary = summarizeSubmitted(pe.req.Payload)
			if len(pe.req.Cleared) > 0 {
				summary += fmt.Sprintf("（清除叶: %s）", strings.Join(pe.req.Cleared, ","))
			}
		}
		if werr != nil {
			// 设备已生效而 desired 写失败：如实报错（下一次对账会以设备实际态
			// 回读收敛，不会回滚设备——诚实呈现优于假装失败）。
			Error(c, 500, "设备已提交，但 desired 存储失败: "+werr.Error())
			return
		}
		h.manager.GetAuditStore().Record(audit.Record{
			DeviceIP:     req.Device,
			Path:         pe.req.Path,
			Summary:      summary,
			Triggered:    true,
			Forced:       force && len(owners) > 0,
			ForcedOwners: forcedOwners(force, owners),
		})
	}
	h.manager.GetRunningCache().InvalidatePrefix(req.Device + "|")
	triggered := false
	for a := range anchors {
		if h.manager.TriggerReconcile(req.Device, a) {
			triggered = true
		}
	}

	Success(c, ChangesetCommitData{
		Status:           "COMMITTED",
		Device:           req.Device,
		Entries:          len(entries),
		NonTransactional: res.NonTransactional,
		Reconciliation: ReconcileInfo{
			Triggered: triggered,
			Message:   "Changeset committed. Reconciliation will verify device state.",
		},
	}, "Changeset committed")
}

// changesetFragments 把解码后的条目映射为 2PC 片段（CS-04）：create/update →
// merge 片段（+cleared 叶的预编码 RawXML 片段，CS-05）、delete → delete 片段。
func changesetFragments(device string, entries []previewEntry) ([]intent.Fragment, error) {
	frags := make([]intent.Fragment, 0, len(entries))
	for _, pe := range entries {
		gs, ok := pe.target.(ygot.GoStruct)
		if !ok {
			return nil, fmt.Errorf("条目 %s: 目标类型 %T 不是 GoStruct", pe.req.Path, pe.target)
		}
		switch pe.req.Op {
		case "delete":
			frags = append(frags, intent.Fragment{
				Device: device, Module: pe.desc.Module, Path: pe.anchor,
				Config: gs, Op: intent.FragmentOpDelete,
			})
		default:
			frags = append(frags, intent.Fragment{
				Device: device, Module: pe.desc.Module, Path: pe.anchor, Config: gs,
			})
			if len(pe.req.Cleared) > 0 {
				if pe.desc.XML == nil {
					return nil, fmt.Errorf("条目 %s: 模块 %s 无 XML 通道，无法表达字段级删除", pe.req.Path, pe.desc.Module)
				}
				raw, err := encodeClearedLeaves(pe.desc, gs, pe.req.Cleared)
				if err != nil {
					return nil, fmt.Errorf("条目 %s: 清除叶编码失败: %w", pe.req.Path, err)
				}
				frags = append(frags, intent.Fragment{
					Device: device, Module: pe.desc.Module, Path: pe.anchor, RawXML: raw,
				})
			}
		}
	}
	return frags, nil
}

// --- 反射工具（container = 单 map 字段的 ygot 容器，与 xmlcodec 同约定）---

// containerSubset 从 baseline 容器中筛出 target 容器所含键的条目子集，返回
// (子集容器, 命中条目数)。baseline 为 nil / 非同型 / 无 map 字段（容器根模块）
// 时返回 (baseline, -1)（容器根直接以整棵基线为准）。
func containerSubset(baseline, target interface{}) (interface{}, int) {
	if baseline == nil || target == nil {
		return baseline, 0
	}
	bv, tv := reflect.ValueOf(baseline), reflect.ValueOf(target)
	if bv.Type() != tv.Type() || bv.Kind() != reflect.Ptr || bv.IsNil() || tv.IsNil() {
		return baseline, -1
	}
	bMap, idx := containerMapField(bv.Elem())
	tMap, _ := containerMapField(tv.Elem())
	if idx < 0 || !bMap.IsValid() || !tMap.IsValid() || bMap.IsNil() {
		return baseline, -1
	}
	out := reflect.New(bv.Type().Elem())
	outMap := reflect.MakeMap(bMap.Type())
	count := 0
	for _, k := range tMap.MapKeys() {
		if ev := bMap.MapIndex(k); ev.IsValid() && !ev.IsNil() {
			outMap.SetMapIndex(k, ev)
			count++
		}
	}
	out.Elem().Field(idx).Set(outMap)
	return out.Interface(), count
}

// containerMapField 定位容器的唯一 YANG-list map 字段（带 path tag）。
func containerMapField(sv reflect.Value) (reflect.Value, int) {
	t := sv.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Type.Kind() == reflect.Map && f.Tag.Get("path") != "" {
			return sv.Field(i), i
		}
	}
	return reflect.Value{}, -1
}

// listEntries 展开容器 map 条目为 {键: 条目值}；非 list 容器返回空。
func listEntries(container interface{}) map[interface{}]reflect.Value {
	out := map[interface{}]reflect.Value{}
	if container == nil {
		return out
	}
	cv := reflect.ValueOf(container)
	if cv.Kind() != reflect.Ptr || cv.IsNil() {
		return out
	}
	m, idx := containerMapField(cv.Elem())
	if idx < 0 || !m.IsValid() || m.IsNil() {
		return out
	}
	for _, k := range m.MapKeys() {
		if ev := m.MapIndex(k); ev.IsValid() && !ev.IsNil() {
			out[k.Interface()] = ev
		}
	}
	return out
}

// leafValueByTag 按 YANG path tag 取条目结构体的标量叶值（解指针）。
func leafValueByTag(entry reflect.Value, tag string) (interface{}, bool) {
	ev := entry
	if ev.Kind() == reflect.Ptr {
		if ev.IsNil() {
			return nil, false
		}
		ev = ev.Elem()
	}
	t := ev.Type()
	for i := 0; i < t.NumField(); i++ {
		ft := t.Field(i).Tag.Get("path")
		if j := strings.Index(ft, "|"); j >= 0 {
			ft = ft[:j]
		}
		if j := strings.LastIndex(ft, "/"); j >= 0 {
			ft = ft[j+1:]
		}
		if ft != tag {
			continue
		}
		fv := ev.Field(i)
		if fv.Kind() == reflect.Ptr {
			if fv.IsNil() {
				return nil, false
			}
			return fv.Elem().Interface(), true
		}
		if fv.IsZero() {
			return nil, false
		}
		return fv.Interface(), true
	}
	return nil, false
}

// flattenForDTO 把 GoStruct 值降为 JSON 可序列化的摘要（RFC7951）；失败时
// 返回类型名（诚实降级，不 panic）。
func flattenForDTO(v interface{}) interface{} {
	if gs, ok := v.(ygot.GoStruct); ok {
		if js, err := ygot.EmitJSON(gs, &ygot.EmitJSONConfig{Format: ygot.RFC7951, SkipValidation: true}); err == nil {
			return js
		}
	}
	return fmt.Sprintf("%T", v)
}
