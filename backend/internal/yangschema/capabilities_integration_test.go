package yangschema

import (
	"strings"
	"testing"
	"time"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
	"github.com/leezesi/usmp/backend/simulator/netconfsim"
)

func connectTo(t *testing.T, sim *netconfsim.Simulator) *client.NETCONFClient {
	t.Helper()
	c, err := client.NewNETCONFClient(client.DeviceConnectionInfo{
		IP:       sim.Addr(),
		Port:     sim.Port(),
		Username: sim.Username(),
		Password: sim.Password(),
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("connect to simulator: %v", err)
	}
	return c
}

func moduleNameSet(mods []schema.Module) map[string]bool {
	set := make(map[string]bool, len(mods))
	for _, m := range mods {
		set[m.Name()] = true
	}
	return set
}

// TestCapabilitiesNarrowingIntegration (task 1.5): a simulator advertising a
// huawei-vlan module capability, read via the real NETCONF client, narrows the
// loaded module set to the matching module.
func TestCapabilitiesNarrowingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim := netconfsim.NewSimulator()
	sim.SetCapabilities([]string{"urn:huawei:yang:huawei-vlan?module=huawei-vlan&revision=2021-01-01"})
	if err := sim.Start(); err != nil {
		t.Fatalf("start simulator: %v", err)
	}
	defer sim.Stop()

	c := connectTo(t, sim)
	defer c.Close()

	caps := c.ServerCapabilities()
	found := false
	for _, cp := range caps {
		if strings.Contains(cp, "huawei-vlan") {
			found = true
		}
	}
	if !found {
		t.Fatalf("advertised huawei-vlan capability not read back: %v", caps)
	}

	all, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	narrowed := moduleNameSet(schema.NarrowModulesByCapabilities(caps, all.Modules()))
	if !narrowed["vlan"] {
		t.Fatalf("expected 'vlan' module in narrowed set, got %v", narrowed)
	}
	for _, excluded := range []string{"system", "interfaces", "ifm"} {
		if narrowed[excluded] {
			t.Errorf("module %q should be excluded by huawei-vlan capability", excluded)
		}
	}
}

// TestCapabilitiesFallbackIntegration (task 1.3 fallback): a simulator advertising
// only base capabilities yields all loaded modules (model tree authoritative).
func TestCapabilitiesFallbackIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sim := netconfsim.NewSimulator() // default: base caps only, no YANG-module caps
	if err := sim.Start(); err != nil {
		t.Fatalf("start simulator: %v", err)
	}
	defer sim.Stop()

	c := connectTo(t, sim)
	defer c.Close()

	all, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	narrowed := schema.NarrowModulesByCapabilities(c.ServerCapabilities(), all.Modules())
	if len(narrowed) != len(all.Modules()) {
		t.Fatalf("base-only caps should fall back to all %d modules, got %d",
			len(all.Modules()), len(narrowed))
	}
}
