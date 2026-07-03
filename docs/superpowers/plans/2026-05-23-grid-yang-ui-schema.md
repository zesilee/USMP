# Grid YANG UI Schema Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Interfaces vertical slice where the backend converts YANG/ygot interface objects into complete Grid UI schema and the frontend renders it without understanding YANG.

**Architecture:** The backend owns schema generation, schema versioning, validation, and values-to-ygot conversion. The frontend owns generic Grid rendering, drawer editing, page state, and calling the new UI schema/apply APIs. The existing YangRenderer and VLAN/System pages stay unchanged.

**Tech Stack:** Go 1.26, Gin, yang-controller-runtime, ygot generated Huawei structs, Vue 3, TypeScript, Element Plus, Vitest, Playwright, NETCONF simulator.

---

## File Structure

### Backend

- Create `backend/internal/uischema/types.go`
  - Defines stable API-facing UI schema DTOs: layout, sections, widgets, columns, validation, values, apply request, field errors.
- Create `backend/internal/uischema/interfaces.go`
  - Generates the Interfaces grid schema and validates/applies Interfaces values.
- Create `backend/internal/uischema/interfaces_test.go`
  - Unit tests for schema generation, versioning, validation, and values conversion.
- Create `backend/internal/api/ui_schema_handler.go`
  - Gin handler for `GET /ui-schema/devices/:ip/interfaces` and `POST /ui-schema/devices/:ip/interfaces/apply`.
- Create `backend/internal/api/ui_schema_handler_test.go`
  - Handler-level tests for success, validation errors, and schema version mismatch.
- Modify `backend/internal/api/server.go`
  - Registers the new UI schema routes.
- Reuse `backend/internal/api/config_handler.go`
  - Calls existing `convertToTypedStruct` or matching typed conversion for Interfaces apply instead of duplicating ygot mapping.

### Frontend

- Create `frontend/src/types/grid-schema.ts`
  - TypeScript DTOs mirroring backend UI schema and apply responses.
- Modify `frontend/src/api/index.ts`
  - Adds `getInterfaceGridSchema(ip)` and `applyInterfaceGridConfig(ip, payload)`.
- Create `frontend/src/components/grid/GridRenderer.vue`
  - Generic schema renderer with toolbar, sections, submit and refresh events.
- Create `frontend/src/components/grid/GridSection.vue`
  - Renders one section in a 12-column grid layout.
- Create `frontend/src/components/grid/GridWidget.vue`
  - Renders table widgets and dispatches simple controls inside the edit drawer.
- Create `frontend/src/views/InterfaceGridPage.vue`
  - Interfaces vertical slice page using the new API and GridRenderer.
- Modify `frontend/src/router/index.ts`
  - Routes `/config/interface` to `InterfaceGridPage.vue`; VLAN/System stay on existing ConfigPage.
- Create tests under `frontend/test/components/grid/` and `frontend/test/views/InterfaceGridPage.test.ts`.

---

## Task 1: Backend UI Schema Types

**Files:**
- Create: `backend/internal/uischema/types.go`
- Test: `backend/internal/uischema/interfaces_test.go`

- [ ] **Step 1: Write the failing type compile test**

Create `backend/internal/uischema/interfaces_test.go` with this initial test:

```go
package uischema

import "testing"

func TestInterfacesSchemaShape(t *testing.T) {
	schema := NewInterfacesGenerator().BuildSchema("192.168.1.1")

	if schema.SchemaVersion == "" {
		t.Fatalf("expected schema version")
	}
	if schema.Module != "huawei-ifm" {
		t.Fatalf("module = %q, want huawei-ifm", schema.Module)
	}
	if schema.TargetPath != "/ifm:ifm/ifm:interfaces" {
		t.Fatalf("target path = %q", schema.TargetPath)
	}
	if schema.Layout.Type != "grid" || schema.Layout.Columns != 12 {
		t.Fatalf("unexpected layout: %+v", schema.Layout)
	}
	if len(schema.Sections) != 1 {
		t.Fatalf("sections = %d, want 1", len(schema.Sections))
	}
	if len(schema.Widgets) != 1 {
		t.Fatalf("widgets = %d, want 1", len(schema.Widgets))
	}

	widget := schema.Widgets[0]
	if widget.ID != "interfaces-table" || widget.Type != WidgetTable {
		t.Fatalf("unexpected widget: %+v", widget)
	}
	if widget.RowKey != "name" {
		t.Fatalf("row key = %q, want name", widget.RowKey)
	}
	if len(widget.Columns) < 4 {
		t.Fatalf("columns = %d, want at least 4", len(widget.Columns))
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run:

```bash
cd backend
go test ./internal/uischema -run TestInterfacesSchemaShape -v
```

Expected: FAIL because package `internal/uischema` or `NewInterfacesGenerator` does not exist.

- [ ] **Step 3: Add UI schema DTOs**

Create `backend/internal/uischema/types.go`:

```go
package uischema

type WidgetType string

const (
	WidgetText     WidgetType = "text"
	WidgetNumber   WidgetType = "number"
	WidgetSelect   WidgetType = "select"
	WidgetSwitch   WidgetType = "switch"
	WidgetTextarea WidgetType = "textarea"
	WidgetTable    WidgetType = "table"
)

type GridSchema struct {
	SchemaVersion   string                 `json:"schemaVersion"`
	Module          string                 `json:"module"`
	TargetPath      string                 `json:"targetPath"`
	CapabilitySource string                `json:"capabilitySource"`
	Layout          GridLayout             `json:"layout"`
	Sections        []GridSection          `json:"sections"`
	Widgets         []GridWidget           `json:"widgets"`
	Values          map[string]interface{} `json:"values"`
}

