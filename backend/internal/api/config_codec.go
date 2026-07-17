package api

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openconfig/ygot/ygot"

	// 空白导入触发 huawei 驱动描述符注册（DR-01）：本包编解码与 manager 路由
	// 均从 driver 注册表查表。
	_ "github.com/leezesi/usmp/backend/internal/drivers"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/driver"
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
	var parsed ygot.GoStruct
	if d, ok := driver.DecoderFor(path); ok {
		if p, err := d.DecodeXML(raw); err == nil {
			parsed = p
		}
	}
	if parsed == nil {
		return data
	}

	// SkipValidation：回读是「展示设备真值」，设备侧值不合本地 pattern（如子接口
	// number 带小数点）不应让整个回读降级成不透明 XML（R08）；写路径校验不受影响。
	js, err := ygot.EmitJSON(parsed, &ygot.EmitJSONConfig{Format: ygot.RFC7951, SkipValidation: true})
	if err != nil {
		return data
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(js), &out); err != nil {
		return data
	}
	return out
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
