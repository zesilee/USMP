// ops.go — NETCONF 标准操作层：把 8 个动作映射为 <rpc> 内层报文（RFC 6241）。
// 全部返回完整 <rpc-reply> 原文，解析/剥壳由上层适配层负责（Wave 3）。
package netconfcore

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// datastore 进 XML 元素名位置，白名单校验防注入。
var datastoreRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

func validDatastore(ds string) error {
	if !datastoreRe.MatchString(ds) {
		return fmt.Errorf("netconfcore: 非法 datastore: %q", ds)
	}
	return nil
}

// GetConfig <get-config>，subtree 为空则不带 filter。
func (s *Session) GetConfig(ctx context.Context, datastore, subtree string) ([]byte, error) {
	if err := validDatastore(datastore); err != nil {
		return nil, err
	}
	var b strings.Builder
	b.WriteString("<get-config><source><" + datastore + "/></source>")
	if subtree != "" {
		b.WriteString(`<filter type="subtree">` + subtree + `</filter>`)
	}
	b.WriteString("</get-config>")
	return s.Do(ctx, []byte(b.String()))
}

// GetState <get>（运行状态读，config=false 叶也在内）。
func (s *Session) GetState(ctx context.Context, subtree string) ([]byte, error) {
	var b strings.Builder
	b.WriteString("<get>")
	if subtree != "" {
		b.WriteString(`<filter type="subtree">` + subtree + `</filter>`)
	}
	b.WriteString("</get>")
	return s.Do(ctx, []byte(b.String()))
}

// EditConfig <edit-config>（merge 缺省语义，操作细化由 payload 内嵌 operation 属性表达）。
func (s *Session) EditConfig(ctx context.Context, datastore, config string) ([]byte, error) {
	if err := validDatastore(datastore); err != nil {
		return nil, err
	}
	return s.Do(ctx, []byte(
		"<edit-config><target><"+datastore+"/></target><config>"+config+"</config></edit-config>"))
}

// Commit <commit/>（candidate → running）。
func (s *Session) Commit(ctx context.Context) ([]byte, error) {
	return s.Do(ctx, []byte("<commit/>"))
}

// CommitConfirmed 带确认的提交：timeout 秒内无 confirm 自动回滚（2PC prepare）。
func (s *Session) CommitConfirmed(ctx context.Context, timeoutSec int) ([]byte, error) {
	if timeoutSec <= 0 {
		return nil, fmt.Errorf("netconfcore: confirm-timeout 须为正数: %d", timeoutSec)
	}
	return s.Do(ctx, []byte(fmt.Sprintf(
		"<commit><confirmed/><confirm-timeout>%d</confirm-timeout></commit>", timeoutSec)))
}

// DiscardChanges <discard-changes/>（丢弃 candidate 未提交内容，2PC abort）。
func (s *Session) DiscardChanges(ctx context.Context) ([]byte, error) {
	return s.Do(ctx, []byte("<discard-changes/>"))
}
