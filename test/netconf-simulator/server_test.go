package netsim

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServer_New(t *testing.T) {
	cfg := &ServerConfig{
		Addr:     "127.0.0.1",
		Port:     1830,
		Username: "admin",
		Password: "admin",
	}

	s := New(cfg)
	assert.NotNil(t, s)
	assert.Equal(t, "127.0.0.1", s.config.Addr)
	assert.Equal(t, 1830, s.config.Port)
	assert.Equal(t, "admin", s.config.Username)
	assert.Equal(t, "admin", s.config.Password)
	assert.NotNil(t, s.datastore)
	assert.False(t, s.IsRunning())
}

func TestServer_StartStop(t *testing.T) {
	cfg := &ServerConfig{
		Addr:     "127.0.0.1",
		Port:     0, // 随机端口
		Username: "admin",
		Password: "admin",
	}

	s := New(cfg)
	err := s.Start()
	assert.NoError(t, err)
	assert.True(t, s.IsRunning())
	assert.Positive(t, s.Port())

	err = s.Stop()
	assert.NoError(t, err)
	assert.False(t, s.IsRunning())
}

func TestServer_Start_AddrInUse(t *testing.T) {
	cfg1 := &ServerConfig{
		Addr:     "127.0.0.1",
		Port:     0,
		Username: "admin",
		Password: "admin",
	}
	s1 := New(cfg1)
	err := s1.Start()
	assert.NoError(t, err)
	defer s1.Stop()

	// Try to start another server on the same port
	cfg2 := &ServerConfig{
		Addr:     "127.0.0.1",
		Port:     s1.Port(),
		Username: "admin",
		Password: "admin",
	}
	s2 := New(cfg2)
	err = s2.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "address already in use")
}
