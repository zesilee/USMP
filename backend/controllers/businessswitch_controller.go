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
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/actor"
	netconfclient "github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
)

const (
	// BusinessSwitchFinalizer Finalizer 名称
	BusinessSwitchFinalizer = "biz.usmp.io/businessswitch-finalizer"
)

// BusinessSwitchReconciler reconciles a BusinessSwitch object
type BusinessSwitchReconciler struct {
	k8sclient.Client
	Scheme     *runtime.Scheme
	ClientPool netconfclient.ClientPool
}

//+kubebuilder:rbac:groups=biz.usmp.io,resources=businessswitches,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=biz.usmp.io,resources=businessswitches/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=biz.usmp.io,resources=businessswitches/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *BusinessSwitchReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 1. 获取 BusinessSwitch CR
	businessSwitch := &bizv1.BusinessSwitch{}
	if err := r.Get(ctx, req.NamespacedName, businessSwitch); err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get BusinessSwitch")
		return ctrl.Result{}, err
	}

	// 2. 检查是否被删除
	if !businessSwitch.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(businessSwitch, BusinessSwitchFinalizer) {
			logger.Info("Cleaning up BusinessSwitch resources", "device", businessSwitch.Spec.DeviceIP)
			// 移除 Finalizer
			controllerutil.RemoveFinalizer(businessSwitch, BusinessSwitchFinalizer)
			if err := r.Update(ctx, businessSwitch); err != nil {
				logger.Error(err, "Failed to remove finalizer")
				return ctrl.Result{}, err
			}
			logger.Info("Successfully deleted BusinessSwitch", "device", businessSwitch.Spec.DeviceIP)
		}
		return ctrl.Result{}, nil
	}

	// 3. 添加 Finalizer（如果不存在）
	if !controllerutil.ContainsFinalizer(businessSwitch, BusinessSwitchFinalizer) {
		logger.Info("Adding finalizer for BusinessSwitch")
		controllerutil.AddFinalizer(businessSwitch, BusinessSwitchFinalizer)
		if err := r.Update(ctx, businessSwitch); err != nil {
			logger.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
	}

	// 4. 初始化状态
	if businessSwitch.Status.Phase == "" {
		businessSwitch.Status.Phase = bizv1.PhasePending
		businessSwitch.Status.OnlineStatus = bizv1.DeviceUnknown
		if err := r.Status().Update(ctx, businessSwitch); err != nil {
			logger.Error(err, "Failed to update initial status")
			return ctrl.Result{}, err
		}
	}

	// 5. 检查是否启用
	if !businessSwitch.Spec.Enabled {
		logger.Info("Device is disabled, skipping sync", "device", businessSwitch.Spec.DeviceIP)
		businessSwitch.Status.Phase = bizv1.PhasePending
		businessSwitch.Status.Message = "设备已禁用，跳过同步"
		r.Status().Update(ctx, businessSwitch)
		// 每 10 分钟检查一次是否启用
		return ctrl.Result{RequeueAfter: 10 * time.Minute}, nil
	}

	// 6. 设备连接测试与状态采集
	logger.Info("Checking device connectivity", "device", businessSwitch.Spec.DeviceIP)
	err := r.probeDevice(ctx, businessSwitch)
	if err != nil {
		// 设备探测失败，按错误类型处理重试
		return r.handleProbeError(ctx, businessSwitch, err)
	}

	// 7. 探测成功，更新状态
	businessSwitch.Status.Phase = bizv1.PhaseSynced
	businessSwitch.Status.OnlineStatus = bizv1.DeviceOnline
	businessSwitch.Status.LastSeenTime = metav1.Now()
	businessSwitch.Status.LastSyncTime = metav1.Now()
	businessSwitch.Status.Message = "设备在线，连接正常"
	businessSwitch.Status.RetryCount = 0
	businessSwitch.Status.ErrorType = ""

	if err := r.Status().Update(ctx, businessSwitch); err != nil {
		logger.Error(err, "Failed to update sync status")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully probed device", "device", businessSwitch.Spec.DeviceIP)

	// 8. 定期重同步（根据配置的间隔，默认 5 分钟）
	syncInterval := time.Duration(businessSwitch.Spec.SyncInterval) * time.Minute
	if syncInterval <= 0 {
		syncInterval = 5 * time.Minute
	}
	return ctrl.Result{RequeueAfter: syncInterval}, nil
}

// probeDevice 探测设备连接状态并采集基本信息
func (r *BusinessSwitchReconciler) probeDevice(
	ctx context.Context,
	businessSwitch *bizv1.BusinessSwitch,
) error {
	// 构建设备连接信息
	deviceAddr := fmt.Sprintf("%s:%d", businessSwitch.Spec.DeviceIP, businessSwitch.Spec.Port)
	if businessSwitch.Spec.Port == 0 {
		deviceAddr = fmt.Sprintf("%s:830", businessSwitch.Spec.DeviceIP)
	}

	deviceID := fmt.Sprintf("%s:%s@%s",
		businessSwitch.Spec.Credentials.Username,
		businessSwitch.Spec.Credentials.Password,
		deviceAddr,
	)

	// 创建 VLAN Actor 用于探测（复用现有连接机制）
	translator := actor.NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans]()
	vlanActor := actor.NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans](
		fmt.Sprintf("probe-%s", businessSwitch.Name),
		deviceID,
		r.ClientPool,
		translator,
	)
	defer vlanActor.Stop()

	// 启动 Actor 并等待初始化
	if err := vlanActor.Start(); err != nil {
		return fmt.Errorf("连接设备失败: %w", err)
	}

	// 等待 Actor 初始化
	time.Sleep(500 * time.Millisecond)

	// 发送 StatusQuery 探测
	statusCmd := &actor.StatusQueryCmd{
		BaseMessage: actor.NewBaseMessageWithContext(
			fmt.Sprintf("probe-%s", businessSwitch.Name),
			actor.MsgStatusQuery,
			ctx,
		),
		IncludeDetails: true,
	}

	promise, err := vlanActor.Send(statusCmd)
	if err != nil {
		return fmt.Errorf("探测命令发送失败: %w", err)
	}

	result := <-promise
	if !result.Success {
		return fmt.Errorf("设备探测失败: %v", result.Error)
	}

	// 从返回结果中提取设备信息（简化处理）
	data := result.Data
	if data != nil {
		// 更新硬件状态
		hardware := &bizv1.DeviceHardwareStatus{}
		if uptime, ok := data["uptime"].(string); ok {
			hardware.Uptime = uptime
		}
		businessSwitch.Status.Hardware = hardware
	}

	return nil
}

