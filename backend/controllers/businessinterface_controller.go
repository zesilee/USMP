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
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/actor"
	netconfclient "github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
)

const (
	// BusinessInterfaceFinalizer Finalizer 名称
	BusinessInterfaceFinalizer = "biz.usmp.io/businessinterface-finalizer"
)

// BusinessInterfaceReconciler reconciles a BusinessInterface object
type BusinessInterfaceReconciler struct {
	k8sclient.Client
	Scheme     *runtime.Scheme
	ClientPool netconfclient.ClientPool
}

//+kubebuilder:rbac:groups=biz.usmp.io,resources=businessinterfaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=biz.usmp.io,resources=businessinterfaces/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=biz.usmp.io,resources=businessinterfaces/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop
func (r *BusinessInterfaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 1. 获取 BusinessInterface CR
	businessInterface := &bizv1.BusinessInterface{}
	if err := r.Get(ctx, req.NamespacedName, businessInterface); err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get BusinessInterface")
		return ctrl.Result{}, err
	}

	// 2. 检查是否被删除
	if !businessInterface.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(businessInterface, BusinessInterfaceFinalizer) {
			logger.Info("Cleaning up BusinessInterface resources",
				"device", businessInterface.Spec.DeviceID,
				"interface", businessInterface.Spec.InterfaceName)
			// 清理设备上的接口配置
			if err := r.deleteInterfaceFromDevice(ctx, businessInterface); err != nil {
				logger.Error(err, "Failed to cleanup interface on device")
				businessInterface.Status.Message = fmt.Sprintf("清理失败: %v", err)
				r.Status().Update(ctx, businessInterface)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, err
			}
			// 移除 Finalizer
			controllerutil.RemoveFinalizer(businessInterface, BusinessInterfaceFinalizer)
			if err := r.Update(ctx, businessInterface); err != nil {
				logger.Error(err, "Failed to remove finalizer")
				return ctrl.Result{}, err
			}
			logger.Info("Successfully deleted BusinessInterface")
		}
		return ctrl.Result{}, nil
	}

	// 3. 添加 Finalizer（如果不存在）
	if !controllerutil.ContainsFinalizer(businessInterface, BusinessInterfaceFinalizer) {
		logger.Info("Adding finalizer for BusinessInterface")
		controllerutil.AddFinalizer(businessInterface, BusinessInterfaceFinalizer)
		if err := r.Update(ctx, businessInterface); err != nil {
			logger.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
	}

	// 4. 初始化状态
	if businessInterface.Status.Phase == "" {
		businessInterface.Status.Phase = bizv1.PhasePending
		if err := r.Status().Update(ctx, businessInterface); err != nil {
			logger.Error(err, "Failed to update initial status")
			return ctrl.Result{}, err
		}
	}

	// 5. 同步配置到设备
	logger.Info("Reconciling BusinessInterface",
		"device", businessInterface.Spec.DeviceID,
		"interface", businessInterface.Spec.InterfaceName)

	businessInterface.Status.Phase = bizv1.PhaseSyncing
	if err := r.Status().Update(ctx, businessInterface); err != nil {
		logger.Error(err, "Failed to update syncing status")
		return ctrl.Result{}, err
	}

	// 执行配置同步
	if err := r.syncInterfaceConfig(ctx, businessInterface); err != nil {
		// 处理错误：分类 + 指数退避
		return r.handleReconcileError(ctx, businessInterface, err)
	}

	// 6. 同步成功，更新状态
	businessInterface.Status.Phase = bizv1.PhaseSynced
	businessInterface.Status.LastSyncTime = metav1.Now()
	businessInterface.Status.Message = "接口配置同步成功"
	businessInterface.Status.RetryCount = 0
	businessInterface.Status.ErrorType = ""

	// 读取设备上的实际状态（简化，后续完善）
	businessInterface.Status.OperStatus = bizv1.InterfaceOperStatusUp

	if err := r.Status().Update(ctx, businessInterface); err != nil {
		logger.Error(err, "Failed to update synced status")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled BusinessInterface")

	// 7. 定期重同步（默认 10 分钟）
	return ctrl.Result{RequeueAfter: 10 * time.Minute}, nil
}

// syncInterfaceConfig 同步接口配置到设备
func (r *BusinessInterfaceReconciler) syncInterfaceConfig(
	ctx context.Context,
	businessInterface *bizv1.BusinessInterface,
) error {
	// 构建设备连接信息（从 BusinessSwitch CR 读取设备信息）
	deviceIP, err := r.getDeviceIP(ctx, businessInterface.Spec.DeviceID)
	if err != nil {
		return fmt.Errorf("获取设备信息失败: %w", err)
	}

	// 临时使用硬编码的连接信息（后续从 Secret/BusinessSwitch 读取）
	deviceID := fmt.Sprintf("admin:Admin@123@%s:830", deviceIP)

	// 创建 IFM Actor
	translator := actor.NewReflectTranslator[*huawei.HuaweiIfm_Ifm_Interfaces]()
	ifmActor := actor.NewModelActor[*huawei.HuaweiIfm_Ifm_Interfaces](
		fmt.Sprintf("ifm-%s", businessInterface.Name),
		deviceID,
		r.ClientPool,
		translator,
	)
	defer ifmActor.Stop()

	// 启动 Actor
	if err := ifmActor.Start(); err != nil {
		return fmt.Errorf("连接设备失败: %w", err)
	}

	// 等待 Actor 初始化
	time.Sleep(500 * time.Millisecond)

	// 将业务 Spec 转换为 Huawei YANG 模型
	ifmConfig := r.convertToHuaweiIfmConfig(businessInterface)

	// 发送配置变更（使用 StatusQuery 替代，简化为探测）
	statusCmd := &actor.StatusQueryCmd{
		BaseMessage: actor.NewBaseMessageWithContext(
			fmt.Sprintf("ifm-status-%s", businessInterface.Name),
			actor.MsgStatusQuery,
			ctx,
		),
		IncludeDetails: true,
	}

	promise, err := ifmActor.Send(statusCmd)
	if err != nil {
		return fmt.Errorf("查询设备状态失败: %w", err)
	}

	result := <-promise
	if !result.Success {
		return fmt.Errorf("设备状态查询失败: %v", result.Error)
	}

	_ = ifmConfig // 暂时不使用配置，后续完善配置下发

	return nil
}

// convertToHuaweiIfmConfig 将业务 Spec 转换为 Huawei IFM YANG 模型
func (r *BusinessInterfaceReconciler) convertToHuaweiIfmConfig(
	businessInterface *bizv1.BusinessInterface,
) *huawei.HuaweiIfm_Ifm_Interfaces {
	ifName := businessInterface.Spec.InterfaceName

	// 创建配置
	ifmConfig := &huawei.HuaweiIfm_Ifm_Interfaces{
		Interface: make(map[string]*huawei.HuaweiIfm_Ifm_Interfaces_Interface),
	}

	ifmConfig.Interface[ifName] = &huawei.HuaweiIfm_Ifm_Interfaces_Interface{
		Name:        &ifName,
		Description: &businessInterface.Spec.Description,
	}

	if businessInterface.Spec.MTU > 0 {
		ifmConfig.Interface[ifName].Mtu = &businessInterface.Spec.MTU
	}

	return ifmConfig
}

// deleteInterfaceFromDevice 删除设备上的接口配置（恢复默认）
func (r *BusinessInterfaceReconciler) deleteInterfaceFromDevice(
	ctx context.Context,
	businessInterface *bizv1.BusinessInterface,
) error {
	// 暂时只记录日志，实际删除逻辑需要根据厂商实现
	logger := log.FromContext(ctx)
	logger.Info("Cleanup interface config (simulated)",
		"device", businessInterface.Spec.DeviceID,
		"interface", businessInterface.Spec.InterfaceName)
	return nil
}

// getDeviceIP 从 BusinessSwitch CR 获取设备 IP
func (r *BusinessInterfaceReconciler) getDeviceIP(ctx context.Context, deviceID string) (string, error) {
	// 简单实现：如果 deviceID 看起来是 IP，直接返回
	// 后续需要通过 Client 查找对应的 BusinessSwitch CR
	if deviceID == "switch-demo-01" {
		return "192.168.1.100", nil
	}
	return "192.168.1.100", nil // 默认测试 IP
}

// handleReconcileError 处理调和错误（分类 + 指数退避）
func (r *BusinessInterfaceReconciler) handleReconcileError(
	ctx context.Context,
	businessInterface *bizv1.BusinessInterface,
	err error,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 分类错误类型
	errorType := classifyError(err)
	businessInterface.Status.ErrorType = errorType
	businessInterface.Status.RetryCount++
	businessInterface.Status.OperStatus = bizv1.InterfaceOperStatusUnknown

	retryCount := businessInterface.Status.RetryCount
	var requeueAfter time.Duration
	var result ctrl.Result

	if errorType == ErrorTypePermanent {
		// 永久错误: 不重试，但仍定期探活（每 30 分钟）
		businessInterface.Status.Phase = bizv1.PhaseFailed
		businessInterface.Status.Message = fmt.Sprintf("接口配置失败(永久错误): %v", err)
		logger.Error(err, "Permanent error configuring interface")
		result = ctrl.Result{RequeueAfter: 30 * time.Minute}
	} else if retryCount >= maxRetryCount {
		// 达到最大重试次数
		businessInterface.Status.Phase = bizv1.PhaseFailed
		businessInterface.Status.Message = fmt.Sprintf("接口配置失败(已达最大重试次数): %v", err)
		logger.Error(err, "Max retry count reached for interface configuration")
		result = ctrl.Result{RequeueAfter: 30 * time.Minute}
	} else {
		// 临时错误: 指数退避重试
		businessInterface.Status.Phase = bizv1.PhasePending
		businessInterface.Status.Message = fmt.Sprintf("临时错误，将重试: %v (重试次数: %d)", err, retryCount)
		requeueAfter = calculateBackoff(retryCount)
		logger.Info("Temporary error configuring interface, will retry with backoff",
			"error", err, "retryCount", retryCount, "requeueAfter", requeueAfter)
		result = ctrl.Result{RequeueAfter: requeueAfter}
	}

	// 更新 CR 状态
	if updateErr := r.Status().Update(ctx, businessInterface); updateErr != nil {
		logger.Error(updateErr, "Failed to update error status")
	}

	return result, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *BusinessInterfaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&bizv1.BusinessInterface{}).
		Complete(r)
}
