package api

import (
	"net/http"
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/stretchr/testify/assert"
)

// 成功下发应产生一条审计记录，字段来自诚实来源（ip/path/提交摘要/触发结果）。
func TestSetConfig_RecordsAudit(t *testing.T) {
	mgr := manager.New()
	h := NewConfigHandler(mgr)

	w := postConfigReq(h, "10.0.0.1", "/vlan:vlan/vlan:vlans", `{"vlan":[{"id":10,"name":"VLAN10"},{"id":20,"name":"VLAN20"}]}`)
	assert.Equal(t, http.StatusOK, w.Code)

	logs := mgr.GetAuditStore().List()
	assert.Len(t, logs, 1)
	assert.Equal(t, "10.0.0.1", logs[0].DeviceIP)
	assert.Equal(t, "/vlan:vlan/vlan:vlans", logs[0].Path)
	assert.Equal(t, "vlan (2)", logs[0].Summary) // 提交 2 条 VLAN
	assert.Equal(t, "system", logs[0].Actor)     // 无鉴权来源
}

func TestSummarizeSubmitted(t *testing.T) {
	assert.Equal(t, "(空)", summarizeSubmitted(map[string]interface{}{}))
	assert.Equal(t, "(空)", summarizeSubmitted(nil))
	// 数组 → 键 (N)；非数组 → 键；多键按字母序稳定
	assert.Equal(t, "vlan (2)", summarizeSubmitted(map[string]interface{}{
		"vlan": []interface{}{map[string]interface{}{"id": 1}, map[string]interface{}{"id": 2}},
	}))
	assert.Equal(t, "enable", summarizeSubmitted(map[string]interface{}{"enable": true}))
	assert.Equal(t, "iface (1), name", summarizeSubmitted(map[string]interface{}{
		"name":  "x",
		"iface": []interface{}{"GE0/0/1"},
	}))
}

// 被拒下发（校验失败 400）不产生审计记录——只有真正接受的操作才入日志。
func TestSetConfig_RejectedPush_NoAudit(t *testing.T) {
	mgr := manager.New()
	h := NewConfigHandler(mgr)

	w := postConfigReq(h, "10.0.0.1", "/vlan:vlan/vlan:vlans", `{"vlan":[{"id":9999,"name":"BAD"}]}`)
	assert.Equal(t, http.StatusOK, w.Code) // 信封 200，body code=400
	assert.Empty(t, mgr.GetAuditStore().List(), "被拒下发不应写审计")
}
