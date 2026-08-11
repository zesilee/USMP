package intent

import (
	"encoding/json"
	"testing"

	"github.com/leezesi/usmp/backend/internal/generated/native/business"
)

// 意图校验行为用例表（语义冻结自 ygot Validate 快照，2026-08-11 双跑一致实证
// 后旧腿随 ygot 包删除；两条历史意外语义保持）：
//   - mandatory 不校验（必填防线在 CRD OpenAPI required）
//   - min-elements 仅对**存在的空 list** 触发（list 缺失不拒）
//
// 收紧任一条=契约变更，须先在此改表拍板。
func validateSnapshotCases() map[string]struct {
	spec string
	ok   bool
} {
	return map[string]struct {
		spec string
		ok   bool
	}{
		"valid full": {`{
			"vlan-id": 100, "name": "biz-a",
			"devices": [{"ip": "10.0.0.1", "access-ports": ["GE0/0/1"], "trunk-ports": ["GE0/0/2"]}]
		}`, true},
		"name absent optional":     {`{"vlan-id": 100, "devices": [{"ip": "10.0.0.1"}]}`, true},
		"vlan-id at range max":     {`{"vlan-id": 4094, "devices": [{"ip": "10.0.0.1"}]}`, true},
		"vlan-id above range":      {`{"vlan-id": 4095, "devices": [{"ip": "10.0.0.1"}]}`, false},
		"vlan-id zero":             {`{"vlan-id": 0, "devices": [{"ip": "10.0.0.1"}]}`, false},
		"vlan-id missing accepted": {`{"devices": [{"ip": "10.0.0.1"}]}`, true},
		"name bad chars":           {`{"vlan-id": 100, "name": "bad name!", "devices": [{"ip": "10.0.0.1"}]}`, false},
		"name too long":            {`{"vlan-id": 100, "name": "0123456789012345678901234567890123456789", "devices": [{"ip": "10.0.0.1"}]}`, false},
		"ip not numeric":           {`{"vlan-id": 100, "devices": [{"ip": "abc"}]}`, false},
		"ip 999 passes pattern":    {`{"vlan-id": 100, "devices": [{"ip": "999.1.1.1"}]}`, true},
		"devices empty":            {`{"vlan-id": 100, "devices": []}`, false},
		"devices missing accepted": {`{"vlan-id": 100}`, true},
	}
}

func decodeBusinessDevice(t *testing.T, spec string) *business.Device {
	t.Helper()
	var sv interface{}
	if err := json.Unmarshal([]byte(spec), &sv); err != nil {
		t.Fatalf("bad case json: %v", err)
	}
	payload, err := json.Marshal(map[string]interface{}{"business-vlan-service": sv})
	if err != nil {
		t.Fatal(err)
	}
	root := &business.Device{}
	if err := business.Unmarshal(payload, root); err != nil {
		return nil // Unmarshal 层拒绝也算拒绝（端到端结论口径）
	}
	return root
}

var errRejectedAtDecode = jsonError("rejected at unmarshal")

type jsonError string

func (e jsonError) Error() string { return string(e) }
