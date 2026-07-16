package api

import (
	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	uns "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/leezesi/usmp/backend/internal/intent"
)

// BusinessHandler proxies BusinessVlanService CR CRUD through the USMP API
// (design D7)：前端不直连 apiserver；kubectl/GitOps 直写是旁路而非前端路径。
// 写路径先过约束引擎（DecodeSpec，BIC-03 写前校验）——USMP API 是一期唯一受
// 支持写入面的强契约。clientFn 惰性取用（无集群时返回 nil → 503 降级，R08）。
type BusinessHandler struct {
	clientFn  func() client.Client
	namespace string
}

// NewBusinessHandler builds the proxy handler. clientFn returns the
// controller-runtime client wired at intent.Register (nil without a cluster).
func NewBusinessHandler(clientFn func() client.Client, namespace string) *BusinessHandler {
	return &BusinessHandler{clientFn: clientFn, namespace: namespace}
}

// BusinessVlanServiceItem is one intent instance (spec + status passthrough:
// 结构由意图 YANG/CRD 定义，API 不再重复建模——YANG 是唯一 schema 源).
type BusinessVlanServiceItem struct {
	Name   string                 `json:"name"`
	Spec   map[string]interface{} `json:"spec"`
	Status map[string]interface{} `json:"status,omitempty"`
}

// BusinessListData is the list response payload.
type BusinessListData struct {
	Items []BusinessVlanServiceItem `json:"items"`
}

// BusinessApplyRequest creates or updates an intent instance.
type BusinessApplyRequest struct {
	Name string                 `json:"name" binding:"required"`
	Spec map[string]interface{} `json:"spec" binding:"required"`
}

func (h *BusinessHandler) client(c *gin.Context) client.Client {
	cl := h.clientFn()
	if cl == nil {
		Error(c, 503, "业务网络配置不可用：未连接 Kubernetes 集群（意图持久化载体）")
		return nil
	}
	return cl
}

func itemFromCR(u *uns.Unstructured) BusinessVlanServiceItem {
	spec, _, _ := uns.NestedMap(u.Object, "spec")
	status, _, _ := uns.NestedMap(u.Object, "status")
	return BusinessVlanServiceItem{Name: u.GetName(), Spec: spec, Status: status}
}

// List returns all intent instances.
//
// @Summary  列出业务 VLAN 打通意图实例
// @Tags     business
// @Produce  json
// @Success  200 {object} Response{data=BusinessListData} "实例清单（含收敛状态）"
// @Failure  503 {object} Response "未连接集群，业务配置不可用"
// @Router   /business/vlan-services [get]
func (h *BusinessHandler) List(c *gin.Context) {
	cl := h.client(c)
	if cl == nil {
		return
	}
	list := &uns.UnstructuredList{}
	list.SetGroupVersionKind(intent.GVK.GroupVersion().WithKind(intent.GVK.Kind + "List"))
	if err := cl.List(c.Request.Context(), list, client.InNamespace(h.namespace)); err != nil {
		Error(c, 502, "读取意图实例失败: "+err.Error())
		return
	}
	data := BusinessListData{Items: make([]BusinessVlanServiceItem, 0, len(list.Items))}
	for i := range list.Items {
		data.Items = append(data.Items, itemFromCR(&list.Items[i]))
	}
	Success(c, data, "business vlan services listed")
}

// Get returns one intent instance.
//
// @Summary  读取单个业务 VLAN 打通意图实例
// @Tags     business
// @Produce  json
// @Param    name path string true "实例名"
// @Success  200 {object} Response{data=BusinessVlanServiceItem} "实例详情"
// @Failure  404 {object} Response "实例不存在"
// @Failure  503 {object} Response "未连接集群"
// @Router   /business/vlan-services/{name} [get]
func (h *BusinessHandler) Get(c *gin.Context) {
	cl := h.client(c)
	if cl == nil {
		return
	}
	u := &uns.Unstructured{}
	u.SetGroupVersionKind(intent.GVK)
	err := cl.Get(c.Request.Context(), types.NamespacedName{Namespace: h.namespace, Name: c.Param("name")}, u)
	if apierrors.IsNotFound(err) {
		Error(c, 404, "意图实例不存在: "+c.Param("name"))
		return
	}
	if err != nil {
		Error(c, 502, "读取意图实例失败: "+err.Error())
		return
	}
	Success(c, itemFromCR(u), "business vlan service resolved")
}

