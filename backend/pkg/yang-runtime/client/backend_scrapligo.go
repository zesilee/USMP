package client

import (
	"context"
	"fmt"
	"time"

	"github.com/scrapli/scrapligo/driver/netconf"
	"github.com/scrapli/scrapligo/driver/opoptions"
	"github.com/scrapli/scrapligo/driver/options"
	"github.com/scrapli/scrapligo/response"
	"github.com/scrapli/scrapligo/transport"
	"github.com/scrapli/scrapligo/util"
)

// scrapligoBackend 把 scrapligo v1.4.0 封进 ncDriver（现网在岗路径）。
// 其已知缺陷（死连接 Close 死锁、异常关闭协程泄漏）的既有补丁原样随迁：
// 有界 Close、Kill 走 Channel.Close + recover（详见各方法注释）。
type scrapligoBackend struct {
	driver *netconf.Driver
}

// dialScrapligo 建连（原 NETCONFClient.connect 逻辑原样迁入）。
func dialScrapligo(info DeviceConnectionInfo) (ncDriver, error) {
	opts := []util.Option{
		options.WithAuthUsername(info.Username),
		options.WithAuthPassword(info.Password),
		options.WithPort(info.Port),
		options.WithTimeoutSocket(info.Timeout),
		options.WithAuthNoStrictKey(),
		options.WithTransportType(transport.StandardTransport),
	}
	driver, err := netconf.NewDriver(info.IP, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create NETCONF driver: %w", err)
	}
	if err := driver.Open(); err != nil {
		return nil, fmt.Errorf("failed to open NETCONF connection: %w", err)
	}
	return &scrapligoBackend{driver: driver}, nil
}

func toNCResult(resp *response.NetconfResponse) ncResult {
	if resp == nil {
		return ncResult{}
	}
	return ncResult{Result: resp.Result, Failed: resp.Failed}
}

func (b *scrapligoBackend) GetConfig(_ context.Context, datastore, filterElem string) (ncResult, error) {
	withFilter := func(o interface{}) error {
		op, ok := o.(*netconf.OperationOptions)
		if !ok {
			return util.ErrIgnoredOption
		}
		op.Filter = filterElem
		return nil
	}
	var resp *response.NetconfResponse
	var err error
	if filterElem != "" {
		resp, err = b.driver.GetConfig(datastore, withFilter)
	} else {
		resp, err = b.driver.GetConfig(datastore)
	}
	return toNCResult(resp), err
}

func (b *scrapligoBackend) GetState(_ context.Context, subtree string) (ncResult, error) {
	resp, err := b.driver.Get(subtree)
	return toNCResult(resp), err
}

func (b *scrapligoBackend) EditConfig(_ context.Context, datastore, configXML string) (ncResult, error) {
	resp, err := b.driver.EditConfig(datastore, configXML)
	return toNCResult(resp), err
}

func (b *scrapligoBackend) Commit(_ context.Context) (ncResult, error) {
	resp, err := b.driver.Commit()
	return toNCResult(resp), err
}

func (b *scrapligoBackend) CommitConfirmed(_ context.Context, timeoutSec uint) (ncResult, error) {
	resp, err := b.driver.Commit(opoptions.WithCommitConfirmed(), opoptions.WithCommitConfirmTimeout(timeoutSec))
	return toNCResult(resp), err
}

func (b *scrapligoBackend) Discard(_ context.Context) (ncResult, error) {
	resp, err := b.driver.Discard()
	return toNCResult(resp), err
}

func (b *scrapligoBackend) RPC(_ context.Context, payload string) (ncResult, error) {
	withPayload := func(o interface{}) error {
		op, ok := o.(*netconf.OperationOptions)
		if !ok {
			return util.ErrIgnoredOption
		}
		op.Filter = payload
		return nil
	}
	resp, err := b.driver.RPC(withPayload)
	return toNCResult(resp), err
}

func (b *scrapligoBackend) Capabilities() []string {
	return b.driver.ServerCapabilities()
}

// closeTimeout bounds the graceful <close-session> teardown. scrapligo v1.4.0
// 在半死连接（read loop 已退）上 Close 会永久阻塞于无缓冲 done 发送——健康
// 连接的优雅关闭远快于此界，超时即判定连接已死。
const closeTimeout = 5 * time.Second

// Close 有界关闭：优雅路径走 driver.Close()（发 <close-session>），超时/内部
// panic 则退化为直接关传输层释放 fd。半死连接上泄漏一个阻塞在 scrapligo done
// 发送上的 goroutine，量级与异常关闭次数同阶，换取调用链永不挂死（R08）。
func (b *scrapligoBackend) Close() error {
	driver := b.driver
	done := make(chan error, 1)
	go func() {
		defer func() { _ = recover() }() // 第三方 double-close panic 不许崩进程（R09）
		done <- driver.Close()
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(closeTimeout):
		go func() {
			defer func() { _ = recover() }()
			_ = driver.Channel.Close()
		}()
		return fmt.Errorf("netconf close timed out after %s (connection presumed dead, transport force-closed)", closeTimeout)
	}
}

// Kill 强制切断：不能调 driver.Close()——scrapligo v1.4.0 在死连接上 Close
// 必死锁（read loop 阻塞在无缓冲 errs 发送、Close 阻塞在无缓冲 done 发送）。
// 直接关 Channel/Transport 释放 fd；卡在 errs 上的 read goroutine 是 scrapligo
// 缺陷，泄漏量与断连次数同阶，可接受。异步 + recover：关闭仅是清理，不能
// 阻塞调用链，第三方 double-close 也不许崩进程（R08）。
func (b *scrapligoBackend) Kill() {
	driver := b.driver
	go func() {
		defer func() { _ = recover() }()
		_ = driver.Channel.Close()
	}()
}
