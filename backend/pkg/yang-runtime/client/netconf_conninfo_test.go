package client

import (
	"testing"
	"time"
)

// TestNewNETCONFClient_DefaultsCredentials: when connection info carries no
// username/password (e.g. reconcile is triggered with a bare device IP and the
// credential store is not shared), the client must fall back to admin/admin so
// SSH offers password auth instead of only "none". The immediate connect fails
// (no real device) but info defaults are applied before connecting.
func TestNewNETCONFClient_DefaultsCredentials(t *testing.T) {
	c, _ := NewNETCONFClient(DeviceConnectionInfo{IP: "192.0.2.1", Timeout: 10 * time.Millisecond})
	if c == nil {
		t.Fatal("client must be returned even on connect failure")
	}
	if c.info.Username != "admin" || c.info.Password != "admin" {
		t.Fatalf("empty credentials must default to admin/admin, got %q/%q", c.info.Username, c.info.Password)
	}
}

// TestNewNETCONFClient_KeepsExplicitCredentials: explicit credentials must never
// be overwritten by the fallback.
func TestNewNETCONFClient_KeepsExplicitCredentials(t *testing.T) {
	c, _ := NewNETCONFClient(DeviceConnectionInfo{IP: "192.0.2.1", Username: "bob", Password: "s3cret", Timeout: 10 * time.Millisecond})
	if c.info.Username != "bob" || c.info.Password != "s3cret" {
		t.Fatalf("explicit credentials must be preserved, got %q/%q", c.info.Username, c.info.Password)
	}
}
