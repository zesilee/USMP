package client

import (
	"errors"
	"fmt"

	yangdriver "github.com/leezesi/usmp/backend/pkg/yang-runtime/driver"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/xmlcodec"
)

// ErrNoXMLEncoder 表示该值没有注册 XML 编码通道（driver 描述符缺 XML spec）。
// 上层据此决策：Set 链路降级到 legacy 兜底编码；预览链路（CS-03）如实报
// 「不支持报文预览」，绝不伪造报文。
var ErrNoXMLEncoder = errors.New("no XML encoder registered for value")

// EncodeChangeXML 把单个 Change 编码为 edit-config 片段 XML。纯函数：仅做
// 注册表分派 + 通用引擎编码，不含连接态、不含 legacy 字符串替换兜底
// （CS-01 提取自 NETCONFClient.marshalChange，注册表命中路径字节等价）。
//   - DeleteChange → 按 OldValue 查注册表走 xmlcodec.EncodeDelete（键定位删除）
//   - 其余类型   → NewValue 为 string/[]byte 时原样透传；否则查注册表走 xmlcodec.Encode
//   - 注册表未命中 → 包装 ErrNoXMLEncoder 的明确错误
func EncodeChangeXML(change Change) (string, error) {
	if change.Type == DeleteChange {
		return encodeDeleteXML(change.OldValue)
	}
	if change.NewValue == nil {
		// 非删除变更缺 NewValue 无从编码——明确报错优于发送无目标的裸元素（R08）。
		return "", fmt.Errorf("encode change: nil NewValue for %s change at %s", change.Type, change.Path)
	}
	switch v := change.NewValue.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	}
	if d, ok := yangdriver.XMLEncoderForValue(change.NewValue); ok {
		gs, err := d.WrapXMLValue(change.NewValue)
		if err != nil {
			return "", err
		}
		return xmlcodec.Encode(d.XML, gs)
	}
	return "", fmt.Errorf("encode change at %s (%T): %w", change.Path, change.NewValue, ErrNoXMLEncoder)
}

// encodeDeleteXML 构造键定位的条目删除片段（DP-07）：外层模型容器 + 条目元素
// 带 nc:operation="delete" + 仅 key 叶。未注册模型返回 ErrNoXMLEncoder 包装错误。
func encodeDeleteXML(target interface{}) (string, error) {
	if d, ok := yangdriver.XMLEncoderForValue(target); ok {
		gs, err := d.WrapXMLValue(target)
		if err != nil {
			return "", err
		}
		return xmlcodec.EncodeDelete(d.XML, gs)
	}
	return "", fmt.Errorf("encode delete: unsupported model %T: %w", target, ErrNoXMLEncoder)
}
