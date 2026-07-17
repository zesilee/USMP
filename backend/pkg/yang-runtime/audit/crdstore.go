package audit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	toolscache "k8s.io/client-go/tools/cache"
	crcache "sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	usmpv1 "github.com/leezesi/usmp/backend/api/core/v1"
)

// DeviceIPLabel 标记 AuditRecord CR 所属设备 IP（筛选用，OA-02）。
const DeviceIPLabel = "usmp.io/device-ip"

// CRDStoreScheme returns the runtime scheme for AuditRecord CRs.
func CRDStoreScheme() (*runtime.Scheme, error) {
	s := runtime.NewScheme()
	if err := usmpv1.AddToScheme(s); err != nil {
		return nil, err
	}
	return s, nil
}

// crdStore 是集群模式的审计 Store（OA-01/02/03）：每条记录一个 AuditRecord CR，
// 进程内 watch 镜像承接查询（GET /logs 零 apiserver RTT），写入异步落 CR
// （失败仅记日志不阻断下发），超上限由任意副本按时间幂等清理。
type crdStore struct {
	ctx        context.Context
	ns         string
	writer     ctrlclient.Client
	maxRecords int

	mu   sync.RWMutex
	recs map[string]Record // keyed by ID (= CR name)

	cleanupMu sync.Mutex // 串行化清理（去抖）
}

// NewCRDStore 构建集群模式审计 store：起 AuditRecord informer 并等待首次同步
// （重启保留即来自这次 list）。同步失败返回错误（装配方降级内存实现，R08）。
func NewCRDStore(ctx context.Context, cfg *rest.Config, namespace string, maxRecords int) (Store, error) {
	if maxRecords <= 0 {
		maxRecords = 1000
	}
	scheme, err := CRDStoreScheme()
	if err != nil {
		return nil, fmt.Errorf("audit crdstore scheme: %w", err)
	}
	writer, err := ctrlclient.New(cfg, ctrlclient.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("audit crdstore client: %w", err)
	}
	ca, err := crcache.New(cfg, crcache.Options{
		Scheme:            scheme,
		DefaultNamespaces: map[string]crcache.Config{namespace: {}},
	})
	if err != nil {
		return nil, fmt.Errorf("audit crdstore cache: %w", err)
	}

	s := &crdStore{
		ctx:        ctx,
		ns:         namespace,
		writer:     writer,
		maxRecords: maxRecords,
		recs:       make(map[string]Record),
	}

	inf, err := ca.GetInformer(ctx, &usmpv1.AuditRecord{})
	if err != nil {
		return nil, fmt.Errorf("audit crdstore informer: %w", err)
	}
	if _, err := inf.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { s.applyEvent(obj) },
		UpdateFunc: func(_, obj interface{}) { s.applyEvent(obj) },
		DeleteFunc: func(obj interface{}) { s.removeEvent(obj) },
	}); err != nil {
		return nil, fmt.Errorf("audit crdstore event handler: %w", err)
	}

	go func() {
		if err := ca.Start(ctx); err != nil {
			log.Printf("[audit] crdstore cache stopped: %v", err)
		}
	}()
	syncCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if !ca.WaitForCacheSync(syncCtx) {
		return nil, fmt.Errorf("audit crdstore cache sync timeout (namespace %q)", namespace)
	}
	return s, nil
}

func (s *crdStore) applyEvent(obj interface{}) {
	cr, ok := obj.(*usmpv1.AuditRecord)
	if !ok {
		return
	}
	s.mu.Lock()
	s.recs[cr.Name] = recordFromCR(cr)
	overflow := len(s.recs) > s.maxRecords
	s.mu.Unlock()
	if overflow {
		// 任意副本看到超限都可清理（OA-03 幂等，跨副本收敛）。
		go s.cleanup()
	}
}

func (s *crdStore) removeEvent(obj interface{}) {
	if tomb, ok := obj.(toolscache.DeletedFinalStateUnknown); ok {
		obj = tomb.Obj
	}
	cr, ok := obj.(*usmpv1.AuditRecord)
	if !ok {
		return
	}
	s.mu.Lock()
	delete(s.recs, cr.Name)
	s.mu.Unlock()
}

