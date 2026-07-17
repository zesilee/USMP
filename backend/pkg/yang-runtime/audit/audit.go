// Package audit records config-delivery operations as an append-only log so
// they can be surfaced via the API (操作日志). Two backends implement Store:
// an in-memory, concurrency-safe log (R09, 无集群降级) and an AuditRecord CRD
// log (OA-02: 每条一 CR、跨副本可见、跨重启存活). 本地文件持久化已退役
// （OA-05/SC-06）——传入文件路径仅产生弃用警告。No database (R03).
//
// Honesty: only fields with a truthful source at push time are recorded. The
// reconcile outcome is NOT stored here (it is joined live from status.Reader at
// query time, since it changes asynchronously after the push). There is no
// authenticated user in the backend, so Actor defaults to "system".
package audit

import (
	"log"
	"strconv"
	"sync"
	"time"
)

// Record is a single config-delivery audit entry.
type Record struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	DeviceIP  string    `json:"device_ip"`
	Path      string    `json:"path"`
	Summary   string    `json:"summary"`   // 提交内容摘要（如 list keys）
	Triggered bool      `json:"triggered"` // 是否有 controller 接管对账
	Actor     string    `json:"actor"`     // 无鉴权来源，默认 "system"
	// Forced/ForcedOwners：force 覆盖归属硬锁的留痕（OA-01 二期）——谁在哪条
	// 认领路径上显式覆盖了哪些意图的认领。缺省零值=普通下发。
	Forced       bool     `json:"forced,omitempty"`
	ForcedOwners []string `json:"forcedOwners,omitempty"`
}

// Recorder is the write side (SetConfig records after a successful push).
type Recorder interface {
	Record(r Record)
}

// Reader is the read-only view handed to API handlers.
type Reader interface {
	List() []Record // newest-first
	ListByDevice(ip string) []Record
}

// Store 是操作审计日志的统一接口（内存降级实现与 AuditRecord CRD 实现共用，
// GET /logs 契约只依赖此四方法）。
type Store interface {
	Recorder
	Reader
	// Flush 留给需要落盘语义的实现（内存/CRD 实现恒 nil）。
	Flush() error
}

// memStore is an in-memory, concurrency-safe audit log bounded to maxRecords
// newest entries（无集群降级路径，重启即丢）。
type memStore struct {
	mu         sync.RWMutex
	records    []Record // oldest → newest
	maxRecords int
	seq        uint64
}

// NewStore creates an in-memory audit store bounded to maxRecords. filePath
// 已退役（OA-05/SC-06 禁本地持久）：非空仅产生弃用警告，不读不写任何文件。
func NewStore(filePath string, maxRecords int) Store {
	if filePath != "" {
		log.Printf("[audit] USMP_AUDIT_FILE/本地审计文件已退役（SC-06），忽略 %q：集群模式走 AuditRecord CRD，无集群为纯内存", filePath)
	}
	return NewMemStore(maxRecords)
}

// NewMemStore creates the in-memory Store implementation.
func NewMemStore(maxRecords int) Store {
	if maxRecords <= 0 {
		maxRecords = 1000
	}
	return &memStore{records: make([]Record, 0), maxRecords: maxRecords}
}

// Record appends an entry, assigning ID/Timestamp/Actor defaults, trimming to
// the bound, and persisting best-effort (a write failure never breaks a push).
func (s *memStore) Record(r Record) {
	s.mu.Lock()
	s.seq++
	r.ID = strconv.FormatUint(s.seq, 10)
	if r.Timestamp.IsZero() {
		r.Timestamp = time.Now()
	}
	if r.Actor == "" {
		r.Actor = "system"
	}
	s.records = append(s.records, r)
	if len(s.records) > s.maxRecords {
		s.records = append(s.records[:0], s.records[len(s.records)-s.maxRecords:]...)
	}
	s.mu.Unlock()
}

// List returns every record, newest-first (copy).
func (s *memStore) List() []Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return reversed(s.records)
}

// ListByDevice returns records for a single device, newest-first.
func (s *memStore) ListByDevice(ip string) []Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Record, 0)
	for i := len(s.records) - 1; i >= 0; i-- {
		if s.records[i].DeviceIP == ip {
			out = append(out, s.records[i])
		}
	}
	return out
}

// Flush implements Store（内存模式无落盘语义，恒 nil）。
func (s *memStore) Flush() error { return nil }

func reversed(in []Record) []Record {
	out := make([]Record, len(in))
	for i, r := range in {
		out[len(in)-1-i] = r
	}
	return out
}
