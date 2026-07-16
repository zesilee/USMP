package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/leezesi/usmp/backend/internal/intent"
)

// BR-11 / BIO-07（矩阵 B3）—— 归属查询 API 与手改警告：命中返回 ownershipWarning
// 与意图清单，未命中零噪音。

func withOwnership(t *testing.T, claims map[string][]intent.Claim) {
	t.Helper()
	for key := range claims {
		intent.DefaultOwnership.Replace(key, claims[key])
	}
	t.Cleanup(func() {
		for key := range claims {
			intent.DefaultOwnership.Remove(key)
		}
	})
}

func TestOwnershipQueryByPath(t *testing.T) {
	withOwnership(t, map[string][]intent.Claim{
		"default/biz-100": {{Device: "10.0.0.1", Module: "vlan", Path: intent.VlanPath + "/vlan[id=100]"}},
	})
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ownership/:device", NewOwnershipHandler().Query)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ownership/10.0.0.1?path="+intent.VlanPath, nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data OwnershipData `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Data.Intents) != 1 || resp.Data.Intents[0] != "default/biz-100" {
		t.Fatalf("intents = %v", resp.Data.Intents)
	}

	// 未认领设备：零命中。
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/ownership/10.9.9.9?path="+intent.VlanPath, nil))
	var resp2 struct {
		Data OwnershipData `json:"data"`
	}
	_ = json.Unmarshal(w2.Body.Bytes(), &resp2)
	if len(resp2.Data.Intents) != 0 {
		t.Fatalf("unclaimed device intents = %v", resp2.Data.Intents)
	}
}

func TestOwnershipQueryDeviceWide(t *testing.T) {
	withOwnership(t, map[string][]intent.Claim{
		"default/biz-100": {
			{Device: "10.0.0.1", Module: "vlan", Path: intent.VlanPath + "/vlan[id=100]"},
			{Device: "10.0.0.1", Module: "ifm", Path: intent.IfmPath + "/interface[name=GE0/0/1]"},
		},
	})
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ownership/:device", NewOwnershipHandler().Query)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ownership/10.0.0.1", nil))
	var resp struct {
		Data OwnershipData `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Data.Claims) != 2 {
		t.Fatalf("claims = %+v, want 2", resp.Data.Claims)
	}
}

// ownershipWarningFor：命中→警告体（意图名+提示），未命中→nil（omitempty 零噪音）。
func TestOwnershipWarningFor(t *testing.T) {
	withOwnership(t, map[string][]intent.Claim{
		"default/biz-100": {{Device: "10.0.0.1", Module: "vlan", Path: intent.VlanPath + "/vlan[id=100]"}},
	})
	w := ownershipWarningFor("10.0.0.1", intent.VlanPath)
	if w == nil || len(w.Intents) != 1 || w.Intents[0] != "default/biz-100" || !strings.Contains(w.Message, "业务网络配置") {
		t.Fatalf("warning = %+v", w)
	}
	if got := ownershipWarningFor("10.0.0.1", "/system:system"); got != nil {
		t.Fatalf("unclaimed path should yield nil warning, got %+v", got)
	}
	// 序列化零噪音：nil 警告在 JSON 中不出现字段。
	b, _ := json.Marshal(ConfigSetData{Status: "ACCEPTED"})
	if strings.Contains(string(b), "ownershipWarning") {
		t.Errorf("nil warning must be omitted from JSON: %s", b)
	}
}