type GridLayout struct {
	Type    string `json:"type"`
	Columns int    `json:"columns"`
	Gap     string `json:"gap"`
}

type GridSection struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Widgets     []string `json:"widgets"`
}

type GridWidget struct {
	ID       string                 `json:"id"`
	Type     WidgetType             `json:"type"`
	Label    string                 `json:"label"`
	Help     string                 `json:"help,omitempty"`
	RowKey   string                 `json:"rowKey,omitempty"`
	Grid     WidgetGrid             `json:"grid"`
	Columns  []GridColumn           `json:"columns,omitempty"`
	Binding  map[string]interface{} `json:"binding,omitempty"`
	Disabled bool                   `json:"disabled,omitempty"`
	DisabledReason string           `json:"disabledReason,omitempty"`
}

type WidgetGrid struct {
	Span   int `json:"span"`
	Offset int `json:"offset,omitempty"`
	Order  int `json:"order,omitempty"`
}

type GridColumn struct {
	ID          string          `json:"id"`
	Type        WidgetType      `json:"type"`
	Label       string          `json:"label"`
	Placeholder string          `json:"placeholder,omitempty"`
	Readonly    bool            `json:"readonly,omitempty"`
	Options     []GridOption    `json:"options,omitempty"`
	Validation  GridValidation  `json:"validation,omitempty"`
}

type GridOption struct {
	Label string      `json:"label"`
	Value interface{} `json:"value"`
}

type GridValidation struct {
	Required  bool `json:"required,omitempty"`
	Min       *int `json:"min,omitempty"`
	Max       *int `json:"max,omitempty"`
	MinLength *int `json:"minLength,omitempty"`
	MaxLength *int `json:"maxLength,omitempty"`
}

type ApplyRequest struct {
	SchemaVersion string                 `json:"schemaVersion"`
	Values        map[string]interface{} `json:"values"`
}

type ApplyResult struct {
	SchemaVersion string                 `json:"schemaVersion"`
	Values        map[string]interface{} `json:"values,omitempty"`
	LastSync      string                 `json:"lastSync,omitempty"`
}

type FieldError struct {
	Field    string   `json:"field"`
	Messages []string `json:"messages"`
}

type ValidationError struct {
	Code        string              `json:"code"`
	Message     string              `json:"message"`
	FieldErrors map[string][]string `json:"fieldErrors,omitempty"`
}

func (e *ValidationError) Error() string {
	return e.Message
}
```

- [ ] **Step 4: Add minimal generator skeleton**

Create `backend/internal/uischema/interfaces.go`:

```go
package uischema

const (
	InterfacesWidgetID = "interfaces-table"
	InterfacesTargetPath = "/ifm:ifm/ifm:interfaces"
)

type InterfacesGenerator struct{}

func NewInterfacesGenerator() *InterfacesGenerator {
	return &InterfacesGenerator{}
}

func (g *InterfacesGenerator) BuildSchema(deviceIP string) GridSchema {
	maxLen80 := 80
	minMTU := 1280
	maxMTU := 9216

	return GridSchema{
		SchemaVersion: "interfaces:v1",
		Module: "huawei-ifm",
		TargetPath: InterfacesTargetPath,
		CapabilitySource: "module-set",
		Layout: GridLayout{Type: "grid", Columns: 12, Gap: "md"},
		Sections: []GridSection{{
			ID: "interfaces",
			Title: "接口配置",
			Description: "管理设备接口基础配置",
			Widgets: []string{InterfacesWidgetID},
		}},
		Widgets: []GridWidget{{
			ID: InterfacesWidgetID,
			Type: WidgetTable,
			Label: "接口列表",
			RowKey: "name",
			Grid: WidgetGrid{Span: 12},
			Columns: []GridColumn{
				{ID: "name", Type: WidgetText, Label: "接口名称", Readonly: true, Validation: GridValidation{Required: true}},
				{ID: "description", Type: WidgetText, Label: "描述", Validation: GridValidation{MaxLength: &maxLen80}},
				{ID: "mtu", Type: WidgetNumber, Label: "MTU", Validation: GridValidation{Min: &minMTU, Max: &maxMTU}},
				{ID: "admin-status", Type: WidgetSelect, Label: "管理状态", Options: []GridOption{{Label: "启用", Value: float64(1)}, {Label: "禁用", Value: float64(0)}}},
			},
			Binding: map[string]interface{}{"targetPath": InterfacesTargetPath},
		}},
		Values: map[string]interface{}{InterfacesWidgetID: []interface{}{}},
	}
}
```

- [ ] **Step 5: Run the test to verify it passes**

Run:

```bash
cd backend
go test ./internal/uischema -run TestInterfacesSchemaShape -v
```

Expected: PASS.

- [ ] **Step 6: Commit Task 1**

```bash
git add backend/internal/uischema/types.go backend/internal/uischema/interfaces.go backend/internal/uischema/interfaces_test.go
git commit -m "$(cat <<'EOF'
feat: 新增 Interfaces Grid UI Schema 类型

What: 新增后端 Grid UI schema DTO 和 Interfaces schema generator 的最小实现。
Why: 前端 Grid 重构需要后端提供完整控件数据，避免前端继续理解 YANG schema。
How: 在 internal/uischema 定义 layout、section、widget、validation、apply 请求结构，并生成 Interfaces table widget schema。

Co-Authored-By: Claude Code <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Backend Interfaces Validation and Conversion