// Apply creates or updates an intent instance (spec 先过约束引擎校验).
//
// @Summary  创建/更新业务 VLAN 打通意图实例
// @Tags     business
// @Accept   json
// @Produce  json
// @Param    body body BusinessApplyRequest true "实例名 + 意图 spec（结构由 usmp-business-vlan YANG 定义）"
// @Success  200 {object} Response{data=BusinessVlanServiceItem} "已提交（控制器异步收敛，状态见 status）"
// @Failure  400 {object} Response "spec 违反意图 YANG 约束"
// @Failure  503 {object} Response "未连接集群"
// @Router   /business/vlan-services [post]
func (h *BusinessHandler) Apply(c *gin.Context) {
	cl := h.client(c)
	if cl == nil {
		return
	}
	var req BusinessApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "请求体不合法: "+err.Error())
		return
	}

	u := &uns.Unstructured{}
	u.SetGroupVersionKind(intent.GVK)
	u.SetNamespace(h.namespace)
	u.SetName(req.Name)
	_ = uns.SetNestedMap(u.Object, req.Spec, "spec")

	// 写前校验（BIC-03）：USMP API 是强契约写入面，非法 spec 不落 CR。
	if _, err := intent.DecodeSpec(u); err != nil {
		Error(c, 400, "意图校验失败: "+err.Error())
		return
	}

	existing := &uns.Unstructured{}
	existing.SetGroupVersionKind(intent.GVK)
	err := cl.Get(c.Request.Context(), types.NamespacedName{Namespace: h.namespace, Name: req.Name}, existing)
	switch {
	case apierrors.IsNotFound(err):
		if err := cl.Create(c.Request.Context(), u); err != nil {
			Error(c, 502, "创建意图实例失败: "+err.Error())
			return
		}
	case err != nil:
		Error(c, 502, "读取意图实例失败: "+err.Error())
		return
	default:
		_ = uns.SetNestedMap(existing.Object, req.Spec, "spec")
		if err := cl.Update(c.Request.Context(), existing); err != nil {
			Error(c, 502, "更新意图实例失败: "+err.Error())
			return
		}
		u = existing
	}
	Success(c, itemFromCR(u), "business vlan service applied")
}

// Delete removes an intent instance (finalizer 在集群侧拦截直至设备清理完成).
//
// @Summary  删除业务 VLAN 打通意图实例（设备配置由控制器清理后放行）
// @Tags     business
// @Produce  json
// @Param    name path string true "实例名"
// @Success  200 {object} Response "删除已受理"
// @Failure  404 {object} Response "实例不存在"
// @Failure  503 {object} Response "未连接集群"
// @Router   /business/vlan-services/{name} [delete]
func (h *BusinessHandler) Delete(c *gin.Context) {
	cl := h.client(c)
	if cl == nil {
		return
	}
	u := &uns.Unstructured{}
	u.SetGroupVersionKind(intent.GVK)
	u.SetNamespace(h.namespace)
	u.SetName(c.Param("name"))
	err := cl.Delete(c.Request.Context(), u)
	if apierrors.IsNotFound(err) {
		Error(c, 404, "意图实例不存在: "+c.Param("name"))
		return
	}
	if err != nil {
		Error(c, 502, "删除意图实例失败: "+err.Error())
		return
	}
	Success(c, gin.H{"name": c.Param("name")}, "deletion accepted (device cleanup via finalizer)")
}
