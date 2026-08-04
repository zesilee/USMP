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
	"github.com/scrapli/scrapligo/util"
)

// NETCONFDefaultPort is the default NETCONF port
const NETCONFDefaultPort = 830

// NETCONFClient implements Client interface for NETCONF protocol
type NETCONFClient struct {
	// opMu 串行化同一连接上的所有 RPC（含整段写事务 edit-config…commit/discard）。
	// scrapligo 的 Driver 非并发安全：buildPayload 的 messageID++ 无锁（并发时
	// 产生重复 message-id，响应被错领/丢失后 RPC 挂到 op-timeout），Channel.Write
	// 也无锁（并发写使 NETCONF 帧字节交错，设备端解析卡死）；且两个并发 Set 交错
	// 会把彼此的变更混进同一 candidate（2PC 原子性破坏，R09）。并发调用方
	// （API handler、各 Reconciler）在此排队，而不是并发打到 driver 上。
	// （自研 core 路径会话内部自带串行化，但写事务的跨 RPC 原子性仍靠 opMu。）
	opMu      sync.Mutex
	mu        sync.RWMutex
	info      DeviceConnectionInfo
	backend   ncDriver
	connected bool
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

	// Connect immediately
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
// 同时覆盖 scrapligo 错误族与自研 core 的 ErrSessionDead（双路径共用）。
func isTransportError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) ||
		errors.Is(err, util.ErrTimeoutError) ||
		errors.Is(err, util.ErrConnectionError) ||
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

	// Apply options
	getOpts := &GetOptions{
		Datastore: "running",
	}
	for _, opt := range opts {
		opt.Apply(getOpts)
	}

	// Construct filter
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

	// Apply options
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

	// Apply each change
	for i, change := range changes {
		// For NETCONF, we need to convert the change to XML
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
		// Check for NETCONF level errors (<rpc-error> in response)
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

	// Commit if requested and all changes succeeded
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
		// If response contains <rpc-error>, resp.Failed will be non-nil
		if resp.Failed != nil {
			result.Success = false
			result.Message = fmt.Sprintf("partial success: commit failed: %v", resp.Failed)
			return result, resp.Failed
		}
	}

	if !result.Success {
		// Print any errors for debugging
		for _, ch := range result.Changes {
			if !ch.Success && ch.Error != nil {
				fmt.Printf("Change failed: %v\n", ch.Error)
			}
		}
		// If any change failed, return an error to caller
		return result, fmt.Errorf("one or more changes failed to apply")
	}

	return result, nil
}

// Subscribe implements Client interface
func (c *NETCONFClient) Subscribe(ctx context.Context, path string, handler func(Notification)) error {
	// NETCONF doesn't have built-in subscription like gNMI
	// TODO: Implement NETCONF notification subscription
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

func (c *NETCONFClient) constructFilter(path string) string {
	// For simplicity, we use an XPath filter for the path
	// Convert /interfaces/interface[name='eth0'] to XPath notation
	return fmt.Sprintf(`<filter xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" select="%s"/>`, path)
}

// constructSubtreeFilter builds a subtree-filter body for the <get> RPC from a
// config path（如 /ifm:ifm/ifm:interfaces → <ifm xmlns="…"><interfaces/></ifm>）。
// 模块命名空间经驱动注册表解析（未注册模块降级为无命名空间通配，subtree
// filter 语义下匹配任意命名空间）；list 谓词剥除（整列表读，与写路径的谓词
// 拒绝语义对齐）。空路径返回 ""（scrapligo 不构造 filter → 全量 <get>）。
func constructSubtreeFilter(path string) string {
	norm := strings.TrimRight(strings.TrimSpace(path), "/")
	// 谓词值可含 "/"（如 [name='GE0/0/1']），须在按 "/" 切段前整体剥除 […] 区段。
	stripped := norm
	if strings.Contains(stripped, "[") {
		var sb strings.Builder
		depth := 0
		for _, r := range stripped {
			switch {
			case r == '[':
				depth++
			case r == ']' && depth > 0:
				depth--
			case depth == 0:
				sb.WriteRune(r)
			}
		}
		stripped = sb.String()
	}
	var names []string
	for _, seg := range strings.Split(strings.Trim(stripped, "/"), "/") {
		if i := strings.Index(seg, ":"); i >= 0 {
			seg = seg[i+1:]
		}
		if seg != "" {
			names = append(names, seg)
		}
	}
	if len(names) == 0 {
		return ""
	}
	ns := ""
	if d, ok := yangdriver.DecoderFor(norm); ok && d.XML != nil {
		ns = d.XML.Namespace
	}
	var b strings.Builder
	for i, name := range names {
		b.WriteByte('<')
		b.WriteString(name)
		if i == 0 && ns != "" {
			fmt.Fprintf(&b, ` xmlns=%q`, ns)
		}
		if i == len(names)-1 {
			b.WriteString("/>")
		} else {
			b.WriteByte('>')
		}
	}
	for i := len(names) - 2; i >= 0; i-- {
		b.WriteString("</")
		b.WriteString(names[i])
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

	// Try xml.Marshal for other types
	output, err := xml.Marshal(change.NewValue)
	if err == nil {
		// Success, fix naming and return
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

	// If xml.Marshal failed and it's a map, handle manually
	v := reflect.ValueOf(change.NewValue)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() == reflect.Map {
		var builder strings.Builder

		// Determine container tag based on the path
		containerTag := "vlans"
		if strings.HasSuffix(change.Path, "vlans") {
			containerTag = "vlans"
		} else if strings.HasSuffix(change.Path, "vlan") {
			containerTag = "vlan"
		} else {
			containerTag = "list"
		}
		builder.WriteString(fmt.Sprintf("<%s>", containerTag))

		// Iterate through all map entries and marshal each value individually
		for _, key := range v.MapKeys() {
			entryVal := v.MapIndex(key)
			if entryVal.IsValid() && !entryVal.IsNil() {
				// Each entry is a pointer to a struct that can be marshaled
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

	// Still failed - return original error
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
