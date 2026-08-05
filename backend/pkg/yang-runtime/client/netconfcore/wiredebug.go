package netconfcore

import (
	"log"
	"os"
)

// wire 抓包开关（真机排障）：USMP_NETCONF_WIRE_DEBUG=1 时把每条发出的 <rpc>
// 原文与收到的 <rpc-reply> 原文（含字节长度）打进标准日志，肉眼比对线上报文
// 定位「空回复/超时/框架错位」类问题。缺省关闭零输出；每次读 env（而非 init
// 缓存）使 kubectl set env 后无需改代码即可开关，也便于测试注入。
const wireDebugEnv = "USMP_NETCONF_WIRE_DEBUG"

// wireHeadMax/wireTailMax 截断阈值：兆级全量回读只留头尾（尾部保留能看到
// rpc-reply 收尾是否完整，判定截断/框架问题的关键信号）。
const (
	wireHeadMax = 2048
	wireTailMax = 512
)

func wireLog(dir string, payload []byte) {
	if os.Getenv(wireDebugEnv) != "1" {
		return
	}
	if len(payload) <= wireHeadMax+wireTailMax {
		log.Printf("netconf-wire %s %dB: %s", dir, len(payload), payload)
		return
	}
	log.Printf("netconf-wire %s %dB: %s…%s", dir, len(payload),
		payload[:wireHeadMax], payload[len(payload)-wireTailMax:])
}

// wireLogf 会话级事件（协商结果等），同一开关控制。
func wireLogf(format string, args ...interface{}) {
	if os.Getenv(wireDebugEnv) != "1" {
		return
	}
	log.Printf("netconf-wire "+format, args...)
}
