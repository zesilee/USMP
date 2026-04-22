package netsim

import (
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRPCMessage_Unmarshal(t *testing.T) {
	input := `
<rpc message-id="1">
  <get-config>
    <source>
      <running/>
    </source>
    <filter type="subtree">
      <vlans xmlns="http://openconfig.net/yang/vlan"/>
    </filter>
  </get-config>
</rpc>`

	var rpc RPCMessage
	err := xml.Unmarshal([]byte(input), &rpc)
	assert.NoError(t, err)
	assert.Equal(t, "1", rpc.MessageID)
	assert.Len(t, rpc.Operations, 1)
	assert.Equal(t, "get-config", rpc.Operations[0].XMLName.Local)
	assert.Contains(t, string(rpc.Operations[0].Content), "<source>")
}

func TestGetConfigRequest_Unmarshal(t *testing.T) {
	input := `
<get-config>
  <source>
    <running/>
  </source>
  <filter type="subtree">
    <vlans/>
  </filter>
</get-config>`

	var req GetConfigRequest
	err := xml.Unmarshal([]byte(input), &req)
	assert.NoError(t, err)
	assert.NotNil(t, req.Source.Running)
	assert.Equal(t, "subtree", req.Filter.Type)
	assert.Contains(t, string(req.Filter.Content), "vlans")
}

func TestGetConfigRequest_UnmarshalCandidate(t *testing.T) {
	input := `
<get-config>
  <source>
    <candidate/>
  </source>
</get-config>`

	var req GetConfigRequest
	err := xml.Unmarshal([]byte(input), &req)
	assert.NoError(t, err)
	assert.Nil(t, req.Source.Running)
	assert.NotNil(t, req.Source.Candidate)
}
