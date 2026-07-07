package client

import (
	"testing"
	"time"
)

// TestNewNETCONFClient_NoCredentialFallback: credentials come from the shared
// DeviceStore, not a hardcoded fallback. Empty credentials must stay empty so an
// unregistered device fails authentication cleanly instead of silently using
// admin/admin.
func TestNewNETCONFClient_NoCredentialFallback(t *testing.T) {
	c, _ := NewNETCONFClient(DeviceConnectionInfo{IP: "192.0.2.1", Timeout: 10 * time.Millisecond})
	if c == nil {
		t.Fatal("client must be returned even on connect failure")
	}
	if c.info.Username != "" || c.info.Password != "" {
		t.Fatalf("empty credentials must NOT be defaulted, got %q/%q", c.info.Username, c.info.Password)
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