**Files:**
- Modify: `backend/internal/uischema/interfaces.go`
- Modify: `backend/internal/uischema/interfaces_test.go`

- [ ] **Step 1: Add failing validation tests**

Append to `backend/internal/uischema/interfaces_test.go`:

```go
func TestValidateApplyAcceptsValidInterfaces(t *testing.T) {
	gen := NewInterfacesGenerator()
	req := ApplyRequest{
		SchemaVersion: "interfaces:v1",
		Values: map[string]interface{}{
			InterfacesWidgetID: []interface{}{
				map[string]interface{}{
					"name": "GigabitEthernet0/0/1",
					"description": "uplink",
					"mtu": float64(1500),
					"admin-status": float64(1),
				},
			},
		},
	}

	if err := gen.ValidateApply(req); err != nil {
		t.Fatalf("ValidateApply returned error: %v", err)
	}
}

func TestValidateApplyRejectsSchemaVersionMismatch(t *testing.T) {
	gen := NewInterfacesGenerator()
	req := ApplyRequest{SchemaVersion: "old", Values: map[string]interface{}{}}

	err := gen.ValidateApply(req)
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("error type = %T, want *ValidationError", err)
	}
	if validationErr.Code != "SCHEMA_VERSION_MISMATCH" {
		t.Fatalf("code = %q", validationErr.Code)
	}
}

func TestValidateApplyRejectsInvalidMTU(t *testing.T) {
	gen := NewInterfacesGenerator()
	req := ApplyRequest{
		SchemaVersion: "interfaces:v1",
		Values: map[string]interface{}{
			InterfacesWidgetID: []interface{}{
				map[string]interface{}{"name": "GigabitEthernet0/0/1", "mtu": float64(42)},
			},
		},
	}

	err := gen.ValidateApply(req)
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("error type = %T, want *ValidationError", err)
	}
	key := "interfaces-table:row:GigabitEthernet0/0/1:mtu"
	if len(validationErr.FieldErrors[key]) == 0 {
		t.Fatalf("expected mtu field error, got %+v", validationErr.FieldErrors)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
cd backend
go test ./internal/uischema -run 'TestValidateApply' -v
```

Expected: FAIL because `ValidateApply` is undefined.

- [ ] **Step 3: Implement validation**

Append to `backend/internal/uischema/interfaces.go`:

```go
func (g *InterfacesGenerator) ValidateApply(req ApplyRequest) error {
	if req.SchemaVersion != "interfaces:v1" {
		return &ValidationError{Code: "SCHEMA_VERSION_MISMATCH", Message: "Schema 已更新，请刷新后重试"}
	}

	rows, ok := req.Values[InterfacesWidgetID].([]interface{})
	if !ok {
		return &ValidationError{
			Code: "VALIDATION_FAILED",
			Message: "配置校验失败",
			FieldErrors: map[string][]string{InterfacesWidgetID: {"接口列表格式不正确"}},
		}
	}

	fieldErrors := map[string][]string{}
	for _, row := range rows {
		m, ok := row.(map[string]interface{})
		if !ok {
			fieldErrors[InterfacesWidgetID] = append(fieldErrors[InterfacesWidgetID], "接口行格式不正确")
			continue
		}

		name, _ := m["name"].(string)
		if name == "" {
			name = "unknown"
			fieldErrors["interfaces-table:row:unknown:name"] = append(fieldErrors["interfaces-table:row:unknown:name"], "接口名称不能为空")
		}

		if mtuValue, exists := m["mtu"]; exists {
			mtu, ok := numberToInt(mtuValue)
			if !ok || mtu < 1280 || mtu > 9216 {
				key := "interfaces-table:row:" + name + ":mtu"
				fieldErrors[key] = append(fieldErrors[key], "MTU 必须在 1280 到 9216 之间")
			}
		}

		if desc, exists := m["description"].(string); exists && len(desc) > 80 {
			key := "interfaces-table:row:" + name + ":description"
			fieldErrors[key] = append(fieldErrors[key], "描述长度不能超过 80 个字符")
		}
	}

	if len(fieldErrors) > 0 {
		return &ValidationError{Code: "VALIDATION_FAILED", Message: "配置校验失败", FieldErrors: fieldErrors}
	}
	return nil
}

func numberToInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	case float32:
		return int(n), true
	default:
		return 0, false
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run:

```bash
cd backend
go test ./internal/uischema -v
```

Expected: PASS.

- [ ] **Step 5: Commit Task 2**

```bash
git add backend/internal/uischema/interfaces.go backend/internal/uischema/interfaces_test.go
git commit -m "$(cat <<'EOF'
feat: 增加 Interfaces Grid 提交校验

What: 为 Interfaces Grid apply 请求增加 schemaVersion、接口名称、MTU 和描述长度校验。
Why: 后端需要成为 UI schema 约束的唯一来源，前端只展示后端返回的字段错误。
How: 在 InterfacesGenerator 中实现 ValidateApply，并返回稳定的错误码和 widget/row/field 错误键。

Co-Authored-By: Claude Code <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Backend UI Schema API Routes

**Files:**
- Create: `backend/internal/api/ui_schema_handler.go`
- Create: `backend/internal/api/ui_schema_handler_test.go`
- Modify: `backend/internal/api/server.go`
- Reuse: `backend/internal/api/config_handler.go` typed conversion helpers

- [ ] **Step 1: Write failing handler tests**

Create `backend/internal/api/ui_schema_handler_test.go`:

