package actor

// 本包（Stack A legacy）集成测试经 client.Set 推送华为模型（含内层 list map
// 形态）。snd-xml-codec 后编码走驱动描述符注册表（XC-04），注册靠空白导入
// 触发；生产二进制经 api→internal/drivers 注册，本测试二进制在此显式注册，
// 否则注册表为空、编码落到 xml.Marshal 兜底对 map 报错。
import (
	_ "github.com/leezesi/usmp/backend/internal/drivers"
)
