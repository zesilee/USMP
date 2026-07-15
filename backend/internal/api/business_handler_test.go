package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	uns "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/leezesi/usmp/backend/internal/intent"
)

// D7 数据面（矩阵 A7 写路径 / FE-17 前提）—— 业务意图 CR 的 USMP API 代理：
// 前端不碰 apiserver；写路径先走约束引擎校验（BIC-03 前置校验），非法 400 不落 CR。

func newBizRouter(t *testing.T, objs ...client.Object) (*gin.Engine, client.Client) {
	t.Helper()
	sch := runtime.NewScheme()
	sch.AddKnownTypeWithName(intent.GVK, &uns.Unstructured{})
	sch.AddKnownTypeWithName(schema.GroupVersionKind{Group: intent.GVK.Group, Version: intent.GVK.Version, Kind: intent.GVK.Kind + "List"}, &uns.UnstructuredList{})
	proto := &uns.Unstructured{}
	proto.SetGroupVersionKind(intent.GVK)
	cl := fake.NewClientBuilder().WithScheme(sch).WithObjects(objs...).WithStatusSubresource(proto).Build()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewBusinessHandler(func() client.Client { return cl }, "default")
	router.GET("/business/vlan-services", h.List)
	router.GET("/business/vlan-services/:name", h.Get)
	router.POST("/business/vlan-services", h.Apply)
	router.DELETE("/business/vlan-services/:name", h.Delete)
	return router, cl
}

func bizCR(name string, vlanID int64) *uns.Unstructured {
	u := &uns.Unstructured{}
	u.SetGroupVersionKind(intent.GVK)
	u.SetNamespace("default")
	u.SetName(name)
	_ = uns.SetNestedMap(u.Object, map[string]interface{}{
		"vlan-id": vlanID,
		"devices": []interface{}{map[string]interface{}{"ip": "10.0.0.1"}},
	}, "spec")
	return u
}

func TestBusinessListAndGet(t *testing.T) {
	router, _ := newBizRouter(t, bizCR("biz-100", 100), bizCR("biz-200", 200))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/business/vlan-services", nil))
	if w.Code != 200 {
		t.Fatalf("list status=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data BusinessListData `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Data.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(resp.Data.Items))
	}

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/business/vlan-services/biz-100", nil))
	var one struct {
		Data BusinessVlanServiceItem `json:"data"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &one); err != nil {
		t.Fatal(err)
	}
	if one.Data.Name != "biz-100" || one.Data.Spec["vlan-id"] == nil {
		t.Fatalf("get = %+v", one.Data)
	}

	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, httptest.NewRequest(http.MethodGet, "/business/vlan-services/absent", nil))
	if code := envelopeCode(t, w3); code != 404 {
		t.Fatalf("absent get envelope code = %d, want 404", code)
	}
}

func TestBusinessApplyCreatesAndUpdates(t *testing.T) {
	router, cl := newBizRouter(t)

	body, _ := json.Marshal(map[string]interface{}{
		"name": "biz-300",
		"spec": map[string]interface{}{
			"vlan-id": 300,
			"name":    "office",
			"devices": []interface{}{map[string]interface{}{"ip": "10.0.0.1", "access-ports": []string{"GE0/0/1"}}},
		},
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/business/vlan-services", bytes.NewReader(body)))
	if w.Code != 200 {
		t.Fatalf("create status=%d body=%s", w.Code, w.Body.String())
	}
	u := &uns.Unstructured{}
	u.SetGroupVersionKind(intent.GVK)
	if err := cl.Get(httptest.NewRequest("GET", "/", nil).Context(), types.NamespacedName{Namespace: "default", Name: "biz-300"}, u); err != nil {
		t.Fatalf("CR not created: %v", err)
	}

	// 更新：同名再次 Apply 改 vlan-id。
	body2, _ := json.Marshal(map[string]interface{}{
		"name": "biz-300",
		"spec": map[string]interface{}{
			"vlan-id": 301,
			"devices": []interface{}{map[string]interface{}{"ip": "10.0.0.1"}},
		},
	})
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/business/vlan-services", bytes.NewReader(body2)))
	if w2.Code != 200 {
		t.Fatalf("update status=%d body=%s", w2.Code, w2.Body.String())
	}
	_ = cl.Get(httptest.NewRequest("GET", "/", nil).Context(), types.NamespacedName{Namespace: "default", Name: "biz-300"}, u)
	id, _, _ := uns.NestedInt64(u.Object, "spec", "vlan-id")
	if id != 301 {
		t.Fatalf("spec not updated, vlan-id = %d", id)
	}
}

// 写路径前置校验（BIC-03）：非法 spec 400 且不落 CR。
func TestBusinessApplyRejectsInvalidSpec(t *testing.T) {
	router, cl := newBizRouter(t)
	body, _ := json.Marshal(map[string]interface{}{
		"name": "bad",
		"spec": map[string]interface{}{
			"vlan-id": 5000,
			"devices": []interface{}{map[string]interface{}{"ip": "10.0.0.1"}},
		},
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/business/vlan-services", bytes.NewReader(body)))
	if code := envelopeCode(t, w); code != 400 {
		t.Fatalf("invalid spec envelope code = %d, want 400 (body=%s)", code, w.Body.String())
	}
	u := &uns.Unstructured{}
	u.SetGroupVersionKind(intent.GVK)
	if err := cl.Get(httptest.NewRequest("GET", "/", nil).Context(), types.NamespacedName{Namespace: "default", Name: "bad"}, u); err == nil {
		t.Fatal("invalid CR must not be created")
	}
	// 缺 name 同样 400。
	noName, _ := json.Marshal(map[string]interface{}{"spec": map[string]interface{}{"vlan-id": 100}})
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/business/vlan-services", bytes.NewReader(noName)))
	if code := envelopeCode(t, w2); code != 400 {
		t.Fatalf("missing name envelope code = %d, want 400", code)
	}
}

func TestBusinessDelete(t *testing.T) {
	router, cl := newBizRouter(t, bizCR("biz-100", 100))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/business/vlan-services/biz-100", nil))
	if w.Code != 200 {
		t.Fatalf("delete status=%d body=%s", w.Code, w.Body.String())
	}
	u := &uns.Unstructured{}
	u.SetGroupVersionKind(intent.GVK)
	if err := cl.Get(httptest.NewRequest("GET", "/", nil).Context(), types.NamespacedName{Namespace: "default", Name: "biz-100"}, u); err == nil {
		t.Fatal("CR should be deleted (finalizer 由控制器在真实集群处理)")
	}
}

// 无集群降级（BIO-01 API 面）：client 为 nil 时 503 明确报错，不 panic。
func TestBusinessUnavailableWithoutCluster(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewBusinessHandler(func() client.Client { return nil }, "default")
	router.GET("/business/vlan-services", h.List)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/business/vlan-services", nil))
	if code := envelopeCode(t, w); code != 503 {
		t.Fatalf("no-cluster envelope code = %d, want 503", code)
	}
}