```go
package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUISchemaHandlerGetInterfaces(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewUISchemaHandler(nil)
	r.GET("/api/v1/ui-schema/devices/:ip/interfaces", h.GetInterfaces)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ui-schema/devices/192.168.1.1/interfaces", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var body Response
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !body.Success {
		t.Fatalf("expected success response: %+v", body)
	}
}

func TestUISchemaHandlerApplyRejectsInvalidMTU(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewUISchemaHandler(nil)
	r.POST("/api/v1/ui-schema/devices/:ip/interfaces/apply", h.ApplyInterfaces)

	payload := []byte(`{"schemaVersion":"interfaces:v1","values":{"interfaces-table":[{"name":"GigabitEthernet0/0/1","mtu":42}]}}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ui-schema/devices/192.168.1.1/interfaces/apply", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body["success"] != false {
		t.Fatalf("expected validation failure: %+v", body)
	}
	if body["code"] != "VALIDATION_FAILED" {
		t.Fatalf("code = %v", body["code"])
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
cd backend
go test ./internal/api -run 'TestUISchemaHandler' -v
```

Expected: FAIL because `NewUISchemaHandler` does not exist.

- [ ] **Step 3: Implement handler**

Create `backend/internal/api/ui_schema_handler.go`:

```go
package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/backend/internal/uischema"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
)

type UISchemaHandler struct {
	manager manager.Manager
	interfaces *uischema.InterfacesGenerator
}

func NewUISchemaHandler(m manager.Manager) *UISchemaHandler {
	return &UISchemaHandler{manager: m, interfaces: uischema.NewInterfacesGenerator()}
}

func (h *UISchemaHandler) GetInterfaces(c *gin.Context) {
	ip := c.Param("ip")
	Success(c, h.interfaces.BuildSchema(ip), "UI schema retrieved")
}

func (h *UISchemaHandler) ApplyInterfaces(c *gin.Context) {
	ip := c.Param("ip")
	var req uischema.ApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "code": "INVALID_REQUEST", "message": "Invalid request: " + err.Error()})
		return
	}

	if err := h.interfaces.ValidateApply(req); err != nil {
		if validationErr, ok := err.(*uischema.ValidationError); ok {
			c.JSON(http.StatusOK, gin.H{"success": false, "code": validationErr.Code, "message": validationErr.Message, "fieldErrors": validationErr.FieldErrors})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "code": "APPLY_FAILED", "message": err.Error()})
		return
	}

	configData := map[string]interface{}{"interface": req.Values[uischema.InterfacesWidgetID]}
	desiredConfig, err := convertToTypedStruct(uischema.InterfacesTargetPath, configData)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "code": "CONVERT_FAILED", "message": "Failed to parse configuration: " + err.Error()})
		return
	}

	triggered := false
	if h.manager != nil {
		if err := h.manager.GetConfigStore().Set(ip, uischema.InterfacesTargetPath, desiredConfig); err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "code": "STORE_FAILED", "message": "Failed to store configuration: " + err.Error()})
			return
		}
		triggered = h.manager.TriggerReconcile(ip, uischema.InterfacesTargetPath)
	}

	Success(c, gin.H{
		"schemaVersion": req.SchemaVersion,
		"values": req.Values,
		"lastSync": time.Now().UTC().Format(time.RFC3339),
		"reconciliation": gin.H{"triggered": triggered},
	}, "Configuration applied")
}
```

- [ ] **Step 4: Register routes**

Modify `backend/internal/api/server.go` inside `setupRoutes()` after the YANG routes:

```go
			uiSchemaGroup := v1.Group("/ui-schema")
			{
				uiSchemaHandler := NewUISchemaHandler(s.manager)
				uiSchemaGroup.GET("/devices/:ip/interfaces", uiSchemaHandler.GetInterfaces)
				uiSchemaGroup.POST("/devices/:ip/interfaces/apply", uiSchemaHandler.ApplyInterfaces)
			}
```

- [ ] **Step 5: Run handler tests**

Run:

```bash
cd backend
go test ./internal/api -run 'TestUISchemaHandler' -v
```

Expected: PASS.

- [ ] **Step 6: Run backend build**

Run:

```bash
cd backend
go build ./...
```

Expected: PASS.

- [ ] **Step 7: Commit Task 3**

```bash
git add backend/internal/api/ui_schema_handler.go backend/internal/api/ui_schema_handler_test.go backend/internal/api/server.go
git commit -m "$(cat <<'EOF'
feat: 暴露 Interfaces Grid UI Schema API

What: 新增 Interfaces UI schema 查询和 apply 提交接口。
Why: Grid 前端需要通过后端接口获取控件数据，并由后端统一校验提交内容。
How: 添加 UISchemaHandler，注册 /api/v1/ui-schema/devices/:ip/interfaces 和 /apply 路由，并覆盖 handler 测试。

Co-Authored-By: Claude Code <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Frontend Grid Schema Types and API

**Files:**
- Create: `frontend/src/types/grid-schema.ts`
- Modify: `frontend/src/api/index.ts`
- Test: `frontend/test/api/grid-schema.test.ts`

- [ ] **Step 1: Write failing API tests**

Create `frontend/test/api/grid-schema.test.ts`:

