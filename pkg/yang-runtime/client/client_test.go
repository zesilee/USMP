package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDeviceConnectionInfo(t *testing.T) {
	info := DeviceConnectionInfo{
		IP:       "192.168.1.1",
		Port:     830,
		Username: "admin",
		Password: "admin",
		Protocol: ProtocolNETCONF,
	}

	assert.Equal(t, "192.168.1.1", info.IP)
	assert.Equal(t, 830, info.Port)
}

func TestDefaultClientPool(t *testing.T) {
	factory := DefaultClientFactory(10 * time.Second)
	p := NewDefaultClientPool(factory)
	stats := p.Stats()
	assert.Equal(t, 0, stats.ActiveConnections)
}

func TestGetOptions(t *testing.T) {
	opts := &GetOptions{}
	WithDatastore("candidate").Apply(opts)
	assert.Equal(t, "candidate", opts.Datastore)

	WithTimeout(5 * time.Second).Apply(opts)
	assert.Equal(t, 5*time.Second, opts.Timeout)
}

func TestSetOptions(t *testing.T) {
	opts := &SetOptions{}
	WithCommit(true).Apply(opts)
	assert.True(t, opts.Commit)

	WithCommit(false).Apply(opts)
	assert.False(t, opts.Commit)
}

func TestChangeTypeString(t *testing.T) {
	assert.Equal(t, "ADD", AddChange.String())
	assert.Equal(t, "DELETE", DeleteChange.String())
	assert.Equal(t, "MODIFY", ModifyChange.String())
}
