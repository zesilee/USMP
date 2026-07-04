package schema

import "strings"

// NarrowModulesByCapabilities implements the "capabilities define the module set"
// half of the hybrid schema source: given a device's advertised NETCONF hello
// capabilities and the full set of loaded modules (whose attribute schema comes
// from the ygot model tree), it returns the subset of modules the device
// supports.
//
// Fallback (per design): if the device advertises no YANG-module capabilities
// (only base netconf capabilities), all modules are returned — the schema model
// tree is the authority when the device does not narrow the set.
//
// Matching is heuristic (namespace equality, module= param, or name substring),
// which suits a schema-less narrowing without a full <get-schema> exchange.
func NarrowModulesByCapabilities(caps []string, mods []Module) []Module {
	yangCaps := make([]string, 0, len(caps))
	for _, c := range caps {
		if isYangModuleCapability(c) {
			yangCaps = append(yangCaps, c)
		}
	}
	if len(yangCaps) == 0 {
		return mods // no module capabilities advertised → model tree authoritative
	}
	out := make([]Module, 0, len(mods))
	for _, m := range mods {
		if moduleMatchesAnyCapability(m, yangCaps) {
			out = append(out, m)
		}
	}
	return out
}

// isYangModuleCapability reports whether a capability URI names a YANG module
// (rather than a base NETCONF capability such as base:1.0 / :candidate).
func isYangModuleCapability(c string) bool {
	return c != "" && !strings.HasPrefix(c, "urn:ietf:params:netconf:")
}

// capabilityNamespace returns the namespace portion of a capability URI (before
// any query string).
func capabilityNamespace(c string) string {
	if i := strings.IndexByte(c, '?'); i >= 0 {
		return c[:i]
	}
	return c
}

// capabilityModuleParam returns the value of the module= query parameter, if any.
func capabilityModuleParam(c string) string {
	i := strings.IndexByte(c, '?')
	if i < 0 {
		return ""
	}
	for _, kv := range strings.Split(c[i+1:], "&") {
		if v, ok := strings.CutPrefix(kv, "module="); ok {
			return v
		}
	}
	return ""
}

func moduleMatchesAnyCapability(m Module, caps []string) bool {
	ns := m.Namespace()
	name := m.Name()
	for _, c := range caps {
		if ns != "" && capabilityNamespace(c) == ns {
			return true
		}
		if mp := capabilityModuleParam(c); mp != "" && name != "" &&
			(mp == name || strings.Contains(mp, name) || strings.Contains(name, mp)) {
			return true
		}
		if name != "" && strings.Contains(c, name) {
			return true
		}
	}
	return false
}