```ts
import { describe, expect, it, vi, beforeEach } from 'vitest'
import api, { applyInterfaceGridConfig, getInterfaceGridSchema } from '../../src/api'

vi.mock('../../src/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../src/api')>()
  return {
    ...actual,
    default: {
      get: vi.fn(),
      post: vi.fn()
    }
  }
})

describe('grid schema api', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('loads interface grid schema', async () => {
    vi.mocked(api.get).mockResolvedValue({ data: { success: true } })

    await getInterfaceGridSchema('192.168.1.1')

    expect(api.get).toHaveBeenCalledWith('/ui-schema/devices/192.168.1.1/interfaces')
  })

  it('applies interface grid values', async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { success: true } })
    const payload = { schemaVersion: 'interfaces:v1', values: { 'interfaces-table': [] } }

    await applyInterfaceGridConfig('192.168.1.1', payload)

    expect(api.post).toHaveBeenCalledWith('/ui-schema/devices/192.168.1.1/interfaces/apply', payload)
  })
})
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
cd frontend
npm run test -- grid-schema.test.ts
```

Expected: FAIL because exports do not exist.

- [ ] **Step 3: Add frontend types**

Create `frontend/src/types/grid-schema.ts`:

```ts
export type GridWidgetType = 'text' | 'number' | 'select' | 'switch' | 'textarea' | 'table'

export interface GridSchema {
  schemaVersion: string
  module: string
  targetPath: string
  capabilitySource: string
  layout: GridLayout
  sections: GridSection[]
  widgets: GridWidget[]
  values: Record<string, unknown>
}

export interface GridLayout {
  type: 'grid'
  columns: number
  gap: string
}

export interface GridSection {
  id: string
  title: string
  description?: string
  widgets: string[]
}

export interface GridWidget {
  id: string
  type: GridWidgetType
  label: string
  help?: string
  rowKey?: string
  grid: WidgetGrid
  columns?: GridColumn[]
  binding?: Record<string, unknown>
  disabled?: boolean
  disabledReason?: string
}

export interface WidgetGrid {
  span: number
  offset?: number
  order?: number
}

export interface GridColumn {
  id: string
  type: GridWidgetType
  label: string
  placeholder?: string
  readonly?: boolean
  options?: GridOption[]
  validation?: GridValidation
}

export interface GridOption {
  label: string
  value: string | number | boolean
}

export interface GridValidation {
  required?: boolean
  min?: number
  max?: number
  minLength?: number
  maxLength?: number
}

export interface InterfaceGridApplyPayload {
  schemaVersion: string
  values: Record<string, unknown>
}

export interface InterfaceGridApplyResult {
  schemaVersion: string
  values?: Record<string, unknown>
  lastSync?: string
}
```

- [ ] **Step 4: Add API functions**

Modify `frontend/src/api/index.ts`:

```ts
import type { GridSchema, InterfaceGridApplyPayload, InterfaceGridApplyResult } from '../types/grid-schema'
```

Add below existing config API functions:

```ts
export const getInterfaceGridSchema = (ip: string) => {
  return api.get<ApiResponse<GridSchema>>(`/ui-schema/devices/${ip}/interfaces`)
}

export const applyInterfaceGridConfig = (ip: string, payload: InterfaceGridApplyPayload) => {
  return api.post<ApiResponse<InterfaceGridApplyResult>>(`/ui-schema/devices/${ip}/interfaces/apply`, payload)
}
```

- [ ] **Step 5: Run API tests**

Run:

```bash
cd frontend
npm run test -- grid-schema.test.ts
```

Expected: PASS.

- [ ] **Step 6: Commit Task 4**

```bash
git add frontend/src/types/grid-schema.ts frontend/src/api/index.ts frontend/test/api/grid-schema.test.ts
git commit -m "$(cat <<'EOF'
feat: 新增 Interfaces Grid 前端 API 类型

What: 新增 Grid UI schema TypeScript 类型和 Interfaces schema/apply API 调用。
Why: 前端 GridRenderer 需要稳定的后端契约，且不再依赖 YANG schema 类型。
How: 定义 grid-schema DTO，并在 api/index.ts 中增加 getInterfaceGridSchema 与 applyInterfaceGridConfig。

Co-Authored-By: Claude Code <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Frontend GridRenderer Component

**Files:**
- Create: `frontend/src/components/grid/GridRenderer.vue`
- Create: `frontend/src/components/grid/GridSection.vue`
- Create: `frontend/src/components/grid/GridWidget.vue`
- Test: `frontend/test/components/grid/GridRenderer.test.ts`

- [ ] **Step 1: Write failing component test**

Create `frontend/test/components/grid/GridRenderer.test.ts`:

```ts
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import GridRenderer from '../../../src/components/grid/GridRenderer.vue'
import type { GridSchema } from '../../../src/types/grid-schema'

const schema: GridSchema = {
  schemaVersion: 'interfaces:v1',
  module: 'huawei-ifm',
  targetPath: '/ifm:ifm/ifm:interfaces',
  capabilitySource: 'module-set',
  layout: { type: 'grid', columns: 12, gap: 'md' },
  sections: [{ id: 'interfaces', title: '接口配置', description: '管理接口', widgets: ['interfaces-table'] }],
  widgets: [{
    id: 'interfaces-table',
    type: 'table',
    label: '接口列表',
    rowKey: 'name',
    grid: { span: 12 },
    columns: [
      { id: 'name', type: 'text', label: '接口名称', readonly: true, validation: { required: true } },
      { id: 'mtu', type: 'number', label: 'MTU', validation: { min: 1280, max: 9216 } }
    ]
  }],
  values: { 'interfaces-table': [{ name: 'GigabitEthernet0/0/1', mtu: 1500 }] }
}

