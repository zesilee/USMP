/*
Copyright 2024 USMP Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	bizv1 "github.com/leezesi/usmp/backend/api/v1"
	"github.com/leezesi/usmp/backend/pkg/translator"
	netconfclient "github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
)

const (
	// BusinessRouteFinalizer Finalizer 名称
	BusinessRouteFinalizer = "biz.usmp.io/businessroute-finalizer"
)

// BusinessRouteReconciler reconciles a BusinessRoute object
type BusinessRouteReconciler struct {
	k8sclient.Client
	Scheme     *runtime.Scheme
	ClientPool netconfclient.ClientPool
}

//+kubebuilder:rbac:groups=biz.usmp.io,resources=businessroutes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=biz.usmp.io,resources=businessroutes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=biz.usmp.io,resources=businessroutes/finalizers,verbs=update

// Reconcile 静态路由调和循环
func (r *BusinessRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 1. 获取 BusinessRoute CR
	businessRoute := &bizv1.BusinessRoute{}
	if err := r.Get(ctx, req.NamespacedName, businessRoute); err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get BusinessRoute")
		return ctrl.Result{}, err
	}

	// 2. 检查是否被删除
	if !businessRoute.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(businessRoute, BusinessRouteFinalizer) {
			logger.Info("清理路由配置",
				"device", businessRoute.Spec.DeviceID,
				"destination", businessRoute.Spec.DestinationCIDR)
			// 删除设备上的路由配置
			if err := r.deleteRouteFromDevice(ctx, businessRoute); err != nil {
				logger.Error(err, "删除设备路由配置失败")
				businessRoute.Status.Message = fmt.Sprintf("清理失败: %v", err)
				r.Status().Update(ctx, businessRoute)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, err
			}
			// 移除 Finalizer
			controllerutil.RemoveFinalizer(businessRoute, BusinessRouteFinalizer)
			if err := r.Update(ctx, businessRoute); err != nil {
				logger.Error(err, "Failed to remove finalizer")
				return ctrl.Result{}, err
			}
			logger.Info("Successfully deleted BusinessRoute")
		}
		return ctrl.Result{}, nil
	}

	// 3. 添加 Finalizer（如果不存在）
	if !controllerutil.ContainsFinalizer(businessRoute, BusinessRouteFinalizer) {
		logger.Info("Adding finalizer for BusinessRoute")
		controllerutil.AddFinalizer(businessRoute, BusinessRouteFinalizer)
		if err := r.Update(ctx, businessRoute); err != nil {
			logger.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
	}

	// 4. 初始化状态
	if businessRoute.Status.Phase == "" {
		businessRoute.Status.Phase = bizv1.PhasePending
		if err := r.Status().Update(ctx, businessRoute); err != nil {
			logger.Error(err, "Failed to update initial status")
			return ctrl.Result{}, err
		}
	}

	// 5. 验证路由配置
	if err := r.validateRouteConfig(businessRoute); err != nil {
		businessRoute.Status.Phase = bizv1.PhaseFailed
		businessRoute.Status.Message = fmt.Sprintf("配置验证失败: %v", err)
		businessRoute.Status.ErrorType = ErrorTypePermanent
		r.Status().Update(ctx, businessRoute)
		return ctrl.Result{}, err
	}

	// 6. 同步配置到设备
	logger.Info("Reconciling BusinessRoute",
		"device", businessRoute.Spec.DeviceID,
		"destination", businessRoute.Spec.DestinationCIDR)

	businessRoute.Status.Phase = bizv1.PhaseSyncing
	if err := r.Status().Update(ctx, businessRoute); err != nil {
		logger.Error(err, "Failed to update syncing status")
		return ctrl.Result{}, err
	}

	// 使用翻译引擎验证配置
	if _, err := translator.TranslateConfig(
		translator.VendorHuawei,
		translator.ConfigTypeRoute,
		businessRoute.Spec,
	); err != nil {
		return r.handleReconcileError(ctx, businessRoute, err)
	}

	// TODO: 实际下发 NETCONF 配置
	// 7. 模拟配置成功（后续完善实际下发逻辑）
	businessRoute.Status.Phase = bizv1.PhaseSynced
	businessRoute.Status.LastSyncTime = metav1.Now()
	businessRoute.Status.Message = "静态路由配置同步成功"
	businessRoute.Status.RetryCount = 0
	businessRoute.Status.ErrorType = ""
	businessRoute.Status.RouteStatus = bizv1.RouteStatusActive

	if err := r.Status().Update(ctx, businessRoute); err != nil {
		logger.Error(err, "Failed to update synced status")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled BusinessRoute",
		"destination", businessRoute.Spec.DestinationCIDR)

	// 8. 定期重同步（默认 10 分钟）
	return ctrl.Result{RequeueAfter: 10 * time.Minute}, nil
}

// validateRouteConfig 验证路由配置的基本合法性
func (r *BusinessRouteReconciler) validateRouteConfig(route *bizv1.BusinessRoute) error {
	// 验证设备 ID 不为空
	if route.Spec.DeviceID == "" {
		return fmt.Errorf("deviceID 不能为空")
	}

	// 验证目标 CIDR 格式
	if route.Spec.DestinationCIDR == "" {
		return fmt.Errorf("destinationCIDR 不能为空")
	}

	// 默认路由特殊处理
	if route.Spec.Type == bizv1.RouteTypeDefault {
		if route.Spec.DestinationCIDR != "0.0.0.0/0" {
			return fmt.Errorf("默认路由目标必须是 0.0.0.0/0")
		}
	}

	// 黑洞路由不需要下一跳
	if route.Spec.Type == bizv1.RouteTypeBlackhole {
		return nil
	}

	// 验证下一跳配置
	switch route.Spec.NextHopType {
	case bizv1.NextHopTypeIPAddress:
		if route.Spec.NextHopIP == "" {
			return fmt.Errorf("NextHopType=IPAddress 时 nextHopIP 不能为空")
		}
	case bizv1.NextHopTypeIFName:
		if route.Spec.OutInterface == "" {
			return fmt.Errorf("NextHopType=Interface 时 outInterface 不能为空")
		}
	default:
		// 至少需要配置一种下一跳
		if route.Spec.NextHopIP == "" && route.Spec.OutInterface == "" {
			return fmt.Errorf("必须配置 nextHopIP 或 outInterface")
		}
	}

	// 验证优先级范围
	if route.Spec.Preference > 255 {
		return fmt.Errorf("preference 范围必须在 1-255 之间")
	}

	return nil
}

// deleteRouteFromDevice 从设备删除路由配置
func (r *BusinessRouteReconciler) deleteRouteFromDevice(
	ctx context.Context,
	businessRoute *bizv1.BusinessRoute,
) error {
	// 暂时只记录日志，后续完善实际 NETCONF 下发
	logger := log.FromContext(ctx)
	logger.Info("清理设备路由配置（模拟）",
		"device", businessRoute.Spec.DeviceID,
		"destination", businessRoute.Spec.DestinationCIDR)
	return nil
}

// handleReconcileError 处理调和错误（分类 + 指数退避）
func (r *BusinessRouteReconciler) handleReconcileError(
	ctx context.Context,
	businessRoute *bizv1.BusinessRoute,
	err error,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	errorType := classifyError(err)
	businessRoute.Status.ErrorType = errorType
	businessRoute.Status.RetryCount++

	var requeueAfter time.Duration
	var result ctrl.Result

	if errorType == ErrorTypePermanent {
		businessRoute.Status.Phase = bizv1.PhaseFailed
		businessRoute.Status.Message = fmt.Sprintf("路由配置失败(永久错误): %v", err)
		logger.Error(err, "Permanent error configuring route")
		result = ctrl.Result{RequeueAfter: 30 * time.Minute}
	} else if businessRoute.Status.RetryCount >= maxRetryCount {
		businessRoute.Status.Phase = bizv1.PhaseFailed
		businessRoute.Status.Message = fmt.Sprintf("路由配置失败(已达最大重试次数): %v", err)
		logger.Error(err, "Max retry count reached for route configuration")
		result = ctrl.Result{RequeueAfter: 30 * time.Minute}
	} else {
		businessRoute.Status.Phase = bizv1.PhasePending
		businessRoute.Status.Message = fmt.Sprintf("临时错误，将重试: %v (重试次数: %d)", err, businessRoute.Status.RetryCount)
		requeueAfter = calculateBackoff(businessRoute.Status.RetryCount)
		logger.Info("Temporary error configuring route, will retry with backoff",
			"error", err, "retryCount", businessRoute.Status.RetryCount, "requeueAfter", requeueAfter)
		result = ctrl.Result{RequeueAfter: requeueAfter}
	}

	if updateErr := r.Status().Update(ctx, businessRoute); updateErr != nil {
		logger.Error(updateErr, "Failed to update error status")
	}

	return result, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *BusinessRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&bizv1.BusinessRoute{}).
		Complete(r)
}
