package audit

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func rec(ip, path, summary string) Record {
	return Record{DeviceIP: ip, Path: path, Summary: summary, Triggered: true}
}

func TestRecord_AssignsIDTimestampActor(t *testing.T) {
	s := NewStore("", 100)
	s.Record(rec("10.0.0.1", "/vlan", "vlans: [100]"))

	got := s.List()
	assert.Len(t, got, 1)
	assert.NotEmpty(t, got[0].ID)
	assert.False(t, got[0].Timestamp.IsZero(), "缺省时间戳应自动打上")
	assert.Equal(t, "system", got[0].Actor, "无鉴权来源默认 system")
}

func TestRecord_PreservesCallerActorAndTimestamp(t *testing.T) {
	s := NewStore("", 100)
	ts := time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)
	s.Record(Record{DeviceIP: "1", Path: "/p", Actor: "alice", Timestamp: ts})
	got := s.List()[0]
	assert.Equal(t, "alice", got.Actor)
	assert.Equal(t, ts, got.Timestamp)
}

func TestList_NewestFirst(t *testing.T) {
	s := NewStore("", 100)
	s.Record(rec("a", "/p", "1"))
	s.Record(rec("b", "/p", "2"))
	s.Record(rec("c", "/p", "3"))
	got := s.List()
	assert.Equal(t, []string{"c", "b", "a"}, []string{got[0].DeviceIP, got[1].DeviceIP, got[2].DeviceIP})
}

func TestIDs_MonotonicUnique(t *testing.T) {
	s := NewStore("", 100)
	for i := 0; i < 5; i++ {
		s.Record(rec("a", "/p", "x"))
	}
	seen := map[string]bool{}
	for _, r := range s.List() {
		assert.False(t, seen[r.ID], "ID 应唯一")
		seen[r.ID] = true
	}
	assert.Len(t, seen, 5)
}

func TestListByDevice_Filters(t *testing.T) {
	s := NewStore("", 100)
	s.Record(rec("10.0.0.1", "/p", "1"))
	s.Record(rec("10.0.0.2", "/p", "2"))
	s.Record(rec("10.0.0.1", "/q", "3"))
	got := s.ListByDevice("10.0.0.1")
	assert.Len(t, got, 2)
	for _, r := range got {
		assert.Equal(t, "10.0.0.1", r.DeviceIP)
	}
}

func TestMaxRecords_DropsOldest(t *testing.T) {
	s := NewStore("", 3)
	for i := 0; i < 5; i++ {
		s.Record(rec("a", "/p", string(rune('1'+i))))
	}
	got := s.List()
	assert.Len(t, got, 3, "超界只保留最新 3 条")
	assert.Equal(t, "5", got[0].Summary) // 最新
	assert.Equal(t, "3", got[2].Summary) // 最旧保留
}

func TestPersistence_SurvivesReload(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "audit.json")

	s1 := NewStore(fp, 100)
	s1.Record(rec("10.0.0.1", "/vlan", "vlans: [100]"))
	s1.Record(rec("10.0.0.2", "/ifm", "interface: [GE0/0/1]"))

	// 模拟重启：新 Store 从同一文件加载
	s2 := NewStore(fp, 100)
	got := s2.List()
	assert.Len(t, got, 2)
	assert.Equal(t, "/ifm", got[0].Path) // 仍新最先

	// 加载后继续记录，ID 不与已加载记录冲突
	s2.Record(rec("10.0.0.3", "/vlan", "x"))
	ids := map[string]bool{}
	for _, r := range s2.List() {
		assert.False(t, ids[r.ID], "重启续记 ID 不应与已加载冲突")
		ids[r.ID] = true
	}
}

func TestPersistence_EmptyPathIsMemoryOnly(t *testing.T) {
	s := NewStore("", 100)
	s.Record(rec("a", "/p", "x"))
	assert.Len(t, s.List(), 1) // 不崩、无文件
}

func TestLoad_MissingFileDegradesEmpty(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "nope.json"), 100)
	assert.Empty(t, s.List(), "文件不存在应降级为空、不报错(R08)")
	s.Record(rec("a", "/p", "x"))
	assert.Len(t, s.List(), 1)
}

func TestLoad_CorruptFileDegradesEmpty(t *testing.T) {
	fp := filepath.Join(t.TempDir(), "audit.json")
	assert.NoError(t, os.WriteFile(fp, []byte("{not json"), 0o600))
	s := NewStore(fp, 100) // 不应 panic
	assert.Empty(t, s.List())
	// 损坏文件仍可继续记录并覆盖
	s.Record(rec("a", "/p", "x"))
	assert.Len(t, s.List(), 1)
}

func TestConcurrent_RecordAndList(t *testing.T) {
	// Run with -race
	s := NewStore("", 1000)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				s.Record(rec("a", "/p", "x"))
				_ = s.List()
				_ = s.ListByDevice("a")
			}
		}()
	}
	wg.Wait()
	assert.Len(t, s.List(), 400)
}
