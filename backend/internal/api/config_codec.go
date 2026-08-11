package api

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	// 空白导入触发 huawei 驱动描述符注册（DR-01）：本包编解码与 manager 路由
	// 均从 driver 注册表查表。
	_ "github.com/leezesi/usmp/backend/internal/drivers"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/driver"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/object"
)

// decodeRunningConfig turns a raw NETCONF XML readback into an RFC7951-shaped
// map (yang-named keys, list-as-array) so config-read endpoints return the same
// structure the frontend submits and lists — e.g. {"interface":[{"name":...}]}.
// Without this the handler returned opaque XML bytes (a base64 string over JSON)
// from which the "接口配置" list could extract no rows. Unrecognised paths or
// already-decoded (non-[]byte) data pass through unchanged.
func decodeRunningConfig(path string, data interface{}) interface{} {
	raw, ok := data.([]byte)
	if !ok || len(raw) == 0 || raw[0] != '<' {
		return data
	}

	// 解码器按驱动描述符注册表查表（DR-03）——不再散落路径字符串匹配。
	var parsed object.Object
	var anchor string
	if d, ok := driver.DecoderFor(path); ok {
		p, err := d.DecodeXML(raw)
		if err != nil {
			// 解码失败必须留痕（真机排障教训：静默原始透传=前端零行+无线索，
			// 两天才定位到此）。仍降级透传原始字节（R08），但日志说清败因。
			log.Printf("decodeRunningConfig: %s 解码失败，降级原始透传（前端将无行可渲染）: %v", path, err)
		} else {
			parsed = p
			// DecodeXML 的解码根即 EncodeAnchor 容器（手写块与 registerPlain 均如此
			// 构造），剥层以它为基准把 emit 结果对齐到请求 path。
			anchor = d.EncodeAnchor
		}
	}
	if parsed == nil {
		return data
	}

	// 生成式 MarshalJSON（S3，替 ygot.EmitJSON）：native 生成物无校验面，
	// 天然等价旧 SkipValidation 语义——回读是「展示设备真值」，设备侧值不合
	// 本地 pattern 也不降级（R08）；写路径校验不受影响。
	jm, ok := parsed.(json.Marshaler)
	if !ok {
		return data
	}
	js, err := jm.MarshalJSON()
	if err != nil {
		return data
	}
	var out map[string]interface{}
	if err := json.Unmarshal(js, &out); err != nil {
		return data
	}
	return peelToPath(out, anchor, path)
}

// peelToPath 把「以解码根（anchor）为根」的 RFC7951 map 剥到「以请求 path 为根」的
// 子树——批量接入模块（registerPlain）的解码根恒为模块根容器，读子路径（list Tab、
// 表单 Tab、leafref 拉取）若不剥层，前端会把容器键当数据行渲染（真机 devm ports
// 「一行且位置=port」症状）。规则：
//   - 段对齐按局部名（去模块前缀）——ni 存在 /ni: 与 /network-instance: 双前缀口径，
//     前端谓词段（port[...]）也不带前缀，按前缀比对会误判；
//   - 谓词段（含 '['）停剥：单行状态读（include_state）契约是返回谓词段的父容器
//     子树（前端 sub[listKey] 取行），且 RFC7951 map 无法按谓词索引数组；
//   - 中途键缺失（设备该子树无数据）→ 空 map（前端零行）；
//   - 中途值非 map（形状意外）→ 返回已剥到的层（R08 降级，不伪造也不整树透出）。
//
// 前 skip 段只数段数、不校验与 anchor 局部名对齐——对齐由注册纪律保证
// （MatchDecode 前缀与 EncodeAnchor 同根，见 registerPlain / huawei.go 手写块）；
// MatchDecode 宽于 anchor 的路径（如 /ifm:ifm/ifm:global）本就解不出数据，
// 剥成空 map 是比旧「整树透出错误数据」更安全的降级。
func peelToPath(m map[string]interface{}, anchor, path string) interface{} {
	segs := pathLocals(path)
	skip := len(pathLocals(anchor))
	if anchor == "" || len(segs) < skip {
		return m
	}
	var cur interface{} = m
	for _, seg := range segs[skip:] {
		if strings.Contains(seg, "[") {
			break
		}
		node, ok := cur.(map[string]interface{})
		if !ok {
			break
		}
		child, ok := lookupLocal(node, seg)
		if !ok {
			return map[string]interface{}{}
		}
		cur = child
	}
	return cur
}