describe('GridRenderer', () => {
  it('renders sections and table widget from schema', () => {
    const wrapper = mount(GridRenderer, {
      props: { schema, modelValue: schema.values, errors: {} }
    })

    expect(wrapper.text()).toContain('接口配置')
    expect(wrapper.text()).toContain('接口列表')
    expect(wrapper.text()).toContain('GigabitEthernet0/0/1')
    expect(wrapper.text()).toContain('MTU')
  })

  it('emits submit and refresh from toolbar', async () => {
    const wrapper = mount(GridRenderer, {
      props: { schema, modelValue: schema.values, errors: {} }
    })

    await wrapper.get('[data-test="grid-refresh"]').trigger('click')
    await wrapper.get('[data-test="grid-submit"]').trigger('click')

    expect(wrapper.emitted('refresh')).toBeTruthy()
    expect(wrapper.emitted('submit')).toBeTruthy()
  })
})
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
cd frontend
npm run test -- GridRenderer.test.ts
```

Expected: FAIL because component files do not exist.

- [ ] **Step 3: Implement GridRenderer.vue**

Create `frontend/src/components/grid/GridRenderer.vue`:

```vue
<template>
  <div class="grid-renderer">
    <div class="grid-toolbar">
      <div>
        <h2>{{ schema.sections[0]?.title || '配置管理' }}</h2>
        <p v-if="schema.capabilitySource">能力来源：{{ schema.capabilitySource }}</p>
      </div>
      <div class="grid-toolbar-actions">
        <el-button data-test="grid-refresh" :loading="loading" @click="$emit('refresh')">刷新</el-button>
        <el-button data-test="grid-submit" type="primary" :loading="submitting" @click="$emit('submit')">下发配置</el-button>
      </div>
    </div>

    <el-alert v-if="pageError" :title="pageError" type="error" :closable="false" class="grid-alert" />

    <GridSection
      v-for="section in schema.sections"
      :key="section.id"
      :section="section"
      :widgets="widgetsBySection(section.widgets)"
      :model-value="modelValue"
      :errors="errors"
      @update:model-value="$emit('update:modelValue', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import GridSection from './GridSection.vue'
import type { GridSchema, GridSection as GridSectionType, GridWidget } from '../../types/grid-schema'

const props = defineProps<{
  schema: GridSchema
  modelValue: Record<string, unknown>
  errors?: Record<string, string[]>
  loading?: boolean
  submitting?: boolean
  pageError?: string
}>()

defineEmits<{
  (e: 'update:modelValue', value: Record<string, unknown>): void
  (e: 'submit'): void
  (e: 'refresh'): void
}>()

function widgetsBySection(ids: string[]): GridWidget[] {
  return ids
    .map(id => props.schema.widgets.find(widget => widget.id === id))
    .filter((widget): widget is GridWidget => Boolean(widget))
}
</script>

