// Command schemadump exports the nested presentation schema of every loaded
// YANG module to a versioned JSON fixture (one file per module). The fixtures
// are the shared, checked-in source of truth for the frontend console-derivation
// golden suite and (later) the device-consistency matrix — a pure function of
// the backend schema, produced without HTTP, docker or a running server.
//
// Usage:
//
//	go run ./tools/schemadump -output=testdata/schema-fixtures
//
// The output directory is treated as a generated artifact: existing *.json are
// removed before writing so the directory is a pure function of the schema
// (a module removed upstream leaves no stale fixture behind). CI verifies zero
// drift via regen-and-diff (SF-04).
package main

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/leezesi/usmp/backend/internal/api"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// exportAll returns module-name → indented-JSON fixture for every loaded module
// with a non-nil root (the exact set ListModules exposes over HTTP). Each fixture
// is a pure function of that module's schema via api.BuildYangSchemaNested, so
// the map is deterministic: keys are independent and values carry no ordering
// beyond the schema's own (already sorted) child order.
func exportAll(s schema.Schema) (map[string][]byte, error) {
	out := make(map[string][]byte)
	for _, mod := range s.Modules() {
		if mod.Root() == nil {
			continue
		}
		ys := api.BuildYangSchemaNested(mod)
		b, err := json.MarshalIndent(ys, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal %s: %w", mod.Name(), err)
		}
		// 末尾换行：贴合 POSIX 文本文件惯例，diff 更干净。
		out[mod.Name()] = append(b, '\n')
	}
	return out, nil
}

// sortedNames returns the fixture module names in stable order for deterministic
// aggregate output (logging, index files).
func sortedNames(fixtures map[string][]byte) []string {
	names := make([]string, 0, len(fixtures))
	for n := range fixtures {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
