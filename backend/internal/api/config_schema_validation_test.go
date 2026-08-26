package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/leezesi/usmp/backend/internal/yangschema"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
)

// BR-08 扩展（change config-write-validation）：写路径接入 Schema IR 约束校验。
// 违反 YANG 约束的提交 SHALL 返回 400 且零副作用——不写 ConfigStore、不触发对账、
// 不触达设备。以前这类配置会被原样收下、发到设备侧才失败。

// loadedManager 造一个装了真实 schema IR 的 Manager（生产形态；manager.New()
// 缺省是空 schema，校验无从谈起）。
func loadedManager(t *testing.T) *manager.DefaultManager {
	t.Helper()
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("load schema: %v", err)
	}
	return manager.New(manager.WithSchema(s))
}

type validationEnvelope struct {
	Code    int    `json:"code"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func decodeValidationEnvelope(t *testing.T, body string) validationEnvelope {
	t.Helper()
	var env validationEnvelope
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		t.Fatalf("decode envelope: %v (%s)", err, body)
	}
	return env
}

// 违反 length 约束（接口 description 超长）：400 + 零副作用。
func TestSetConfig_SchemaViolationRejectedWithNoSideEffects(t *testing.T) {
	mgr := loadedManager(t)
	h := NewConfigHandler(mgr)

	const ip = "10.0.0.77"
	const path = "/ifm:ifm/ifm:interfaces"
	longDesc := strings.Repeat("x", 300) // ifm description length 上界远小于 300

	w := postConfigRaw(h, ip, path, "", `{"interface":[{"name":"GigabitEthernet0/0/1","description":"`+longDesc+`"}]}`)
	assert.Equal(t, http.StatusOK, w.Code) // 信封恒 200

	env := decodeValidationEnvelope(t, w.Body.String())
	assert.Equal(t, 400, env.Code, "违反 YANG 约束应返回 400")
	assert.False(t, env.Success)
	assert.Contains(t, env.Message, "description", "错误应点名违约的叶")

	// 零副作用：desired 未被写入。
	stored, _ := mgr.GetConfigStore().Get(ip, path)
	assert.Nil(t, stored, "拒绝的提交不得写入 ConfigStore")
	assert.Empty(t, mgr.GetAuditStore().List(), "拒绝的提交不得写审计")
}

// 合法配置照常接受——防止接入校验把正常路径打死。
func TestSetConfig_ValidConfigStillAccepted(t *testing.T) {
	mgr := loadedManager(t)
	h := NewConfigHandler(mgr)

	const ip = "10.0.0.78"
	const path = "/ifm:ifm/ifm:interfaces"

	w := postConfigRaw(h, ip, path, "", `{"interface":[{"name":"GigabitEthernet0/0/1","description":"uplink"}]}`)
	env := decodeValidationEnvelope(t, w.Body.String())
	assert.NotEqual(t, 400, env.Code, "合法配置不应被校验拒绝: %s", env.Message)
}

// schema 未装载（单测/降级启动）时不拦——无约束可校验，不能因此拒掉全部写入。
func TestSetConfig_UnloadedSchemaDoesNotBlock(t *testing.T) {
	mgr := manager.New() // 空 schema
	h := NewConfigHandler(mgr)

	w := postConfigRaw(h, "10.0.0.79", "/ifm:ifm/ifm:interfaces", "",
		`{"interface":[{"name":"GigabitEthernet0/0/1","description":"uplink"}]}`)
	env := decodeValidationEnvelope(t, w.Body.String())
	assert.NotEqual(t, 400, env.Code, "schema 未装载不应导致写入被拒: %s", env.Message)
}

// 模型未编码的域约束（VLAN ID 范围）仍由显式校验拦下——两道校验覆盖面不重叠。
func TestSetConfig_VlanIDRangeStillEnforced(t *testing.T) {
	mgr := loadedManager(t)
	h := NewConfigHandler(mgr)

	w := postConfigRaw(h, "10.0.0.80", "/vlan:vlan/vlan:vlans", "", `{"vlan":[{"id":4095,"name":"bad"}]}`)
	env := decodeValidationEnvelope(t, w.Body.String())
	assert.Equal(t, 400, env.Code, "VLAN ID 4095 应被拒（YANG 模型未编码此范围，靠显式校验）")
}
