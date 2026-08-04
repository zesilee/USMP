package client

import (
	"context"
)

// ncDriver 是 NETCONFClient 与底层 NETCONF 实现之间的接缝。唯一实现为自研
// netconfcore（backend_core）——scrapligo 已按交付红线 NC-01 移除（2026-08-04，
// 编译不得依赖，守护测试 no_scrapligo_guard_test.go）。接缝保留：未来接入
// 其他协议实现（如 gNMI 转译层）时复用。
//
// 语义契约（原 scrapligo 行为口径，整套 client 测试锁定）：
//   - 各操作返回 ncResult：Result 为应答原文（含 rpc-reply 壳，下游剥壳），
//     Failed 为应答内 <rpc-error>（业务错误，连接仍可用）；error 返回值表示
//     传输/会话级失败（调用方按 isTransportError 判定是否重拨）。
//   - Close 有界优雅关闭，永不挂死调用链；Kill 强制切断（不发 close-session），
//     两者幂等。
type ncDriver interface {
	// GetConfig 下发 <get-config>。filterElem 为完整 <filter …/> 元素（含包装），
	// 空串表示无过滤全量读。
	GetConfig(ctx context.Context, datastore, filterElem string) (ncResult, error)
	// GetState 下发 <get>。subtree 为 subtree filter 体（不含 <filter> 包装），
	// 空串表示全量。
	GetState(ctx context.Context, subtree string) (ncResult, error)
	EditConfig(ctx context.Context, datastore, configXML string) (ncResult, error)
	Commit(ctx context.Context) (ncResult, error)
	// CommitConfirmed 带确认提交，timeoutSec ≥1。
	CommitConfirmed(ctx context.Context, timeoutSec uint) (ncResult, error)
	Discard(ctx context.Context) (ncResult, error)
	// RPC 下发任意操作元素（<rpc> 信封由实现侧包装）。
	RPC(ctx context.Context, payload string) (ncResult, error)
	// Capabilities 服务端 hello 宣告的能力清单。
	Capabilities() []string
	Close() error
	Kill()
}

// ncResult 一次 NETCONF 操作的应答。
type ncResult struct {
	// Result 应答原文（含 rpc-reply 壳）。
	Result string
	// Failed 应答内 rpc-error（nil = 无业务错误）。
	Failed error
}

// dialNCDriver 建连（唯一实现：自研 netconfcore）。
// 历史：Wave 3 曾有 USMP_NETCONF_IMPL 双路径开关，随 scrapligo 移除一并拆除。
func dialNCDriver(info DeviceConnectionInfo) (ncDriver, error) {
	return dialCore(info)
}
