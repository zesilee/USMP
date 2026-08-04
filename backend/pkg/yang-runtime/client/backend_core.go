package client

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client/netconfcore"
)

// coreBackend 把自研 netconfcore 封进 ncDriver（Wave 3 双路径的新芯）。
//
// 语义对齐要点（以 scrapligo 行为为契约，双路径测试套锁定）：
//   - 应答内 <rpc-error> 由 Session.Do 以 *RPCReplyError 返回，此处折算进
//     ncResult.Failed（业务错误，error 返回 nil，会话继续可用）；
//   - 无截止时间的 ctx 补默认操作超时（scrapligo 有 60s op-timeout，语义对齐，
//     否则静默设备会挂死调用方）；
//   - GetConfig 收到的是完整 <filter> 包装元素（XPath select 风格），为字节级
//     对齐 scrapligo 报文形态，直接拼 get-config 体走 Session.Do 原样透传。
type coreBackend struct {
	sess *netconfcore.Session
}

// coreOpTimeout 缺省单操作超时（沿用原 scrapligo op-timeout 量级）。
const coreOpTimeout = 60 * time.Second

// closeTimeout 有界优雅关闭上限：超过即判定连接已死，强切传输层。
const closeTimeout = 5 * time.Second

func dialCore(info DeviceConnectionInfo) (ncDriver, error) {
	conn, err := netconfcore.DialSSH(info.IP, info.Port, info.Username, info.Password, info.Timeout)
	if err != nil {
		// 错误文案对齐 scrapligo 路径（"NETCONF" 关键字是 AUTO 协议分派
		// 测试锁定的行为契约），双路径调用方看到同形错误。
		return nil, fmt.Errorf("failed to open NETCONF connection: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), info.Timeout)
	defer cancel()
	sess, err := netconfcore.NewSession(ctx, conn)
	if err != nil {
		return nil, fmt.Errorf("failed to open NETCONF connection: %w", err)
	}
	return &coreBackend{sess: sess}, nil
}

func (b *coreBackend) opCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, coreOpTimeout)
}

// toResult 把 Session 返回值折算为 ncResult 契约：业务错误进 Failed，
// 传输/会话错误走 error（触发上层 isTransportError→markDisconnected）。
func toResult(raw []byte, err error) (ncResult, error) {
	var replyErr *netconfcore.RPCReplyError
	if errors.As(err, &replyErr) {
		return ncResult{Result: string(raw), Failed: replyErr}, nil
	}
	if err != nil {
		return ncResult{}, err
	}
	return ncResult{Result: string(raw)}, nil
}

func (b *coreBackend) do(ctx context.Context, body string) (ncResult, error) {
	octx, cancel := b.opCtx(ctx)
	defer cancel()
	raw, err := b.sess.Do(octx, []byte(body))
	return toResult(raw, err)
}

func (b *coreBackend) GetConfig(ctx context.Context, datastore, filterElem string) (ncResult, error) {
	return b.do(ctx, "<get-config><source><"+datastore+"/></source>"+filterElem+"</get-config>")
}

func (b *coreBackend) GetState(ctx context.Context, subtree string) (ncResult, error) {
	octx, cancel := b.opCtx(ctx)
	defer cancel()
	raw, err := b.sess.GetState(octx, subtree)
	return toResult(raw, err)
}

func (b *coreBackend) EditConfig(ctx context.Context, datastore, configXML string) (ncResult, error) {
	octx, cancel := b.opCtx(ctx)
	defer cancel()
	raw, err := b.sess.EditConfig(octx, datastore, configXML)
	return toResult(raw, err)
}

func (b *coreBackend) Commit(ctx context.Context) (ncResult, error) {
	octx, cancel := b.opCtx(ctx)
	defer cancel()
	raw, err := b.sess.Commit(octx)
	return toResult(raw, err)
}

func (b *coreBackend) CommitConfirmed(ctx context.Context, timeoutSec uint) (ncResult, error) {
	octx, cancel := b.opCtx(ctx)
	defer cancel()
	raw, err := b.sess.CommitConfirmed(octx, int(timeoutSec))
	return toResult(raw, err)
}

func (b *coreBackend) Discard(ctx context.Context) (ncResult, error) {
	octx, cancel := b.opCtx(ctx)
	defer cancel()
	raw, err := b.sess.DiscardChanges(octx)
	return toResult(raw, err)
}

func (b *coreBackend) RPC(ctx context.Context, payload string) (ncResult, error) {
	return b.do(ctx, payload)
}

func (b *coreBackend) Capabilities() []string {
	return b.sess.Capabilities()
}

// Close 有界优雅关闭：与 scrapligo backend 同款外形——在途操作可能长期持有
// 会话锁，watchdog 超时即 Abort 强切（Session 内部保证无泄漏协程）。
func (b *coreBackend) Close() error {
	done := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), closeTimeout)
		defer cancel()
		done <- b.sess.Close(ctx)
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(closeTimeout + time.Second):
		b.sess.Abort()
		return fmt.Errorf("netconf close timed out after %s (connection presumed dead, transport force-closed)", closeTimeout)
	}
}

// Kill 非阻塞强切（markDisconnected 路径）：在途/后续操作以传输错误失败并
// 自行判死会话，绝不等待在途操作。
func (b *coreBackend) Kill() {
	b.sess.Abort()
}
