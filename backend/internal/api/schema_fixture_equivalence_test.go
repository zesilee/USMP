package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/leezesi/usmp/backend/internal/yangschema"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
)

// SF-03 忠实性防线（B3）。
//
// schemadump 工具经 api.BuildYangSchemaNested 导出 fixture，绕开了 HTTP。这就必须
// 证明「工具导出的」等于「用户经 HTTP 实际拿到的」——否则 fixture 会与线上契约悄悄
// 脱钩，下游前端黄金全绿而实际渲染是错的（比没测试更危险）。本测试是整个 schema
// 驱动测试体系的信任锚点，覆盖全部已加载模块，禁止抽样。
//
// 比对口径：两条代码路径产出的都是 YangSchema 值，各自经同一 json.Marshal 归一为
// canonical（紧凑）JSON 后逐字节比较——消除编码器配置差异，纯粹校验两条路径的
// 语义等值。fixture 文件本身用缩进 JSON 存盘（便于 diff 评审）是独立的呈现选择，
// 不影响此处的等值判定。

// canonicalSchemaJSON marshals a YangSchema to canonical (compact) JSON, the
// normal form both code paths are compared in.
func canonicalSchemaJSON(t *testing.T, ys YangSchema) []byte {
	t.Helper()
	b, err := json.Marshal(ys)
	if err != nil {
		t.Fatalf("marshal schema: %v", err)
	}
	return b
}

// httpSchemaData drives the real GetSchema handler with ?form=nested and returns
// the response envelope's data field re-normalized through canonicalSchemaJSON.
func httpSchemaData(t *testing.T, h *YangHandler, module string) []byte {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "module", Value: module}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/yang/schema/"+module+"?form=nested", nil)
	h.GetSchema(c)

	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("module %q: decode envelope: %v (body=%s)", module, err, w.Body.Bytes())
	}
	var ys YangSchema
	if err := json.Unmarshal(env.Data, &ys); err != nil {
		t.Fatalf("module %q: decode data into YangSchema: %v (data=%s)", module, err, env.Data)
	}
	return canonicalSchemaJSON(t, ys)
}

// SF-03: 对每个已加载模块，导出路径与 HTTP 路径的 schema 逐字节相等。
func TestSchemaFixtureEquivalence_ToolMatchesHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("load schema: %v", err)
	}
	h := NewYangHandler(manager.New(manager.WithSchema(s)))

	count := 0
	for _, mod := range s.Modules() {
		if mod.Root() == nil {
			continue
		}
		name := mod.Name()
		toolJSON := canonicalSchemaJSON(t, BuildYangSchemaNested(mod))
		httpJSON := httpSchemaData(t, h, name)
		if !bytes.Equal(toolJSON, httpJSON) {
			t.Errorf("module %q: exporter path != HTTP path\n tool=%s\n http=%s", name, toolJSON, httpJSON)
		}
		count++
	}
	if count == 0 {
		t.Fatal("no modules compared — equivalence guard would be vacuous")
	}
	t.Logf("verified exporter≡HTTP for %d modules", count)
}

// SF-03 负路径（task 3.3）：证明等值断言非恒真——两条路径若真的分叉，比较必须报不等。
// 构造一个「被污染的 HTTP 路径」（在 schema 上注入一个字段），断言 canonical 比较确实
// 判定不等。防止 TestSchemaFixtureEquivalence_ToolMatchesHTTP 因断言写死而假绿。
func TestSchemaFixtureEquivalence_DetectsDivergence(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, err := yangschema.Load()
	if err != nil {
		t.Fatalf("load schema: %v", err)
	}
	mod, ok := s.Module("vlan")
	if !ok {
		t.Fatal("vlan module not loaded — cannot run divergence guard")
	}

	toolJSON := canonicalSchemaJSON(t, BuildYangSchemaNested(mod))

	diverged := BuildYangSchemaNested(mod)
	diverged.Fields = append(diverged.Fields, FieldDef{
		Path: "/vlan/__injected_divergence__", Type: "string", Label: "injected",
	})
	divergedJSON := canonicalSchemaJSON(t, diverged)

	if bytes.Equal(toolJSON, divergedJSON) {
		t.Fatal("equivalence comparison is vacuous: it treats a diverged schema as equal")
	}
}
