// Package rpcrisk 是 rpc 高危分类的唯一口径，rpcgen（执行确认分级）与
// lefttreegen（左树节点警示）共用，禁止各自复制词表（D2：口径不漂移）。
package rpcrisk

import "strings"

// highRiskWords flag rpcs with major, often irreversible device impact — restart/
// power/delete/rollback/upgrade/etc. Deliberately excludes the mild, common
// reset-/clear-counters family (those get the base confirm, not the escalated
// warning). Heuristic by design; can later switch to a model annotation.
var highRiskWords = []string{
	"restart", "reboot", "reload", "warm", "cold", "power",
	"delete", "batch", "erase", "format", "rollback", "switch",
	"upgrade", "undo",
}

// IsHighRisk reports whether an rpc name matches a high-risk verb (word-boundary
// on '-', so "reset" never matches "restart" and "clear" never matches).
func IsHighRisk(name string) bool {
	for _, s := range strings.Split(name, "-") {
		for _, w := range highRiskWords {
			if s == w {
				return true
			}
		}
	}
	return false
}
