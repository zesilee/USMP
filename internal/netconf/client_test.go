package netconf

import (
	"testing"

	"github.com/leezesi/usmp/internal/actor"
	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	device := actor.DeviceInfo{
		IP:       "192.168.1.1",
		Port:     830,
		Username: "admin",
		Password: "admin",
	}

	client := NewClient(device)
	assert.NotNil(t, client)
	assert.False(t, client.IsConnected())
}

func TestConstructGetConfigFilter(t *testing.T) {
	filter := ConstructGetConfigFilter("/interfaces")
	assert.Contains(t, filter, "interfaces")
	assert.Contains(t, filter, "filter")

	filter = ConstructGetConfigFilter("/vlans")
	assert.Contains(t, filter, "vlans")

	filter = ConstructGetConfigFilter("/unknown")
	assert.Contains(t, filter, "<filter/>")
}
