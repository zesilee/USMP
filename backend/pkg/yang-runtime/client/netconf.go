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

	"github.com/leezesi/usmp/backend/internal/generated/openconfig"
	yangdriver "github.com/leezesi/usmp/backend/pkg/yang-runtime/driver"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/xmlcodec"
	"github.com/scrapli/scrapligo/driver/netconf"
	"github.com/scrapli/scrapligo/driver/options"
	"github.com/scrapli/scrapligo/response"
	"github.com/scrapli/scrapligo/transport"
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
	opMu      sync.Mutex
	mu        sync.RWMutex
	info      DeviceConnectionInfo
	driver    *netconf.Driver
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
	opts := []util.Option{
		options.WithAuthUsername(c.info.Username),
		options.WithAuthPassword(c.info.Password),
		options.WithPort(c.info.Port),
		options.WithTimeoutSocket(c.info.Timeout),
		options.WithAuthNoStrictKey(),
		options.WithTransportType(transport.StandardTransport),
	}

	driver, err := netconf.NewDriver(
		c.info.IP,
		opts...,
	)
	if err != nil {
		return fmt.Errorf("failed to create NETCONF driver: %w", err)
	}

	err = driver.Open()
	if err != nil {
		return fmt.Errorf("failed to open NETCONF connection: %w", err)
	}

	c.driver = driver
	c.connected = true

	return nil
}

// ensureConnected returns a usable driver, dialing if the connection is absent
// or was marked dead. Callers must hold opMu.
func (c *NETCONFClient) ensureConnected() (*netconf.Driver, error) {
	c.mu.RLock()
	driver, ok := c.driver, c.connected
	c.mu.RUnlock()
	if ok && driver != nil {
		return driver, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.connected && c.driver != nil {
		return c.driver, nil
	}
	if err := c.connect(); err != nil {
		return nil, err
	}
	return c.driver, nil
}

// markDisconnected tears down a dead connection so the next call redials.
// 之前传输层死亡后 connected 恒为 true，ClientPool 的 IsConnected() 检查
// 形同虚设，死连接被永久复用——所有请求瞬间 EOF 直到进程重启。
func (c *NETCONFClient) markDisconnected() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.driver != nil {
		driver := c.driver
		// 不能调 driver.Close()：scrapligo v1.4.0 在死连接上 Close 必死锁
		// （read loop 阻塞在无缓冲 errs 发送、Close 阻塞在无缓冲 done 发送）。
		// 直接关 Channel/Transport 释放 fd；卡在 errs 上的 read goroutine 是
		// scrapligo 缺陷，泄漏量与断连次数同阶，可接受。异步 + recover：
		// 关闭仅是清理，不能阻塞调用链，第三方 double-close 也不许崩进程（R08）。
		go func() {
			defer func() { _ = recover() }()
			_ = driver.Channel.Close()
		}()
	}
	c.driver = nil
	c.connected = false
}

