package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/leezesi/usmp/backend/internal/yangschema"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"
)

// rpcExecClient captures ExecuteRPC for assertions (embeds fakeClient for the
// rest of the Client interface).
type rpcExecClient struct {
	fakeClient
	called          bool
	lastNS, lastRPC string
	lastInputs      []client.RPCInput
	result          *client.RPCResult
	err             error
}

func (c *rpcExecClient) ExecuteRPC(_ context.Context, ns, rpc string, inputs []client.RPCInput) (*client.RPCResult, error) {
	c.called = true
	c.lastNS, c.lastRPC, c.lastInputs = ns, rpc, inputs
	return c.result, c.err
}

// rpcTestManager is a Manager double serving a real schema + fake pool/store.
type rpcTestManager struct {
	manager.Manager
	schema schema.Schema
	pool   client.ClientPool
	store  device.Store
}

func (m rpcTestManager) GetSchema() schema.Schema         { return m.schema }
func (m rpcTestManager) GetClientPool() client.ClientPool { return m.pool }
func (m rpcTestManager) GetDeviceStore() device.Store     { return m.store }

func newRPCHandlerTest(t *testing.T, cli client.Client) (*RPCHandler, string) {
	t.Helper()
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("load schema: %v", err)
	}
	ip := "1.2.3.4"
	ds := device.NewStore()
	ds.Put(ip, client.DeviceConnectionInfo{IP: ip, Port: 830, Username: "admin", Password: "admin"})
	mgr := rpcTestManager{schema: s, pool: &fakePool{client: cli}, store: ds}
	return NewRPCHandler(mgr), ip
}

// doExecute drives the handler and returns the app-level response code (the API
// always replies HTTP 200 with the code in the envelope; see response.go).
func doExecute(t *testing.T, h *RPCHandler, ip, module, rpc string, inputs map[string]string) int {
	t.Helper()
	b, _ := json.Marshal(map[string]any{"inputs": inputs})
	c, w := newTestContext(http.MethodPost, "/api/v1/rpc/"+ip+"/"+module+"/"+rpc, bytes.NewReader(b), "ip", ip, "module", module, "rpc", rpc)
	c.Request.Header.Set("Content-Type", "application/json")
	h.Execute(c)

	var env struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v (body=%s)", err, w.Body)
	}
	return env.Code
}

// RPC-03：成功执行——校验 mandatory、经 ExecuteRPC 下发（正确 namespace/input）、返回结果。
func TestRPCExecute_Success(t *testing.T) {
	cli := &rpcExecClient{result: &client.RPCResult{OK: true}}
	h, ip := newRPCHandlerTest(t, cli)

	code := doExecute(t, h, ip, "ifm", "reset-if-counters-by-name", map[string]string{"if-name": "GE0/0/1"})
	if code != 0 {
		t.Fatalf("app code = %d, want 0 (success)", code)
	}
	if cli.lastRPC != "reset-if-counters-by-name" || len(cli.lastInputs) != 1 || cli.lastInputs[0].Value != "GE0/0/1" {
		t.Errorf("ExecuteRPC 入参不符: rpc=%s inputs=%+v", cli.lastRPC, cli.lastInputs)
	}
	if cli.lastNS == "" {
		t.Error("namespace 应非空（取自模块 schema）")
	}
}

// RPC-03：缺 mandatory input 拒绝，不下发。
func TestRPCExecute_MissingMandatory(t *testing.T) {
	cli := &rpcExecClient{result: &client.RPCResult{OK: true}}
	h, ip := newRPCHandlerTest(t, cli)

	code := doExecute(t, h, ip, "ifm", "reset-if-counters-by-name", map[string]string{})
	if code != http.StatusBadRequest {
		t.Fatalf("缺 mandatory 应 400, got app code=%d", code)
	}
	if cli.called {
		t.Error("校验失败不应下发到设备")
	}
}

// RPC-03：未知 rpc → 404。
func TestRPCExecute_UnknownRPC(t *testing.T) {
	cli := &rpcExecClient{result: &client.RPCResult{OK: true}}
	h, ip := newRPCHandlerTest(t, cli)

	code := doExecute(t, h, ip, "ifm", "no-such-rpc", map[string]string{})
	if code != http.StatusNotFound {
		t.Errorf("未知 rpc 应 404, got app code=%d", code)
	}
	if cli.called {
		t.Error("未知 rpc 不应下发")
	}
}

// 确认 ModuleRPCs 有 ifm 的 reset-if-counters-by-name（测试前提）。
func TestRPCExecute_Precondition(t *testing.T) {
	found := false
	for _, r := range yangschema.ModuleRPCs["ifm"] {
		if r.Name == "reset-if-counters-by-name" {
			found = true
		}
	}
	if !found {
		t.Skip("ifm 无 reset-if-counters-by-name（rpc.gen.go 未含），跳过")
	}
}
