package intent

import (
	"sort"
	"strings"
	"sync"
)

// OwnershipIndex is the in-memory soft-ownership registry (BIO-07): intent CR
// key → its claims, aggregated from CR status by the reconciler. Purely a
// derived cache — rebuilt by watch/resync replay, no persistence (多实例安全：
// 每副本各自从 CR status 重建).
type OwnershipIndex struct {
	mu     sync.RWMutex
	claims map[string][]Claim
}

// DefaultOwnership is the process-wide index the config-api consults for
// manual-edit warnings (BR-11).
var DefaultOwnership = NewOwnershipIndex()

// NewOwnershipIndex builds an empty index.
func NewOwnershipIndex() *OwnershipIndex {
	return &OwnershipIndex{claims: map[string][]Claim{}}
}

// Replace swaps the claim set of one intent (called after status writeback).
func (ix *OwnershipIndex) Replace(intentKey string, claims []Claim) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	if len(claims) == 0 {
		delete(ix.claims, intentKey)
		return
	}
	ix.claims[intentKey] = append([]Claim{}, claims...)
}

// Remove drops an intent from the index (deletion lifecycle).
func (ix *OwnershipIndex) Remove(intentKey string) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	delete(ix.claims, intentKey)
}

// Owners returns the intent keys claiming anything at-or-under path on device
// (either direction of prefix containment, so a module-level write warns about
// entry-level claims and vice versa). Sorted, deduplicated; empty when free.
func (ix *OwnershipIndex) Owners(device, path string) []string {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	seen := map[string]bool{}
	for key, claims := range ix.claims {
		for _, c := range claims {
			if c.Device != device {
				continue
			}
			if strings.HasPrefix(c.Path, path) || strings.HasPrefix(path, c.Path) {
				seen[key] = true
				break
			}
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Claims returns all claims of one device (归属查询 API 数据面).
func (ix *OwnershipIndex) Claims(device string) map[string][]Claim {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	out := map[string][]Claim{}
	for key, claims := range ix.claims {
		for _, c := range claims {
			if c.Device == device {
				out[key] = append(out[key], c)
			}
		}
	}
	return out
}
