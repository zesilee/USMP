package intent

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	// 空白导入触发华为驱动描述符注册：marshalChange 的 XML 编码按值类型查
	// registry，未注册会静默 fallback 到 xml.Marshal 并在 map 字段上崩（SND 注册表
	// 的注册可达性约定）。
	_ "github.com/leezesi/usmp/backend/internal/drivers"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
)

// txClient narrows client.Client to the 2PC primitives (implemented by
// NETCONFClient, DP-04/DP-08).
type txClient interface {
	client.Client
	CommitConfirmed(ctx context.Context, timeout time.Duration) error
	ConfirmCommit(ctx context.Context) error
	DiscardCandidate(ctx context.Context) error
}

// TxCoordinator executes the cross-device two-phase push (BIO-03):
//
//	phase 1  edit-config → every device's candidate;   any failure → discard ALL
//	phase 2  confirmed-commit(timeout) on every device; failure → un-committed
//	         devices discard, committed ones roll back via the confirm timeout
//	phase 3  confirming commit on every device
//
// Devices lacking :confirmed-commit downgrade to a plain commit in phase 2 and
// are reported NonTransactional (DP-08). Per-device mutexes serialize
// concurrent intents touching the same device（单实例内；多实例下仅 leader 执行）.
type TxCoordinator struct {
	pool           client.ClientPool
	devices        device.Store
	confirmTimeout time.Duration

	mu       sync.Mutex
	devLocks map[string]*sync.Mutex
}

// NewTxCoordinator builds a coordinator over the shared ClientPool/DeviceStore.
func NewTxCoordinator(pool client.ClientPool, devices device.Store, confirmTimeout time.Duration) *TxCoordinator {
	if confirmTimeout <= 0 {
		confirmTimeout = 60 * time.Second // design open-question 初值，集成测试校准
	}
	return &TxCoordinator{pool: pool, devices: devices, confirmTimeout: confirmTimeout, devLocks: map[string]*sync.Mutex{}}
}

// Push implements Pusher.
func (t *TxCoordinator) Push(ctx context.Context, frags []Fragment) map[string]TxResult {
	byDev := map[string][]Fragment{}
	var devs []string
	for _, f := range frags {
		if _, ok := byDev[f.Device]; !ok {
			devs = append(devs, f.Device)
		}
		byDev[f.Device] = append(byDev[f.Device], f)
	}
	sort.Strings(devs) // 锁序=设备名序，跨意图并发无死锁

	unlock := t.lockAll(devs)
	defer unlock()

	failAll := func(reason string) map[string]TxResult {
		out := map[string]TxResult{}
		for _, d := range devs {
			out[d] = TxResult{Device: d, Err: errors.New(reason)}
		}
		return out
	}

	// 解析全部客户端（任一设备不可达即整体中止，未发任何配置）。
	clients := map[string]txClient{}
	for _, d := range devs {
		c, err := t.pool.Get(t.resolveConn(d))
		if err != nil {
			return failAll(fmt.Sprintf("transaction aborted: connect %s: %v", d, err))
		}
		tc, ok := c.(txClient)
		if !ok {
			return failAll(fmt.Sprintf("transaction aborted: device %s client (%T) lacks 2PC primitives", d, c))
		}
		clients[d] = tc
	}

	// Phase 1 — prepare all candidates.
	var prepared []string
	for _, d := range devs {
		if err := t.prepare(ctx, clients[d], byDev[d]); err != nil {
			t.discardAll(ctx, clients, append(prepared, d))
			return failAll(fmt.Sprintf("transaction aborted: prepare failed on %s: %v", d, err))
		}
		prepared = append(prepared, d)
	}

	// Phase 2 — confirmed-commit everywhere (downgrade to plain commit when the
	// capability is missing).
	nonTx := map[string]bool{}
	var confirmed []string
	for i, d := range devs {
		err := clients[d].CommitConfirmed(ctx, t.confirmTimeout)
		if err != nil && errors.Is(err, client.ErrConfirmedCommitUnsupported) {
			if perr := clients[d].ConfirmCommit(ctx); perr != nil {
				t.discardAll(ctx, clients, devs[i+1:])
				return failAll(fmt.Sprintf("transaction aborted: plain-commit fallback failed on %s: %v (confirmed devices roll back on timeout)", d, perr))
			}
			nonTx[d] = true
			continue
		}
		if err != nil {
			// 当前及后续设备 candidate 直接丢弃；已 confirmed-commit 的设备不发确认，
			// 依赖超时自动回滚（BIO-03 残余窗口，status 呈现）。
			t.discardAll(ctx, clients, devs[i:])
			return failAll(fmt.Sprintf("transaction aborted: confirmed-commit failed on %s: %v (earlier devices roll back on confirm timeout)", d, err))
		}
		confirmed = append(confirmed, d)
	}

	// Phase 3 — confirming commit.
	results := map[string]TxResult{}
	for _, d := range confirmed {
		if err := clients[d].ConfirmCommit(ctx); err != nil {
			results[d] = TxResult{Device: d, Err: fmt.Errorf("confirming commit failed (device rolls back on confirm timeout, transaction inconsistent): %w", err)}
			continue
		}
		results[d] = TxResult{Device: d}
	}
	for d := range nonTx {
		results[d] = TxResult{Device: d, NonTransactional: true}
	}
	return results
}

// prepare pushes one device's fragments into its candidate (no commit).
func (t *TxCoordinator) prepare(ctx context.Context, c txClient, frags []Fragment) error {
	for _, f := range frags {
		res, err := c.Set(ctx, []client.Change{{Type: client.AddChange, Path: f.Path, NewValue: f.Config}}, client.WithCommit(false))
		if err != nil {
			return err
		}
		if res != nil && !res.Success {
			return fmt.Errorf("edit-config rejected: %s", res.Message)
		}
	}
	return nil
}

// discardAll best-effort discards candidates on the named devices (R08 —
// discard errors are logged, not escalated: the candidate never reaches
// running without a commit).
func (t *TxCoordinator) discardAll(ctx context.Context, clients map[string]txClient, devs []string) {
	for _, d := range devs {
		c, ok := clients[d]
		if !ok {
			continue
		}
		if err := c.DiscardCandidate(ctx); err != nil {
			log.Printf("intent: discard candidate on %s: %v", d, err)
		}
	}
}

// resolveConn resolves connection info via the shared DeviceStore, degrading
// to an AUTO/no-credential connection for unregistered devices (PR#100 兜底,
// R08 — auth fails cleanly rather than crash).
func (t *TxCoordinator) resolveConn(deviceID string) client.DeviceConnectionInfo {
	if t.devices != nil {
		if info, ok := t.devices.Get(deviceID); ok {
			return info
		}
	}
	log.Printf("intent: device %q not registered in DeviceStore; using AUTO/no-credential connection", deviceID)
	return client.DeviceConnectionInfo{IP: deviceID, Protocol: client.ProtocolAUTO}
}

// lockAll acquires per-device mutexes in slice order and returns the combined
// unlock.
func (t *TxCoordinator) lockAll(devs []string) func() {
	locks := make([]*sync.Mutex, 0, len(devs))
	for _, d := range devs {
		t.mu.Lock()
		l, ok := t.devLocks[d]
		if !ok {
			l = &sync.Mutex{}
			t.devLocks[d] = l
		}
		t.mu.Unlock()
		l.Lock()
		locks = append(locks, l)
	}
	return func() {
		for i := len(locks) - 1; i >= 0; i-- {
			locks[i].Unlock()
		}
	}
}
