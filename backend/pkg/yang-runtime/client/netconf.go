package client

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client/netconfcore"
	yangdriver "github.com/leezesi/usmp/backend/pkg/yang-runtime/driver"
)

// NETCONFDefaultPort is the default NETCONF port
const NETCONFDefaultPort = 830

// NETCONFClient implements Client interface for NETCONF protocol
type NETCONFClient struct {
	// opMu 串行化同一连接上的所有 RPC（含整段写事务 edit-config…commit/discard）。
	// netconfcore 会话单 RPC 已内部串行化，但两个并发 Set 交错仍会把彼此的变更
	// 混进同一 candidate（2PC 原子性破坏，R09），写事务的跨 RPC 原子性靠 opMu。
	// 并发调用方（API handler、各 Reconciler）在此排队。
	opMu      sync.Mutex
	mu        sync.RWMutex
	info      DeviceConnectionInfo
	backend   ncDriver
	connected bool
	// nodeSupport 节点级不支持集（CN-04）：自带锁，独立于 mu（避免与连接状态
	// 锁交叉）；connect() 清空（重连重学）。
	nodeSupport nodeSupport
}

// NewNETCONFClient creates a new NETCONF client and connects immediately
func NewNETCONFClient(info DeviceConnectionInfo) (*NETCONFClient, error) {
	if info.Port == 0 {
		info.Port = NETCONFDefaultPort
	}
	if info.Timeout == 0 {
		info.Timeout = 10 * time.Second
	}
	// Credentials come from the shared DeviceStore (resolved by callers). No
	// admin/admin fallback here: an unregistered device connects with empty
	// credentials and SSH fails cleanly, rather than silently masking a missing
	// registration.

	c := &NETCONFClient{
		info: info,
	}

	if err := c.connect(); err != nil {
		// Return the client with the error so caller can handle it
		return c, err
	}

	return c, nil
}

func (c *NETCONFClient) connect() error {
	backend, err := dialNCDriver(c.info)
	if err != nil {
		return err
	}
	c.backend = backend
	c.connected = true
	// 重连清空不支持集（CN-04）：设备可能已升级，学习结果随连接生命周期。
	c.resetNodeSupport()
	return nil
}

// ensureConnected returns a usable backend, dialing if the connection is absent
// or was marked dead. Callers must hold opMu.
func (c *NETCONFClient) ensureConnected() (ncDriver, error) {
	c.mu.RLock()
	backend, ok := c.backend, c.connected
	c.mu.RUnlock()
	if ok && backend != nil {
		return backend, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.connected && c.backend != nil {
		return c.backend, nil
	}
	if err := c.connect(); err != nil {
		return nil, err
	}
	return c.backend, nil
}

// markDisconnected tears down a dead connection so the next call redials.
// 之前传输层死亡后 connected 恒为 true，ClientPool 的 IsConnected() 检查
// 形同虚设，死连接被永久复用——所有请求瞬间 EOF 直到进程重启。
// 强杀语义在各 backend 的 Kill 内实现（scrapligo 的 Close 死锁补丁随迁）。
func (c *NETCONFClient) markDisconnected() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.backend != nil {
		c.backend.Kill()
	}
	c.backend = nil
	c.connected = false
}

// isTransportError reports whether err means the NETCONF session itself is
// unusable (vs. an RPC-level <rpc-error>), so the connection must be redialed.
// 除 netconfcore 的 ErrSessionDead 外，还按错误文案兜底匹配断链错误族——
// 文案口径是历史 scrapligo 路径遗留，对任意传输实现同样适用。
func isTransportError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) ||
		errors.Is(err, netconfcore.ErrSessionDead) ||
		errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "EOF") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "use of closed") ||
		strings.Contains(msg, "session closed")
}

