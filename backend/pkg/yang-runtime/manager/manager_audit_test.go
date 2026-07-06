package manager

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/leezesi/usmp/backend/pkg/yang-runtime/audit"
	"github.com/stretchr/testify/assert"
)

func TestManager_AuditStore_Present(t *testing.T) {
	m := New()
	assert.NotNil(t, m.GetAuditStore(), "Manager 应持有审计 store")
	m.GetAuditStore().Record(audit.Record{DeviceIP: "10.0.0.1", Path: "/vlan", Summary: "vlans (1)"})
	assert.Len(t, m.GetAuditStore().List(), 1)
}

func TestManager_AuditFile_Persists(t *testing.T) {
	fp := filepath.Join(t.TempDir(), "sub", "audit.json") // 子目录不存在，store 应自建
	m := New(WithAuditFile(fp))
	m.GetAuditStore().Record(audit.Record{DeviceIP: "10.0.0.1", Path: "/vlan", Summary: "vlans (1)"})

	// 文件已落盘
	_, err := os.Stat(fp)
	assert.NoError(t, err, "记录后应持久化到本地 JSON（§8）")

	// 新 Manager 从同文件加载（模拟重启）
	m2 := New(WithAuditFile(fp))
	assert.Len(t, m2.GetAuditStore().List(), 1)
}