// pathLocals 把配置路径拆成局部名段（去模块前缀，保留谓词部分）。
// 已知局限：按 '/' 切分不感知引号，谓词值含 '/'（如 position='MEth0/0/0'）会把
// 谓词错切成多段——当前无害，因 peelToPath 遇首个含 '[' 的段即停剥、错切尾段
// 永不被访问。若将来改停剥逻辑（如支持按谓词索引数组），须先把此函数改成
// 引号感知切分。
func pathLocals(path string) []string {
	raw := strings.Split(strings.Trim(path, "/"), "/")
	segs := make([]string, 0, len(raw))
	for _, s := range raw {
		if s == "" {
			continue
		}
		// 只剥谓词前的模块前缀（谓词内可能含 ':'，如命名空间限定的 key）。
		name, pred := s, ""
		if i := strings.Index(s, "["); i >= 0 {
			name, pred = s[:i], s[i:]
		}
		if j := strings.Index(name, ":"); j >= 0 {
			name = name[j+1:]
		}
		segs = append(segs, name+pred)
	}
	return segs
}

// lookupLocal 按局部名取子节点：RFC7951 emit 在跨模块边界（augment 子树）会带
// "module:name" 前缀键，同模块子节点为裸名，两种形态都须命中。
func lookupLocal(node map[string]interface{}, local string) (interface{}, bool) {
	if v, ok := node[local]; ok {
		return v, true
	}
	for k, v := range node {
		if i := strings.Index(k, ":"); i >= 0 && k[i+1:] == local {
			return v, true
		}
	}
	return nil, false
}

// convertConfig decodes request data into a typed desired config via the single
// RFC7951 path (BR-06)：body 契约 = 以 path 为根的 RFC7951 子树。按 driver 注册表
// 查得编码描述符 → 按其 EncodeAnchor（DR-05）把子树机械包裹成锚点相对 JSON →
// 生成的 Unmarshal 根级解码。未注册路径 / path 与锚点非前缀 / path 段含 list 谓词 /
// 解码失败一律显式报错（调用方 400），SHALL NOT 回退手写转换器或静默存原始 map。
func convertConfig(path string, data map[string]interface{}) (interface{}, error) {
	v, _, err := convertConfigAnchored(path, data)
	return v, err
}

// convertConfigAnchored 同 convertConfig，并返回描述符锚点路径：解码值以锚点为根，
// desired 的存储与对账触发 SHALL 以锚点为 key（子路径下发归一化，周期对账按模块
// 路径入队才能看到它）。
func convertConfigAnchored(path string, data map[string]interface{}) (interface{}, string, error) {
	d, ok := driver.EncoderFor(path)
	if !ok {
		return nil, "", fmt.Errorf("路径 %q 未注册编码驱动（driver 注册表无描述符覆盖）", path)
	}
	wrapped, err := wrapToAnchor(d.EncodeAnchor, path, data)
	if err != nil {
		return nil, "", err
	}
	jsonBytes, err := json.Marshal(wrapped)
	if err != nil {
		return nil, "", err
	}
	dest := d.NewStruct()
	if err := d.Unmarshal(jsonBytes, dest); err != nil {
		return nil, "", fmt.Errorf("RFC7951 解码失败（body 须为以 path 为根的 YANG 真名子树）: %w", err)
	}
	return dest, d.EncodeAnchor, nil
}

// wrapToAnchor 把「以 path 为根的子树」机械包裹为「以描述符锚点为根」的 JSON：
// path 剥去锚点前缀后的每个段（去模块前缀）自内向外套一层对象。path==锚点 → 零包裹。
func wrapToAnchor(anchor, path string, data map[string]interface{}) (map[string]interface{}, error) {
	if anchor == "" {
		return nil, fmt.Errorf("驱动描述符缺少 EncodeAnchor（DR-05）")
	}
	norm := strings.TrimRight(path, "/")
	if norm != anchor && !strings.HasPrefix(norm, anchor+"/") {
		return nil, fmt.Errorf("路径 %q 不在编码锚点 %q 之下", path, anchor)
	}
	cur := data
	suffix := strings.TrimPrefix(norm, anchor)
	segs := strings.Split(strings.Trim(suffix, "/"), "/")
	for i := len(segs) - 1; i >= 0; i-- {
		seg := segs[i]
		if seg == "" {
			continue
		}
		if strings.ContainsAny(seg, "[]") {
			return nil, fmt.Errorf("路径段 %q 含 list 谓词，子树写入不支持（请写入其容器路径）", seg)
		}
		if j := strings.Index(seg, ":"); j >= 0 {
			seg = seg[j+1:]
		}
		cur = map[string]interface{}{seg: cur}
	}
	return cur, nil
}
