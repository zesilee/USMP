package api

import (
	"context"
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/stretchr/testify/assert"
)

// xmlClient is a fake Client whose Get returns canned NETCONF XML, standing in
// for a real device readback.
type xmlClient struct{ xml string }

func (c *xmlClient) Get(context.Context, string, ...client.GetOption) (*client.GetResult, error) {
	return &client.GetResult{Data: []byte(c.xml)}, nil
}
func (c *xmlClient) Set(context.Context, []client.Change, ...client.SetOption) (*client.SetResult, error) {
	return nil, nil
}
func (c *xmlClient) Subscribe(context.Context, string, func(client.Notification)) error { return nil }
func (c *xmlClient) Close() error                                                       { return nil }
func (c *xmlClient) IsConnected() bool                                                  { return true }
func (c *xmlClient) DiscardCandidate(context.Context) error                             { return nil }

// TestFetchFromDevice_ParsesIfmInterfaces: the config-read path must decode the
// NETCONF XML readback into the typed interface struct, not hand back raw XML
// bytes. Returning raw bytes (base64 over JSON) is why the "接口配置" list page
// could not extract any rows for a freshly added interface.
func TestFetchFromDevice_ParsesIfmInterfaces(t *testing.T) {
	xml := `<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm"><interfaces>` +
		`<interface><name>GigabitEthernet0/0/2</name></interface></interfaces></ifm>`
	p := &fakePool{client: &xmlClient{xml: xml}}
	h := NewConfigHandler(fakePoolManager{pool: p})

	got, err := h.fetchFromDevice(context.Background(), "192.168.1.1", "/ifm:ifm/ifm:interfaces")
	assert.NoError(t, err)

	// Must be an RFC7951-shaped map {"interface":[{"name":...}]} — the shape the
	// frontend lists from (listKey='interface', keyField='name') — NOT raw XML
	// bytes and NOT the Go-cased ygot struct.
	m, ok := got.(map[string]interface{})
	assert.True(t, ok, "回读应返回 RFC7951 结构 map，而非裸 XML 字节/ygot 结构（否则前端列表提取不出行）")
	if !ok {
		return
	}
	list, ok := m["interface"].([]interface{})
	assert.True(t, ok, "应有小写 yang 键 interface 的数组")
	assert.Len(t, list, 1)
	first, _ := list[0].(map[string]interface{})
	assert.Equal(t, "GigabitEthernet0/0/2", first["name"], "回读应含新增接口的 yang 命名字段")
}