// Record implements Store：分配 ID/缺省值 → 写穿镜像（查询即时可见）→ 异步落
// CR + 超限清理（持久化失败仅记日志，不阻断下发，OA-01）。
func (s *crdStore) Record(r Record) {
	if r.Timestamp.IsZero() {
		r.Timestamp = time.Now()
	}
	if r.Actor == "" {
		r.Actor = "system"
	}
	r.ID = auditCRName(r.Timestamp)

	s.mu.Lock()
	s.recs[r.ID] = r
	s.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
		defer cancel()
		cr := &usmpv1.AuditRecord{
			ObjectMeta: metav1.ObjectMeta{
				Name:      r.ID,
				Namespace: s.ns,
				Labels:    map[string]string{DeviceIPLabel: r.DeviceIP},
			},
			Spec: usmpv1.AuditRecordSpec{
				Timestamp:    metav1.NewTime(r.Timestamp),
				DeviceIP:     r.DeviceIP,
				Path:         r.Path,
				Summary:      r.Summary,
				Triggered:    r.Triggered,
				Actor:        r.Actor,
				Forced:       r.Forced,
				ForcedOwners: append([]string(nil), r.ForcedOwners...),
			},
		}
		if err := s.writer.Create(ctx, cr); err != nil && !apierrors.IsAlreadyExists(err) {
			// best-effort：失败不阻断下发，镜像内仍可查（进程内可见，OA-01）。
			log.Printf("[audit] persist AuditRecord %s failed, keeping in-memory only: %v", r.ID, err)
		}
		s.cleanup()
	}()
}

// cleanup 按时间删除超出上限的最旧 CR（NotFound 容忍幂等，OA-03）。
func (s *crdStore) cleanup() {
	s.cleanupMu.Lock()
	defer s.cleanupMu.Unlock()

	s.mu.RLock()
	all := s.sortedLocked() // newest-first
	s.mu.RUnlock()
	if len(all) <= s.maxRecords {
		return
	}
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()
	for _, victim := range all[s.maxRecords:] {
		cr := &usmpv1.AuditRecord{}
		cr.Namespace, cr.Name = s.ns, victim.ID
		if err := s.writer.Delete(ctx, cr); err != nil && !apierrors.IsNotFound(err) {
			log.Printf("[audit] cleanup %s: %v", victim.ID, err)
			continue
		}
		s.mu.Lock()
		delete(s.recs, victim.ID)
		s.mu.Unlock()
	}
}

// List implements Store（镜像查询，newest-first，OA-04）。
func (s *crdStore) List() []Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := s.sortedLocked()
	if len(out) > s.maxRecords {
		out = out[:s.maxRecords]
	}
	return out
}

// ListByDevice implements Store.
func (s *crdStore) ListByDevice(ip string) []Record {
	out := make([]Record, 0)
	for _, r := range s.List() {
		if r.DeviceIP == ip {
			out = append(out, r)
		}
	}
	return out
}

// Flush implements Store（CR 写入即持久，无落盘语义）。
func (s *crdStore) Flush() error { return nil }

// sortedLocked returns all records newest-first（调用方持读锁或写锁）。
func (s *crdStore) sortedLocked() []Record {
	out := make([]Record, 0, len(s.recs))
	for _, r := range s.recs {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Timestamp.Equal(out[j].Timestamp) {
			return out[i].Timestamp.After(out[j].Timestamp)
		}
		return out[i].ID > out[j].ID
	})
	return out
}

func recordFromCR(cr *usmpv1.AuditRecord) Record {
	return Record{
		ID:           cr.Name,
		Timestamp:    cr.Spec.Timestamp.Time,
		DeviceIP:     cr.Spec.DeviceIP,
		Path:         cr.Spec.Path,
		Summary:      cr.Spec.Summary,
		Triggered:    cr.Spec.Triggered,
		Actor:        cr.Spec.Actor,
		Forced:       cr.Spec.Forced,
		ForcedOwners: append([]string(nil), cr.Spec.ForcedOwners...),
	}
}

// auditCRName 生成时间有序且跨副本唯一的 CR 名（ID=名字）。
func auditCRName(ts time.Time) string {
	suffix := make([]byte, 2)
	_, _ = rand.Read(suffix)
	return fmt.Sprintf("audit-%019d-%s", ts.UnixNano(), hex.EncodeToString(suffix))
}
