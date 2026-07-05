package manager

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestManager_RunningCache_Present(t *testing.T) {
	m := New()
	rc := m.GetRunningCache()
	assert.NotNil(t, rc)
	assert.Equal(t, 30*time.Second, rc.TTL(), "running-config cache TTL must be 30s (§8)")
}

func TestManager_InterfaceExposesRunningCache(t *testing.T) {
	var m Manager = New()
	assert.NotNil(t, m.GetRunningCache())
}

// running cache and desired ConfigStore must be independent instances (different
// TTLs / lifetimes), so caching running reads never touches desired state.
func TestManager_RunningCache_SeparateFromConfigStore(t *testing.T) {
	m := New()
	m.GetRunningCache().Set("10.0.0.1|/vlans", "running-value")
	// desired store is empty for this key
	v, err := m.GetConfigStore().Get("10.0.0.1", "/vlans")
	assert.NoError(t, err)
	assert.Nil(t, v, "running cache write must not leak into desired ConfigStore")
}

func TestManager_Stop_NoPanic(t *testing.T) {
	m := New()
	assert.NoError(t, m.Start(context.Background()))
	assert.NotPanics(t, func() { _ = m.Stop() })
}