<style scoped>
.grid-renderer { padding: 20px; }
.grid-toolbar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.grid-toolbar h2 { margin: 0; font-size: 20px; }
.grid-toolbar p { margin: 6px 0 0; color: #909399; font-size: 13px; }
.grid-toolbar-actions { display: flex; gap: 8px; }
.grid-alert { margin-bottom: 16px; }
</style>
```

- [ ] **Step 4: Implement GridSection.vue**

Create `frontend/src/components/grid/GridSection.vue`:

```vue
<template>
  <el-card class="grid-section">
    <template #header>
      <div>
        <strong>{{ section.title }}</strong>
        <p v-if="section.description">{{ section.description }}</p>
      </div>
    </template>
    <div class="grid-section-body">
      <GridWidget
        v-for="widget in widgets"
        :key="widget.id"
        :widget="widget"
        :model-value="modelValue"
        :errors="errors"
        @update:model-value="$emit('update:modelValue', $event)"
      />
    </div>
  </el-card>
</template>

<script setup lang="ts">
import GridWidget from './GridWidget.vue'
import type { GridSection, GridWidget as GridWidgetType } from '../../types/grid-schema'

defineProps<{
  section: GridSection
  widgets: GridWidgetType[]
  modelValue: Record<string, unknown>
  errors?: Record<string, string[]>
}>()

defineEmits<{
  (e: 'update:modelValue', value: Record<string, unknown>): void
}>()
</script>

<style scoped>
.grid-section { margin-bottom: 20px; }
.grid-section p { margin: 6px 0 0; color: #909399; font-size: 13px; }
.grid-section-body { display: grid; grid-template-columns: repeat(12, 1fr); gap: 16px; }
</style>
```

- [ ] **Step 5: Implement GridWidget.vue**

Create `frontend/src/components/grid/GridWidget.vue`:

```vue
<template>
  <div class="grid-widget" :style="gridStyle">
    <template v-if="widget.type === 'table'">
      <h3>{{ widget.label }}</h3>
      <el-table :data="rows" border>
        <el-table-column
          v-for="column in widget.columns || []"
          :key="column.id"
          :prop="column.id"
          :label="column.label"
        />
      </el-table>
    </template>
    <el-empty v-else :description="`暂不支持控件 ${widget.type}`" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { GridWidget } from '../../types/grid-schema'

const props = defineProps<{
  widget: GridWidget
  modelValue: Record<string, unknown>
  errors?: Record<string, string[]>
}>()

defineEmits<{
  (e: 'update:modelValue', value: Record<string, unknown>): void
}>()

const rows = computed(() => {
  const value = props.modelValue[props.widget.id]
  return Array.isArray(value) ? value : []
})

const gridStyle = computed(() => ({
  gridColumn: `span ${props.widget.grid?.span || 12}`
}))
</script>

<style scoped>
.grid-widget { min-width: 0; }
.grid-widget h3 { margin: 0 0 12px; font-size: 16px; }
</style>
```

- [ ] **Step 6: Run GridRenderer tests**

Run:

```bash
cd frontend
npm run test -- GridRenderer.test.ts
```

Expected: PASS.

- [ ] **Step 7: Commit Task 5**

```bash
git add frontend/src/components/grid/GridRenderer.vue frontend/src/components/grid/GridSection.vue frontend/src/components/grid/GridWidget.vue frontend/test/components/grid/GridRenderer.test.ts
git commit -m "$(cat <<'EOF'
feat: 新增通用 GridRenderer 组件

What: 新增 GridRenderer、GridSection 和 GridWidget，支持按后端 UI schema 渲染接口表格。
Why: Interfaces 重构要求前端只消费控件 schema，不再解析或推导 YANG 模型。
How: 使用 Element Plus 渲染 toolbar、section 和 table widget，并通过 Vitest 覆盖 schema 渲染与提交刷新事件。

Co-Authored-By: Claude Code <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: InterfaceGridPage Vertical Slice

**Files:**
- Create: `frontend/src/views/InterfaceGridPage.vue`
- Modify: `frontend/src/router/index.ts`
- Test: `frontend/test/views/InterfaceGridPage.test.ts`

- [ ] **Step 1: Write failing page test**

Create `frontend/test/views/InterfaceGridPage.test.ts`:

```ts
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import InterfaceGridPage from '../../src/views/InterfaceGridPage.vue'
import { applyInterfaceGridConfig, getInterfaceGridSchema } from '../../src/api'

vi.mock('../../src/api', () => ({
  getInterfaceGridSchema: vi.fn(),
  applyInterfaceGridConfig: vi.fn()
}))

const schemaResponse = {
  data: {
    success: true,
    data: {
      schemaVersion: 'interfaces:v1',
      module: 'huawei-ifm',
      targetPath: '/ifm:ifm/ifm:interfaces',
      capabilitySource: 'module-set',
      layout: { type: 'grid', columns: 12, gap: 'md' },
      sections: [{ id: 'interfaces', title: '接口配置', widgets: ['interfaces-table'] }],
      widgets: [{
        id: 'interfaces-table',
        type: 'table',
        label: '接口列表',
        rowKey: 'name',
        grid: { span: 12 },
        columns: [{ id: 'name', type: 'text', label: '接口名称' }]
      }],
      values: { 'interfaces-table': [{ name: 'GigabitEthernet0/0/1' }] }
    }
  }
}

describe('InterfaceGridPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(getInterfaceGridSchema).mockResolvedValue(schemaResponse as any)
    vi.mocked(applyInterfaceGridConfig).mockResolvedValue({ data: { success: true, data: { schemaVersion: 'interfaces:v1' } } } as any)
  })

  it('loads and renders backend grid schema', async () => {
    const wrapper = mount(InterfaceGridPage, { props: { deviceIp: '192.168.1.1' } })
    await flushPromises()

    expect(getInterfaceGridSchema).toHaveBeenCalledWith('192.168.1.1')
    expect(wrapper.text()).toContain('接口配置')
    expect(wrapper.text()).toContain('GigabitEthernet0/0/1')
  })

  it('submits schemaVersion and values to apply api', async () => {
    const wrapper = mount(InterfaceGridPage, { props: { deviceIp: '192.168.1.1' } })
    await flushPromises()

    await wrapper.get('[data-test="grid-submit"]').trigger('click')
    await flushPromises()

    expect(applyInterfaceGridConfig).toHaveBeenCalledWith('192.168.1.1', {
      schemaVersion: 'interfaces:v1',
      values: { 'interfaces-table': [{ name: 'GigabitEthernet0/0/1' }] }
    })
  })
})
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
cd frontend
npm run test -- InterfaceGridPage.test.ts
```

Expected: FAIL because `InterfaceGridPage.vue` does not exist.

- [ ] **Step 3: Implement InterfaceGridPage.vue**

Create `frontend/src/views/InterfaceGridPage.vue`:

```vue
<template>
  <GridRenderer
    v-if="schema"
    v-model="values"
    :schema="schema"
    :loading="loading"
    :submitting="submitting"
    :errors="fieldErrors"
    :page-error="pageError"
    @refresh="loadSchema"
    @submit="submit"
  />
  <div v-else class="interface-grid-loading">
    <el-icon class="is-loading"><Loading /></el-icon>
    <span>加载接口配置中...</span>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Loading } from '@element-plus/icons-vue'
import GridRenderer from '../components/grid/GridRenderer.vue'
import { applyInterfaceGridConfig, getInterfaceGridSchema } from '../api'
import type { GridSchema } from '../types/grid-schema'

const props = withDefaults(defineProps<{ deviceIp?: string }>(), {
  deviceIp: '192.168.1.1'
})

const schema = ref<GridSchema | null>(null)
const values = ref<Record<string, unknown>>({})
const fieldErrors = ref<Record<string, string[]>>({})
const pageError = ref('')
const loading = ref(false)
const submitting = ref(false)

async function loadSchema() {
  loading.value = true
  pageError.value = ''
  try {
    const res = await getInterfaceGridSchema(props.deviceIp)
    if (res.data.success && res.data.data) {
      schema.value = res.data.data
      values.value = res.data.data.values || {}
      fieldErrors.value = {}
    } else {
      pageError.value = res.data.message || '加载接口 UI schema 失败'
    }
  } catch (error: any) {
    pageError.value = error.message || '加载接口 UI schema 失败'
  } finally {
    loading.value = false
  }
}

async function submit() {
  if (!schema.value) return
  submitting.value = true
  fieldErrors.value = {}
  pageError.value = ''
  try {
    const res = await applyInterfaceGridConfig(props.deviceIp, {
      schemaVersion: schema.value.schemaVersion,
      values: values.value
    })
    if (res.data.success) {
      ElMessage.success('配置下发成功')
      if (res.data.data?.values) {
        values.value = res.data.data.values
      }
    } else {
      const data = res.data as any
      fieldErrors.value = data.fieldErrors || {}
      pageError.value = res.data.message || '配置下发失败'
    }
  } catch (error: any) {
    pageError.value = error.message || '配置下发失败'
  } finally {
    submitting.value = false
  }
}

onMounted(loadSchema)
</script>

<style scoped>
.interface-grid-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  min-height: 240px;
  color: #909399;
}
</style>
```

- [ ] **Step 4: Route Interfaces to the new page**

Modify `frontend/src/router/index.ts` route `/config/interface`:

```ts
  {
    path: '/config/interface',
    name: 'interface',
    component: () => import('../views/InterfaceGridPage.vue')
  },
```

- [ ] **Step 5: Run page tests**

Run:

```bash
cd frontend
npm run test -- InterfaceGridPage.test.ts
```

Expected: PASS.

- [ ] **Step 6: Commit Task 6**

```bash
git add frontend/src/views/InterfaceGridPage.vue frontend/src/router/index.ts frontend/test/views/InterfaceGridPage.test.ts
git commit -m "$(cat <<'EOF'
feat: 接入 Interfaces Grid 配置页面

What: 将接口配置路由切换到新的 InterfaceGridPage，加载后端 UI schema 并提交 apply 请求。
Why: Interfaces 垂直切片需要验证前端不理解 YANG 时仍能完成配置页面渲染和提交。
How: 新增 InterfaceGridPage，复用 GridRenderer，调用 getInterfaceGridSchema 与 applyInterfaceGridConfig，并保留 VLAN 等旧页面不变。

Co-Authored-By: Claude Code <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Verification and E2E Coverage

**Files:**
- Create: `frontend/tests/interfaces-grid.spec.ts`
- Optional Modify: `backend/internal/api/ui_schema_handler_test.go`

- [ ] **Step 1: Add Playwright E2E smoke test**

Create `frontend/tests/interfaces-grid.spec.ts`:

```ts
import { test, expect } from '@playwright/test'

test('interfaces grid page renders backend-driven grid', async ({ page }) => {
  await page.route('**/api/v1/ui-schema/devices/*/interfaces', async route => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        code: 0,
        message: 'UI schema retrieved',
        data: {
          schemaVersion: 'interfaces:v1',
          module: 'huawei-ifm',
          targetPath: '/ifm:ifm/ifm:interfaces',
          capabilitySource: 'module-set',
          layout: { type: 'grid', columns: 12, gap: 'md' },
          sections: [{ id: 'interfaces', title: '接口配置', widgets: ['interfaces-table'] }],
          widgets: [{
            id: 'interfaces-table',
            type: 'table',
            label: '接口列表',
            rowKey: 'name',
            grid: { span: 12 },
            columns: [{ id: 'name', type: 'text', label: '接口名称' }]
          }],
          values: { 'interfaces-table': [{ name: 'GigabitEthernet0/0/1' }] }
        }
      })
    })
  })

  await page.goto('/config/interface')

  await expect(page.getByText('接口配置')).toBeVisible()
  await expect(page.getByText('接口列表')).toBeVisible()
  await expect(page.getByText('GigabitEthernet0/0/1')).toBeVisible()
})
```

- [ ] **Step 2: Run E2E test**

Run:

```bash
cd frontend
npm run e2e -- interfaces-grid.spec.ts
```

Expected: PASS. If Playwright browsers are missing, run `npx playwright install` only after user approval because it downloads dependencies.

- [ ] **Step 3: Run full verification**

Run:

```bash
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "$REPO_ROOT/backend"
go build ./...
go test ./... -short -v
cd "$REPO_ROOT/frontend"
npm run build
npm run test
```

Expected: all commands PASS.

- [ ] **Step 4: Commit Task 7**

```bash
git add frontend/tests/interfaces-grid.spec.ts
git commit -m "$(cat <<'EOF'
test: 增加 Interfaces Grid 页面 E2E 冒烟测试

What: 新增 Playwright 用例验证接口配置页能渲染后端驱动的 Grid UI schema。
Why: Interfaces 垂直切片需要端到端保障，避免路由或 schema 渲染回归。
How: 在 E2E 中 mock UI schema API，访问 /config/interface 并断言接口列表内容可见。

Co-Authored-By: Claude Code <noreply@anthropic.com>
EOF
)"
```

---

## Self-Review

- Spec coverage: backend schema generation, apply API, ConfigStore/reconcile handoff, frontend GridRenderer, Interfaces page, error handling, and tests are covered by Tasks 1-7.
- Scope control: VLAN/System migration, advanced row inline editing, visual schema designer, and database storage are excluded.
- Placeholder scan: no unresolved placeholders are present.
- Type consistency: backend `schemaVersion`, `values`, `fieldErrors`, `interfaces-table`, and frontend DTO names are consistent across tasks.
- Apply path: Task 3 validates the Grid values, converts them through existing typed configuration helpers, writes desired state to ConfigStore, and triggers reconciliation for `/ifm:ifm/ifm:interfaces`.
