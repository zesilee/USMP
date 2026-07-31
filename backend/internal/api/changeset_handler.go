package api

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/openconfig/ygot/ygot"

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
}

// NewChangesetHandler 构造变更集 handler，设备读闭包与 ConfigHandler 同实现。
func NewChangesetHandler(mgr manager.Manager) *ChangesetHandler {
	cfg := NewConfigHandler(mgr)
	return &ChangesetHandler{manager: mgr, fetch: cfg.fetch}
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
func (h *ChangesetHandler) Preview(c *gin.Context) {
	req, entries, ok := h.decodeChangeset(c)
	if !ok {
		return
	}

	data := ChangesetPreviewData{Device: req.Device, Entries: make([]ChangesetPreviewEntry, 0, len(entries))}
	engine := diff.NewDefaultDiffEngine()
	for _, pe := range entries {
		res := h.previewOne(pe, req.Device, engine)
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
func (h *ChangesetHandler) decodeChangeset(c *gin.Context) (ChangesetReq, []previewEntry, bool) {
	var req ChangesetReq
	if err := c.ShouldBindJSON(&req); err != nil {
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

// previewOne 计算单条目的预览：基线→diff→正向/回滚报文。
func (h *ChangesetHandler) previewOne(pe previewEntry, device string, engine *diff.DefaultDiffEngine) ChangesetPreviewEntry {
	res := ChangesetPreviewEntry{Op: pe.req.Op, Path: pe.req.Path, Diff: []DiffChangeDTO{}}

	baseline, source := h.baseline(device, pe)
	res.BaselineSource = source
	baseSubset, baseCount := containerSubset(baseline, pe.target)

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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
