package client

import (
	"errors"
	"strings"
	"sync"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client/netconfcore"
)

// 节点级不支持集（CN-04）：设备软件版本没有的 YANG 节点（真机对 <get-config>/
// <get>/<edit-config> 回 unknown-element/313）按连接生命周期记忆——与 hello
// capabilities 同口径：内存、不持久化（R03）、重连清空重学（设备升级自然刷新）。
// 学习触发在 API 层（归因需请求路径），存取在此（连接是设备身份的天然锚点）。

// nodeSupport is the per-connection unsupported-path set. Zero value ready.
type nodeSupport struct {
	mu    sync.RWMutex
	paths map[string]struct{}
}

// normalizeNodePath 归一首尾斜杠："/x"、"x"、"x/" 是同一条目。调用方形态不一
// （API *path 带首斜杠、reconciler 常量带首斜杠、模块根前缀不带），不归一会
// 「标了但查不中」静默失效。
func normalizeNodePath(p string) string { return strings.Trim(p, "/") }

func (n *nodeSupport) mark(path string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.paths == nil {
		n.paths = make(map[string]struct{})
	}
	n.paths[normalizeNodePath(path)] = struct{}{}
}

func (n *nodeSupport) unmark(path string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.paths, normalizeNodePath(path))
}

func (n *nodeSupport) has(path string) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	_, ok := n.paths[normalizeNodePath(path)]
	return ok
}

// under returns marked paths that equal prefix or live below it（按段边界，
// "devm:devm" 不命中 "devm:devm2/..."）。
func (n *nodeSupport) under(prefix string) []string {
	p := normalizeNodePath(prefix)
	n.mu.RLock()
	defer n.mu.RUnlock()
	var out []string
	for k := range n.paths {
		if k == p || strings.HasPrefix(k, p+"/") {
			out = append(out, k)
		}
	}
	return out
}

func (n *nodeSupport) clear() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.paths = nil
}

// MarkUnsupportedPath 记录设备不支持的路径（unknown-element 归因命中后调用）。
func (c *NETCONFClient) MarkUnsupportedPath(path string) { c.nodeSupport.mark(path) }

// ClearUnsupportedPath 移除标记（force 重试成功后的恢复通道，BR-12）。
func (c *NETCONFClient) ClearUnsupportedPath(path string) { c.nodeSupport.unmark(path) }

// IsUnsupportedPath 查询路径是否已被学习为不支持（API 快速失败判定）。
func (c *NETCONFClient) IsUnsupportedPath(path string) bool { return c.nodeSupport.has(path) }

// UnsupportedPathsUnder 返回前缀下已学习的不支持路径（CN-05 透出）。
func (c *NETCONFClient) UnsupportedPathsUnder(prefix string) []string {
	return c.nodeSupport.under(prefix)
}

// resetNodeSupport 清空不支持集；connect()（含重连）调用（CN-04 重连清空）。
func (c *NETCONFClient) resetNodeSupport() { c.nodeSupport.clear() }

// UnknownElementForPath reports whether err is an unknown-element rpc-error
// attributable to path: severity=error、tag ∈ {unknown-element, bad-element}，
// 且 bad-element 与路径某段的局部名一致（保守归因——瞬时/无关错误不入集）。
func UnknownElementForPath(path string, err error) bool {
	var re *netconfcore.RPCReplyError
	if err == nil || !errors.As(err, &re) {
		return false
	}
	segs := pathSegmentNames(path)
	if len(segs) == 0 {
		return false
	}
	for _, e := range re.Errors {
		if e.Severity != "" && e.Severity != "error" {
			continue
		}
		if e.Tag != "unknown-element" && e.Tag != "bad-element" {
			continue
		}
		bad := strings.TrimSpace(e.BadElement)
		if bad == "" {
			continue
		}
		for _, s := range segs {
			if s == bad {
				return true
			}
		}
	}
	return false
}

// pathSegmentNames yields each segment's local name: strips [predicates] and
// namespace prefixes ("devm:cards" → "cards").
func pathSegmentNames(path string) []string {
	var out []string
	for _, seg := range strings.Split(path, "/") {
		if i := strings.IndexByte(seg, '['); i >= 0 {
			seg = seg[:i]
		}
		if i := strings.LastIndexByte(seg, ':'); i >= 0 {
			seg = seg[i+1:]
		}
		if seg = strings.TrimSpace(seg); seg != "" {
			out = append(out, seg)
		}
	}
	return out
}
