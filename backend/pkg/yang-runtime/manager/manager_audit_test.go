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

// OA-05: WithAuditFile 已退役——设置路径仅弃用警告、走内存、绝不写盘（SC-06）。
func TestManager_AuditFile_Deprecated_NoWrite(t *testing.T) {
	fp := filepath.Join(t.TempDir(), "sub", "audit.json")
	m := New(WithAuditFile(fp))
	m.GetAuditStore().Record(audit.Record{DeviceIP: "10.0.0.1", Path: "/vlan", Summary: "vlans (1)"})

	assert.Len(t, m.GetAuditStore().List(), 1, "内存记录照常")
	_, err := os.Stat(fp)
	assert.True(t, os.IsNotExist(err), "OA-05: 不应写任何审计文件")
}

// OA-02: WithAuditStore 注入自定义后端（集群模式 CRD 实现走此缝）。
func TestManager_WithAuditStore_Injects(t *testing.T) {
	custom := audit.NewMemStore(5)
	m := New(WithAuditStore(custom))
	m.GetAuditStore().Record(audit.Record{DeviceIP: "10.0.0.2"})
	assert.Len(t, custom.List(), 1, "注入的后端应被 Manager 使用")
}