// handleProbeError 处理设备探测错误
func (r *BusinessSwitchReconciler) handleProbeError(
	ctx context.Context,
	businessSwitch *bizv1.BusinessSwitch,
	err error,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 分类错误类型
	errorType := classifyError(err)
	businessSwitch.Status.ErrorType = errorType
	businessSwitch.Status.RetryCount++
	businessSwitch.Status.OnlineStatus = bizv1.DeviceOffline
	businessSwitch.Status.LastSeenTime = metav1.Time{} // 清空在线时间

	retryCount := businessSwitch.Status.RetryCount
	var requeueAfter time.Duration
	var result ctrl.Result

	if errorType == ErrorTypePermanent {
		// 永久错误: 不重试，但仍定期探活（每 30 分钟）
		businessSwitch.Status.Phase = bizv1.PhaseFailed
		businessSwitch.Status.Message = fmt.Sprintf("设备连接失败(永久错误): %v", err)
		logger.Error(err, "Permanent error connecting to device")
		result = ctrl.Result{RequeueAfter: 30 * time.Minute}
	} else if retryCount >= maxRetryCount {
		// 达到最大重试次数
		businessSwitch.Status.Phase = bizv1.PhaseFailed
		businessSwitch.Status.Message = fmt.Sprintf("设备连接失败(已达最大重试次数): %v", err)
		logger.Error(err, "Max retry count reached for device probe")
		result = ctrl.Result{RequeueAfter: 30 * time.Minute}
	} else {
		// 临时错误: 指数退避重试
		businessSwitch.Status.Phase = bizv1.PhasePending
		businessSwitch.Status.Message = fmt.Sprintf("临时错误，将重试: %v (重试次数: %d)", err, retryCount)
		requeueAfter = calculateBackoff(retryCount)
		logger.Info("Temporary error probing device, will retry with backoff",
			"error", err, "retryCount", retryCount, "requeueAfter", requeueAfter)
		result = ctrl.Result{RequeueAfter: requeueAfter}
	}

	// 更新 CR 状态
	if updateErr := r.Status().Update(ctx, businessSwitch); updateErr != nil {
		logger.Error(updateErr, "Failed to update error status")
	}

	return result, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *BusinessSwitchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&bizv1.BusinessSwitch{}).
		Complete(r)
}