// Get implements Client interface
func (c *NETCONFClient) Get(ctx context.Context, path string, opts ...GetOption) (*GetResult, error) {
	c.opMu.Lock()
	defer c.opMu.Unlock()

	getOpts := &GetOptions{
		Datastore: "running",
	}
	for _, opt := range opts {
		opt.Apply(getOpts)
	}

	filter := c.constructFilter(path)

	backend, err := c.ensureConnected()
	if err != nil {
		return &GetResult{Error: err}, err
	}

	// IncludeState → <get>（配置+状态合并，DP-09）；缺省 <get-config>（DP-03）。
	// GetState 接收 subtree filter 体（实现侧包 <filter type="subtree">），
	// GetConfig 接收完整 <filter> 包装元素。
	doGet := func(d ncDriver) (ncResult, error) {
		if getOpts.IncludeState {
			return d.GetState(ctx, constructSubtreeFilter(path))
		}
		return d.GetConfig(ctx, getOpts.Datastore, filter)
	}

	resp, err := doGet(backend)
	if err != nil && isTransportError(err) {
		// 连接已死（设备重启/闪断/超时后被底层关闭）：重连并重试一次。
		// get/get-config 均幂等，重试安全。
		c.markDisconnected()
		backend, rerr := c.ensureConnected()
		if rerr != nil {
			return &GetResult{Error: err}, err
		}
		resp, err = doGet(backend)
	}
	if err != nil {
		if isTransportError(err) {
			c.markDisconnected()
		}
		return &GetResult{
			Error: err,
		}, err
	}

	// 业务级 rpc-error（设备拒绝，会话仍可用）必须透出为错误：曾只有 Set 检查
	// Failed，Get 把整段 rpc-error XML 当数据成功返回——真机 unknown-element
	// （设备版本无此节点，CN-04）因此被解码层吞成空/乱数据而非错误。
	if resp.Failed != nil {
		return &GetResult{
			Path:      path,
			Timestamp: time.Now(),
			Error:     resp.Failed,
		}, resp.Failed
	}

	if len(resp.Result) == 0 {
		return &GetResult{
			Path:      path,
			Data:      nil,
			Timestamp: time.Now(),
			Error:     fmt.Errorf("empty response"),
		}, fmt.Errorf("empty response")
	}

	result := &GetResult{
		Path:      path,
		Data:      []byte(resp.Result),
		Timestamp: time.Now(),
		Error:     nil,
	}

	return result, nil
}

// Set implements Client interface
func (c *NETCONFClient) Set(ctx context.Context, changes []Change, opts ...SetOption) (*SetResult, error) {
	c.opMu.Lock()
	defer c.opMu.Unlock()

	backend, err := c.ensureConnected()
	if err != nil {
		return nil, err
	}

	setOpts := &SetOptions{
		Datastore: "candidate",
		Commit:    true,
	}
	for _, opt := range opts {
		opt.Apply(setOpts)
	}

	result := &SetResult{
		Success:   true,
		Timestamp: time.Now(),
		Changes:   make([]ChangeResult, len(changes)),
	}

	for i, change := range changes {
		xmlConfig, err := c.marshalChange(change)
		if err != nil {
			result.Changes[i] = ChangeResult{
				Change:  change,
				Success: false,
				Error:   err,
			}
			result.Success = false
			continue
		}

		var resp ncResult
		resp, err = backend.EditConfig(ctx, setOpts.Datastore, xmlConfig)
		if err != nil {
			// 事务中途连接死亡：不在此重试（candidate 状态已不可知），只标记
			// 断连让下一次调用重连重推整个 desired，避免半套配置落盘。
			if isTransportError(err) {
				result.Changes[i] = ChangeResult{Change: change, Success: false, Error: err}
				result.Success = false
				c.markDisconnected()
				return result, err
			}
			result.Changes[i] = ChangeResult{
				Change:  change,
				Success: false,
				Error:   err,
			}
			result.Success = false
			continue
		}
		if resp.Failed != nil {
			result.Changes[i] = ChangeResult{
				Change:  change,
				Success: false,
				Error:   resp.Failed,
			}
			result.Success = false
			continue
		}

		result.Changes[i] = ChangeResult{
			Change:  change,
			Success: true,
			Error:   nil,
		}
	}

	if setOpts.Commit && result.Success {
		resp, err := backend.Commit(ctx)
		if err != nil {
			if isTransportError(err) {
				c.markDisconnected()
			}
			result.Success = false
			result.Message = fmt.Sprintf("partial success: failed to commit: %v", err)
			return result, err
		}
		if resp.Failed != nil {
			result.Success = false
			result.Message = fmt.Sprintf("partial success: commit failed: %v", resp.Failed)
			return result, resp.Failed
		}
	}

	if !result.Success {
		for _, ch := range result.Changes {
			if !ch.Success && ch.Error != nil {
				fmt.Printf("Change failed: %v\n", ch.Error)
			}
		}
		return result, fmt.Errorf("one or more changes failed to apply")
	}

	return result, nil
}

