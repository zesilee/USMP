package netsim

import (
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHelloMessage_Marshal(t *testing.T) {
	hello := HelloMessage{
		Capabilities: []Capability{
			{URI: "urn:ietf:params:netconf:base:1.0"},
			{URI: "urn:ietf:params:netconf:base:1.1"},
		},
	}

	data, err := xml.MarshalIndent(hello, "", "  ")
	assert.NoError(t, err)
	assert.Contains(t, string(data), "<hello")
	assert.Contains(t, string(data), "<capabilities")
	assert.Contains(t, string(data), "urn:ietf:params:netconf:base:1.0")
	assert.Contains(t, string(data), "urn:ietf:params:netconf:base:1.1")
}

func TestHelloMessage_Unmarshal(t *testing.T) {
	input := `<?xml version="1.0" encoding="UTF-8"?>
<hello>
  <capabilities>
    <capability>urn:ietf:params:netconf:base:1.0</capability>
    <capability>urn:ietf:params:netconf:capability:candidate:1.0</capability>
  </capabilities>
</hello>`

	var hello HelloMessage
	err := xml.Unmarshal([]byte(input), &hello)
	assert.NoError(t, err)
	assert.Len(t, hello.Capabilities, 2)
	assert.Equal(t, "urn:ietf:params:netconf:base:1.0", hello.Capabilities[0].URI)
}

func TestNegotiateFraming_Base11(t *testing.T) {
	server := New(&ServerConfig{})
	session := NewSession(server, nil)

	clientHello := &HelloMessage{
		Capabilities: []Capability{
			{URI: "urn:ietf:params:netconf:base:1.1"},
		},
	}

	session.negotiateFraming(clientHello)
	assert.Equal(t, Base11, session.framer.version)
}

func TestNegotiateFraming_Base10(t *testing.T) {
	server := New(&ServerConfig{})
	session := NewSession(server, nil)

	clientHello := &HelloMessage{
		Capabilities: []Capability{
			{URI: "urn:ietf:params:netconf:base:1.0"},
		},
	}

	session.negotiateFraming(clientHello)
	assert.Equal(t, Base10, session.framer.version)
}
