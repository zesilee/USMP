package controllers

import (
	"errors"
	"net"
	"strings"
	"time"
)

// Shared error-classification and backoff helpers used by the remaining business
// CRD controllers (Route/Switch/Interface). Extracted from the retired
// BusinessVlan Actor controller (P2 组4a).
const (
	// 错误类型分类
	ErrorTypeTemporary = "Temporary"
	ErrorTypePermanent = "Permanent"
	// 最大重试次数
	maxRetryCount = 5
	// 初始重试间隔
	baseRetryInterval = 5 * time.Second
)

// classifyError 分类错误类型: 临时错误 vs 永久错误
func classifyError(err error) string {
	// 网络连接错误通常是临时的
	if strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "i/o timeout") ||
		strings.Contains(err.Error(), "connection reset") ||
		strings.Contains(err.Error(), "no route to host") ||
		strings.Contains(err.Error(), "network is unreachable") {
		return ErrorTypeTemporary
	}

	// 认证错误通常是配置问题（永久）
	if strings.Contains(err.Error(), "authentication failed") ||
		strings.Contains(err.Error(), "permission denied") ||
		strings.Contains(err.Error(), "unauthorized") {
		return ErrorTypePermanent
	}

	// NETCONF 协议错误通常是配置问题（永久）
	if strings.Contains(err.Error(), "rpc-error") ||
		strings.Contains(err.Error(), "invalid value") ||
		strings.Contains(err.Error(), "bad attribute") ||
		strings.Contains(err.Error(), "unknown element") {
		return ErrorTypePermanent
	}

	// DNS 解析错误可能是临时的
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return ErrorTypeTemporary
	}

	// 默认视为临时错误，给予重试机会
	return ErrorTypeTemporary
}

// calculateBackoff 计算指数退避的重队列时间
func calculateBackoff(retryCount int) time.Duration {
	if retryCount <= 0 {
		return baseRetryInterval
	}
	// 指数退避: 5s, 10s, 20s, 40s, 60s (最大值)
	backoff := baseRetryInterval * time.Duration(1<<retryCount)
	maxBackoff := 60 * time.Second
	if backoff > maxBackoff {
		return maxBackoff
	}
	return backoff
}