// Subscribe implements Client interface
func (c *NETCONFClient) Subscribe(ctx context.Context, path string, handler func(Notification)) error {
	// NETCONF has no built-in subscription channel like gNMI; pushing device
	// state changes would mean implementing RFC5277 <create-subscription>.
	// TODO(openspec/tasks/code-todo-backlog.md#a1): implement notification subscription.
	return fmt.Errorf("subscription not implemented for NETCONF")
}

// Close implements Client interface.
// 有界关闭语义在各 backend 内实现（scrapligo 的死锁补丁随迁）：优雅路径发
// <close-session>，超时/内部 panic 退化为强切传输层，调用链永不挂死（R08）。
func (c *NETCONFClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.backend == nil {
		return nil
	}
	backend := c.backend
	c.connected = false
	c.backend = nil

	return backend.Close()
}

// IsConnected implements Client interface
func (c *NETCONFClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected && c.backend != nil
}

// ServerCapabilities returns the NETCONF capabilities the device advertised in
// its hello, or nil if not connected. Used by the hybrid schema resolver to
// narrow the usable YANG module set per device.
func (c *NETCONFClient) ServerCapabilities() []string {
	c.mu.RLock()
	backend := c.backend
	c.mu.RUnlock()
	if backend == nil {
		return nil
	}
	return backend.Capabilities()
}

// DiscardCandidate discards the candidate configuration on the device.
// This is used to abort a 2PC transaction before commit.
func (c *NETCONFClient) DiscardCandidate(ctx context.Context) error {
	c.opMu.Lock()
	defer c.opMu.Unlock()
	backend, err := c.ensureConnected()
	if err != nil {
		return err
	}

	resp, err := backend.Discard(ctx)
	if err != nil {
		if isTransportError(err) {
			c.markDisconnected()
		}
		return fmt.Errorf("failed to discard candidate: %w", err)
	}

	if resp.Failed != nil {
		return fmt.Errorf("discard candidate failed: %w", resp.Failed)
	}

	return nil
}

// ErrConfirmedCommitUnsupported reports that the device did not advertise the
// :confirmed-commit capability; callers downgrade to a plain commit (呈现为
// 非事务下发, DP-08).
var ErrConfirmedCommitUnsupported = errors.New("device does not advertise :confirmed-commit capability")

// supportsConfirmedCommit reports whether the advertised capabilities include
// :confirmed-commit (1.0 or 1.1).
func supportsConfirmedCommit(caps []string) bool {
	for _, cap := range caps {
		if strings.HasPrefix(cap, "urn:ietf:params:netconf:capability:confirmed-commit:") {
			return true
		}
	}
	return false
}

// CommitConfirmed sends <commit><confirmed/><confirm-timeout>N</confirm-timeout></commit>:
// the device promotes candidate to running but rolls back automatically unless
// ConfirmCommit arrives within the timeout. Capability is checked before any
// RPC is sent (DP-08).
func (c *NETCONFClient) CommitConfirmed(ctx context.Context, timeout time.Duration) error {
	c.opMu.Lock()
	defer c.opMu.Unlock()
	backend, err := c.ensureConnected()
	if err != nil {
		return err
	}
	if !supportsConfirmedCommit(backend.Capabilities()) {
		return fmt.Errorf("commit confirmed: %w", ErrConfirmedCommitUnsupported)
	}

	secs := uint(timeout / time.Second)
	if secs == 0 {
		secs = 1
	}
	resp, err := backend.CommitConfirmed(ctx, secs)
	if err != nil {
		if isTransportError(err) {
			c.markDisconnected()
		}
		return fmt.Errorf("commit confirmed failed: %w", err)
	}
	if resp.Failed != nil {
		return fmt.Errorf("commit confirmed rejected: %w", resp.Failed)
	}
	return nil
}

// ConfirmCommit sends the confirming <commit/> that finalizes a pending
// confirmed commit (cancels the device-side rollback timer).
func (c *NETCONFClient) ConfirmCommit(ctx context.Context) error {
	c.opMu.Lock()
	defer c.opMu.Unlock()
	backend, err := c.ensureConnected()
	if err != nil {
		return err
	}
	resp, err := backend.Commit(ctx)
	if err != nil {
		if isTransportError(err) {
			c.markDisconnected()
		}
		return fmt.Errorf("confirming commit failed: %w", err)
	}
	if resp.Failed != nil {
		return fmt.Errorf("confirming commit rejected: %w", resp.Failed)
	}
	return nil
}

