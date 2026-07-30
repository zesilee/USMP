package rpcrisk

import "testing"

// 表格驱动：高危动词命中 / 温和 reset-clear 家族不命中 / 词边界（reset 不匹配 restart）。
func TestIsHighRisk(t *testing.T) {
	cases := map[string]bool{
		"restart-if":                true,
		"reboot-system":             true,
		"rollback-config":           true,
		"batch-delete-users":        true,
		"upgrade-firmware":          true,
		"undo-config":               true,
		"reset-if-counters-by-name": false,
		"reset-if-counters-all":     false,
		"clear-arp":                 false,
		"ping-host":                 false,
		// 词边界：restarting 不是 restart 段、reset 不含 restart。
		"restarting":  false,
		"reset":       false,
		"if-restart":  true,
		"powersave":   false,
		"power-cycle": true,
		"":            false,
	}
	for name, want := range cases {
		if got := IsHighRisk(name); got != want {
			t.Errorf("IsHighRisk(%q) = %v, want %v", name, got, want)
		}
	}
}
