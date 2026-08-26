package client

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"
)

// 回归：Set 的失败明细必须走日志，不能裸 fmt.Printf 打 stdout。
//
// 病灶（code-todo-backlog B4）：原实现在 `!result.Success` 分支里写
// `fmt.Printf("Change failed: %v\n", ch.Error)`。库代码直接占用 stdout 有三个问题：
// 绕过进程日志配置（无时间戳、无法重定向/分级）、在以 stdout 为数据通道的场景
// 会污染输出、且与本仓其余子系统一律 log.Printf 的约定不一致。

// nopDriver 是只满足 ncDriver 接口的空壳：本用例的变更在编码阶段就失败，
// 根本走不到任何 RPC，驱动只用来让 ensureConnected 通过。
type nopDriver struct{}

func (nopDriver) GetConfig(context.Context, string, string) (ncResult, error) { return ncResult{}, nil }
func (nopDriver) GetState(context.Context, string) (ncResult, error)          { return ncResult{}, nil }
func (nopDriver) EditConfig(context.Context, string, string) (ncResult, error) {
	return ncResult{}, nil
}
func (nopDriver) Commit(context.Context) (ncResult, error) { return ncResult{}, nil }
func (nopDriver) CommitConfirmed(context.Context, uint) (ncResult, error) {
	return ncResult{}, nil
}
func (nopDriver) Discard(context.Context) (ncResult, error)     { return ncResult{}, nil }
func (nopDriver) RPC(context.Context, string) (ncResult, error) { return ncResult{}, nil }
func (nopDriver) Capabilities() []string                        { return nil }
func (nopDriver) Close() error                                  { return nil }
func (nopDriver) Kill()                                         {}

// unregisteredModel 不在驱动注册表里，删除编码会按 DP-07 明确报错，
// 从而在不碰真实设备的前提下把 Set 推进失败分支。
type unregisteredModel struct {
	Name string
}

// connectedClient 造一个「已连接」的客户端，绕开真实 SSH 拨号。
func connectedClient(t *testing.T) *NETCONFClient {
	t.Helper()
	c := &NETCONFClient{info: DeviceConnectionInfo{IP: "10.0.0.9"}}
	c.backend = nopDriver{}
	c.connected = true
	return c
}

func captureLog(t *testing.T) func() string {
	t.Helper()
	var buf bytes.Buffer
	prevOut, prevFlags := log.Writer(), log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
	})
	return buf.String
}

func TestSetLogsFailedChangeDetail(t *testing.T) {
	logged := captureLog(t)
	c := connectedClient(t)

	res, err := c.Set(context.Background(), []Change{{
		Type:     DeleteChange,
		Path:     "/nope:nope/nope:items",
		OldValue: unregisteredModel{Name: "x"},
	}})

	if err == nil {
		t.Fatal("未注册模型的删除编码应失败（DP-07），却返回 nil")
	}
	if res == nil || res.Success {
		t.Fatalf("失败变更应使 result.Success=false，实际 %+v", res)
	}

	out := logged()
	if !strings.Contains(out, "/nope:nope/nope:items") {
		t.Errorf("失败明细应落日志并带出变更路径，实际日志为 %q", out)
	}
	if !strings.Contains(out, "netconf:") {
		t.Errorf("日志需带 netconf 子系统前缀（对齐本仓 log.Printf 约定），实际 %q", out)
	}
}

// 全部变更成功时不应产生失败日志——否则正常下发也刷噪声。
func TestSetSilentWhenAllChangesSucceed(t *testing.T) {
	logged := captureLog(t)
	c := connectedClient(t)

	res, err := c.Set(context.Background(), nil, WithCommit(false))
	if err != nil {
		t.Fatalf("空变更集不应报错: %v", err)
	}
	if res == nil || !res.Success {
		t.Fatalf("空变更集应成功，实际 %+v", res)
	}

	if out := logged(); strings.Contains(out, "netconf:") {
		t.Errorf("无失败变更时不应打 netconf 失败日志，实际 %q", out)
	}
}