// constructFilter builds the complete <filter> element for <get-config>（真机
// 回归）：曾是自造 XPath 形态 `<filter … select="/ifm:ifm/…"/>`——RFC6241 无
// type 缺省按 subtree 解释，空元素=什么都不选，真机正确回空 <data/>（列表全空
// +对账永久漂移）。改与状态读同源：constructSubtreeFilter 生成带命名空间的
// 嵌套 subtree 体，外裹 <filter type="subtree">；空路径返回 "" = 全量读。
func (c *NETCONFClient) constructFilter(path string) string {
	sub := constructSubtreeFilter(path)
	if sub == "" {
		return ""
	}
	return `<filter type="subtree">` + sub + `</filter>`
}

// constructSubtreeFilter builds a subtree-filter body for the <get>/<get-config>
// RPC from a config path（如 /ifm:ifm/ifm:interfaces → <ifm xmlns="…"><interfaces/></ifm>）。
// 模块命名空间经驱动注册表解析（未注册模块降级为无命名空间通配，subtree
// filter 语义下匹配任意命名空间）。list 谓词 [key='val'] 转为 RFC6241
// content-match（<interface><name>val</name></interface> 只选中该行——按需单行
// 状态读的地基；整列表 <get> 全量状态真机 30s+ 不回首字节，wire 实证）；
// 非 key='val' 形态的谓词整体剥除（与写路径的谓词拒绝语义对齐）。
// 空路径返回 ""（不构造 filter → 全量读）。
func constructSubtreeFilter(path string) string {
	// 路径规范化（真机回归，wire 抓包实证）：前导双斜杠（URL 手拼等来源）会让
	// 驱动注册表的 HasPrefix 匹配落空 → namespace 静默丢失 → 严格设备 subtree
	// 匹配不到回空。统一收敛为单前导斜杠形态再做后续解析。
	norm := "/" + strings.Trim(strings.TrimSpace(path), "/")
	if norm == "/" {
		return ""
	}
	type filterSeg struct{ name, predKey, predVal string }
	var segs []filterSeg
	var cur strings.Builder
	depth := 0
	flush := func() {
		raw := cur.String()
		cur.Reset()
		if raw == "" {
			return
		}
		name, pk, pv := raw, "", ""
		if i := strings.IndexByte(raw, '['); i >= 0 {
			name = raw[:i]
			body := strings.TrimSuffix(raw[i+1:], "]")
			if j := strings.IndexByte(body, '='); j >= 0 {
				k := strings.TrimSpace(body[:j])
				v := strings.TrimSpace(body[j+1:])
				if len(v) >= 2 && (v[0] == '\'' || v[0] == '"') && v[len(v)-1] == v[0] {
					pk, pv = k, v[1:len(v)-1]
					if c := strings.IndexByte(pk, ':'); c >= 0 {
						pk = pk[c+1:]
					}
				}
			}
		}
		if c := strings.IndexByte(name, ':'); c >= 0 {
			name = name[c+1:]
		}
		if name != "" {
			segs = append(segs, filterSeg{name, pk, pv})
		}
	}
	// 谓词值可含 "/"（如 [name='GE0/0/1']）：括号深度内的 "/" 不切段。
	for _, r := range strings.Trim(norm, "/") {
		switch {
		case r == '[':
			depth++
			cur.WriteRune(r)
		case r == ']':
			if depth > 0 {
				depth--
			}
			cur.WriteRune(r)
		case r == '/' && depth == 0:
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	if len(segs) == 0 {
		return ""
	}
	ns := ""
	if d, ok := yangdriver.DecoderFor(norm); ok && d.XML != nil {
		ns = d.XML.Namespace
	}
	var b strings.Builder
	for i, sg := range segs {
		b.WriteByte('<')
		b.WriteString(sg.name)
		if i == 0 && ns != "" {
			fmt.Fprintf(&b, ` xmlns=%q`, ns)
		}
		if i == len(segs)-1 && sg.predKey == "" {
			b.WriteString("/>")
			continue
		}
		b.WriteByte('>')
		if sg.predKey != "" {
			b.WriteByte('<')
			b.WriteString(sg.predKey)
			b.WriteByte('>')
			_ = xml.EscapeText(&b, []byte(sg.predVal))
			b.WriteString("</")
			b.WriteString(sg.predKey)
			b.WriteByte('>')
		}
	}
	for i := len(segs) - 1; i >= 0; i-- {
		if i == len(segs)-1 && segs[i].predKey == "" {
			continue
		}
		b.WriteString("</")
		b.WriteString(segs[i].name)
		b.WriteByte('>')
	}
	return b.String()
}

func (c *NETCONFClient) marshalChange(change Change) (string, error) {
	// 注册表分派与删除编码提取为导出纯函数（CS-01）；仅在注册表未命中且
	// 非删除变更时降级到本方法保留的 legacy xml.Marshal 兜底链（R08）。
	out, encErr := EncodeChangeXML(change)
	if encErr == nil || !errors.Is(encErr, ErrNoXMLEncoder) || change.Type == DeleteChange {
		return out, encErr
	}

	output, err := xml.Marshal(change.NewValue)
	if err == nil {
		outputStr := string(output)
		repl := strings.NewReplacer(
			"<VlanId>", "<vlan-id>",
			"</VlanId>", "</vlan-id>",
			"<Vlan>", "<vlan>",
			"</Vlan>", "</vlan>",
			"<VLans>", "<vlans>",
			"</VLans>", "</vlans>",
			"<Name>", "<name>",
			"</Name>", "</name>",
			"<Status>", "<status>",
			"</Status>", "</status>",
			"<Config>", "<config>",
			"</Config>", "</config>",
		)
		outputStr = repl.Replace(outputStr)
		return outputStr, nil
	}

	// encoding/xml cannot marshal a map at all, so the typed-map change values
	// the diff engine emits reach this point and are assembled entry by entry.
	v := reflect.ValueOf(change.NewValue)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() == reflect.Map {
		var builder strings.Builder

		containerTag := "vlans"
		if strings.HasSuffix(change.Path, "vlans") {
			containerTag = "vlans"
		} else if strings.HasSuffix(change.Path, "vlan") {
			containerTag = "vlan"
		} else {
			containerTag = "list"
		}
		builder.WriteString(fmt.Sprintf("<%s>", containerTag))

		for _, key := range v.MapKeys() {
			entryVal := v.MapIndex(key)
			if entryVal.IsValid() && !entryVal.IsNil() {
				// Each entry is a struct pointer, which xml.Marshal handles fine.
				entryXML, err2 := xml.Marshal(entryVal.Interface())
				if err2 != nil {
					return "", fmt.Errorf("failed to marshal map entry: %w", err2)
				}
				builder.Write(entryXML)
			}
		}

		builder.WriteString(fmt.Sprintf("</%s>", containerTag))
		outputStr := builder.String()

		// Fix XML element naming: convert from Go camelCase to YANG kebab-case
		repl := strings.NewReplacer(
			"<VlanId>", "<vlan-id>",
			"</VlanId>", "</vlan-id>",
			"<Vlan>", "<vlan>",
			"</Vlan>", "</vlan>",
			"<Name>", "<name>",
			"</Name>", "</name>",
			"<Status>", "<status>",
			"</Status>", "</status>",
			"<Config>", "<config>",
			"</Config>", "</config>",
		)
		outputStr = repl.Replace(outputStr)
		return outputStr, nil
	}

	return "", fmt.Errorf("failed to marshal config to XML: %w", err)
}

// NetconfBaseNS is the NETCONF base namespace carrying the edit-config
// `operation` attribute (RFC 6241 §7.2).
const NetconfBaseNS = "urn:ietf:params:xml:ns:netconf:base:1.0"

// marshalDeleteChange builds a keyed edit-config delete for the model entries in
// target (DP-07)：外层模型容器 + 条目元素带 nc:operation="delete" + 仅 key 叶
// （key 为首元素，对齐 RFC 键匹配惯例；真机与 netconfsim 均按此匹配条目）。
// 经驱动注册表 + 通用引擎（ΛListKeyMap）编码（XC-03）；未注册模型返回明确
// 错误，绝不发送无目标的裸 delete 元素（R08）。
func marshalDeleteChange(target interface{}) (string, error) {
	return encodeDeleteXML(target)
}
