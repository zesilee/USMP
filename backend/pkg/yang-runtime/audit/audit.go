// Package audit records config-delivery operations as an append-only log so
// they can be surfaced via the API (操作日志). It is an in-memory,
// concurrency-safe store (R09) with best-effort persistence to a local JSON
// file (§8 本地 JSON 元信息) — no database (R03). Missing/corrupt files degrade
// to an empty log rather than crashing (R08).
//
// Honesty: only fields with a truthful source at push time are recorded. The
// reconcile outcome is NOT stored here (it is joined live from status.Reader at
// query time, since it changes asynchronously after the push). There is no
// authenticated user in the backend, so Actor defaults to "system".
package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
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

// Store is an in-memory, concurrency-safe audit log with best-effort JSON
// persistence. Bounded to maxRecords newest entries. Zero value unusable;
// construct with NewStore.
type Store struct {
	mu         sync.RWMutex
	records    []Record // oldest → newest
	maxRecords int
	filePath   string // "" → memory only (no persistence)
	seq        uint64
}

// NewStore creates an audit store bounded to maxRecords, persisting to filePath
// ("" for memory-only). Any existing file is loaded; a missing or corrupt file
// degrades to an empty log (R08).
func NewStore(filePath string, maxRecords int) *Store {
	if maxRecords <= 0 {
		maxRecords = 1000
	}
	s := &Store{records: make([]Record, 0), maxRecords: maxRecords, filePath: filePath}
	s.load()
	return s
}

// Record appends an entry, assigning ID/Timestamp/Actor defaults, trimming to
// the bound, and persisting best-effort (a write failure never breaks a push).
func (s *Store) Record(r Record) {
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
	s.persistLocked() // best-effort; error ignored so a push never fails on I/O
	s.mu.Unlock()
}

// List returns every record, newest-first (copy).
func (s *Store) List() []Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return reversed(s.records)
}

// ListByDevice returns records for a single device, newest-first.
func (s *Store) ListByDevice(ip string) []Record {
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

// Flush persists the current log to disk (used on shutdown). No-op when
// memory-only. Returns any write error for callers that care.
func (s *Store) Flush() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.persistLocked()
}

func reversed(in []Record) []Record {
	out := make([]Record, len(in))
	for i, r := range in {
		out[len(in)-1-i] = r
	}
	return out
}

// persistLocked writes the log atomically (temp + rename). Caller holds the
// lock. Best-effort: a nil filePath or write error is not fatal.
func (s *Store) persistLocked() error {
	if s.filePath == "" {
		return nil
	}
	data, err := json.Marshal(s.records)
	if err != nil {
		return err
	}
	if dir := filepath.Dir(s.filePath); dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0o755) // best-effort：目录已存在或权限不足时下方写入自会报错
	}
	tmp := s.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.filePath)
}

// load reads an existing log file, restoring records and the ID sequence.
// A missing or corrupt file leaves the store empty (R08).
func (s *Store) load() {
	if s.filePath == "" {
		return
	}
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return // missing file → empty log
	}
	var recs []Record
	if err := json.Unmarshal(data, &recs); err != nil {
		return // corrupt file → empty log (do not crash)
	}
	if len(recs) > s.maxRecords {
		recs = recs[len(recs)-s.maxRecords:]
	}
	s.records = recs
	// Continue the ID sequence past the highest loaded numeric ID so restart
	// records never collide with persisted ones.
	for _, r := range recs {
		if n, perr := strconv.ParseUint(r.ID, 10, 64); perr == nil && n > s.seq {
			s.seq = n
		}
	}
}