// isTransportError reports whether err means the NETCONF session itself is
// unusable (vs. an RPC-level <rpc-error>), so the connection must be redialed.
func isTransportError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) ||
		errors.Is(err, util.ErrTimeoutError) ||
		errors.Is(err, util.ErrConnectionError) {
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
	// Create option that sets the filter on the operation
	withFilter := func(o interface{}) error {
		op, ok := o.(*netconf.OperationOptions)
		if !ok {
			return util.ErrIgnoredOption
		}
		op.Filter = filter
		return nil
	}

	driver, err := c.ensureConnected()
	if err != nil {
		return &GetResult{Error: err}, err
	}

	resp, err := driver.GetConfig(getOpts.Datastore, withFilter)
	if err != nil && isTransportError(err) {
		// 连接已死（设备重启/闪断/超时后被 scrapligo 关闭）：重连并重试一次。
		// get-config 幂等，重试安全。
		c.markDisconnected()
		driver, rerr := c.ensureConnected()
		if rerr != nil {
			return &GetResult{Error: err}, err
		}
		resp, err = driver.GetConfig(getOpts.Datastore, withFilter)
	}
	if err != nil {
		if isTransportError(err) {
			c.markDisconnected()
		}
		return &GetResult{
			Error: err,
		}, err
	}

	if resp == nil || len(resp.Result) == 0 {
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

	driver, err := c.ensureConnected()
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

		var resp *response.NetconfResponse
		resp, err = driver.EditConfig(setOpts.Datastore, xmlConfig)
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
		resp, err := driver.Commit()
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

// Close implements Client interface
func (c *NETCONFClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.driver == nil {
		return nil
	}

	err := c.driver.Close()
	c.connected = false
	c.driver = nil
	return err
}

// IsConnected implements Client interface
func (c *NETCONFClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected && c.driver != nil
}

// ServerCapabilities returns the NETCONF capabilities the device advertised in
// its hello, or nil if not connected. Used by the hybrid schema resolver to
// narrow the usable YANG module set per device.
func (c *NETCONFClient) ServerCapabilities() []string {
	c.mu.RLock()
	driver := c.driver
	c.mu.RUnlock()
	if driver == nil {
		return nil
	}
	return driver.ServerCapabilities()
}

// DiscardCandidate discards the candidate configuration on the device.
// This is used to abort a 2PC transaction before commit.
func (c *NETCONFClient) DiscardCandidate(ctx context.Context) error {
	c.opMu.Lock()
	defer c.opMu.Unlock()
	driver, err := c.ensureConnected()
	if err != nil {
		return err
	}

	// scrapligo's Discard method discards the candidate config
	resp, err := driver.Discard()
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

func (c *NETCONFClient) constructFilter(path string) string {
	// For simplicity, we use an XPath filter for the path
	// Convert /interfaces/interface[name='eth0'] to XPath notation
	return fmt.Sprintf(`<filter xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" select="%s"/>`, path)
}

func (c *NETCONFClient) marshalChange(change Change) (string, error) {
	if change.Type == DeleteChange {
		return marshalDeleteChange(change.OldValue)
	}
	if change.NewValue == nil {
		// 非删除变更缺 NewValue 无从编码——明确报错优于发送无目标的裸元素（R08）。
		return "", fmt.Errorf("marshal change: nil NewValue for %s change at %s", change.Type, change.Path)
	}

	// If the value is already a byte slice/string, use it directly
	switch v := change.NewValue.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	}

	// Registry-first dispatch (XC-04)：按 GoStruct 类型（容器或 diff 引擎产出的
	// 内层 list map 形态）查驱动描述符，命中则由通用引擎按描述符数据编码。
	// 未命中走既有 fallback 链（openconfig 遗留分支、xml.Marshal 兜底），行为不变。
	if d, ok := yangdriver.XMLEncoderForValue(change.NewValue); ok {
		gs, err := d.WrapXMLValue(change.NewValue)
		if err != nil {
			return "", err
		}
		return xmlcodec.Encode(d.XML, gs)
	}

	// Special case: *openconfig.OpenconfigVlan_Vlans - contains a map field that xml.Marshal can't handle
	// We handle it manually by extracting the map and iterating
	if vlans, ok := change.NewValue.(*openconfig.OpenconfigVlan_Vlans); ok && vlans != nil {
		var builder strings.Builder
		builder.WriteString("<vlans>")
		// Iterate through all VLAN entries in the map
		for _, vlan := range vlans.Vlan {
			if vlan == nil {
				continue
			}
			entryXML, err := xml.Marshal(vlan)
			if err != nil {
				return "", fmt.Errorf("failed to marshal VLAN entry: %w", err)
			}
			builder.Write(entryXML)
		}
		builder.WriteString("</vlans>")
		outputStr := builder.String()
		// Fix XML element naming: convert from Go camelCase to YANG kebab-case
		// We specifically match the full opening and closing tags to avoid accidentally replacing
		// substrings in element content (e.g. "NewName" → "Newname" when replacing "Name" → "name")
		repl := strings.NewReplacer(
			"<VlanId>", "<vlan-id>",
			"</VlanId>", "</vlan-id>",
			"OpenconfigVlan_Vlans_Vlan", "vlan",
			"<Vlan>", "<vlan>",
			"</Vlan>", "</vlan>",
			"<Name>", "<name>",
			"</Name>", "</name>",
			"<Status>", "<status>",
			"</Status>", "</status>",
			"<Config>", "<config>",
			"</Config>", "</config>",
			"<VLans>", "<vlans>",
			"</VLans>", "</vlans>",
		)
		outputStr = repl.Replace(outputStr)
		return outputStr, nil
	}

	// Special case: *openconfig.OpenconfigInterfaces_Interfaces - generates
	// OpenConfig standard XML with proper namespace and YANG-conforming element names
	if interfaces, ok := change.NewValue.(*openconfig.OpenconfigInterfaces_Interfaces); ok && interfaces != nil {
		return buildOpenConfigInterfacesXML(interfaces)
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
			"OpenconfigVlan_Vlans_Vlan", "vlan",
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

// OpenConfig XML namespace constants
const (
	OpenConfigInterfacesNS = "http://openconfig.net/yang/interfaces"
	IanaIfTypeNS           = "urn:ietf:params:xml:ns:yang:iana-if-type"
)

// buildOpenConfigInterfacesXML generates OpenConfig-standard XML for interfaces.
func buildOpenConfigInterfacesXML(interfaces *openconfig.OpenconfigInterfaces_Interfaces) (string, error) {
	if interfaces == nil || len(interfaces.Interface) == 0 {
		return fmt.Sprintf(`<interfaces xmlns="%s"/>`, OpenConfigInterfacesNS), nil
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf(`<interfaces xmlns="%s">`, OpenConfigInterfacesNS))

	// Iterate through all interface entries
	for name, iface := range interfaces.Interface {
		if iface == nil {
			continue
		}

		builder.WriteString("<interface>")

		// Interface name - required, use map key as fallback
		if iface.Name != nil {
			builder.WriteString(fmt.Sprintf("<name>%s</name>", xmlEscape(*iface.Name)))
		} else {
			builder.WriteString(fmt.Sprintf("<name>%s</name>", xmlEscape(name)))
		}

		// Config container - standard YANG pattern
		if iface.Config != nil {
			builder.WriteString("<config>")

			if iface.Config.Name != nil {
				builder.WriteString(fmt.Sprintf("<name>%s</name>", xmlEscape(*iface.Config.Name)))
			}

			// config/type - convert enum integer to IANA standard type name
			switch iface.Config.Type {
			case 1: // ethernetCsmacd
				builder.WriteString(fmt.Sprintf(`<type xmlns:ianaift="%s">ianaift:ethernetCsmacd</type>`, IanaIfTypeNS))
			case 24: // softwareLoopback
				builder.WriteString(fmt.Sprintf(`<type xmlns:ianaift="%s">ianaift:softwareLoopback</type>`, IanaIfTypeNS))
			default:
				builder.WriteString(fmt.Sprintf(`<type xmlns:ianaift="%s">ianaift:ethernetCsmacd</type>`, IanaIfTypeNS))
			}

			if iface.Config.Mtu != nil {
				builder.WriteString(fmt.Sprintf("<mtu>%d</mtu>", *iface.Config.Mtu))
			}

			if iface.Config.Enabled != nil {
				builder.WriteString(fmt.Sprintf("<enabled>%t</enabled>", *iface.Config.Enabled))
			}

			if iface.Config.Description != nil {
				builder.WriteString(fmt.Sprintf("<description>%s</description>", xmlEscape(*iface.Config.Description)))
			}

			builder.WriteString("</config>")
		}

		builder.WriteString("</interface>")
	}

	builder.WriteString("</interfaces>")
	return builder.String(), nil
}

// xmlEscape escapes XML special characters in a string
func xmlEscape(s string) string {
	var buf strings.Builder
	for _, r := range s {
		switch r {
		case '<':
			buf.WriteString("&lt;")
		case '>':
			buf.WriteString("&gt;")
		case '&':
			buf.WriteString("&amp;")
		case '"':
			buf.WriteString("&quot;")
		case '\'':
			buf.WriteString("&apos;")
		default:
			buf.WriteRune(r)
		}
	}
	return buf.String()
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
	if d, ok := yangdriver.XMLEncoderForValue(target); ok {
		gs, err := d.WrapXMLValue(target)
		if err != nil {
			return "", err
		}
		return xmlcodec.EncodeDelete(d.XML, gs)
	}
	return "", fmt.Errorf("marshal delete: unsupported model %T", target)
}
