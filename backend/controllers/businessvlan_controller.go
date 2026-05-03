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
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	bizv1 "github.com/leezesi/usmp/backend/api/v1"
	"github.com/leezesi/usmp/backend/pkg/translator"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/actor"
	netconfclient "github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
)

const (
	// BusinessVlanFinalizer Finalizer 名称
	BusinessVlanFinalizer = "biz.usmp.io/businessvlan-finalizer"
	// 错误类型分类
	ErrorTypeTemporary = "Temporary"
	ErrorTypePermanent = "Permanent"
	// 最大重试次数
	maxRetryCount = 5
	// 初始重试间隔
	baseRetryInterval = 5 * time.Second
)

// BusinessVlanReconciler reconciles a BusinessVlan object
// 仅支持华为交换机 VLAN 配置
type BusinessVlanReconciler struct {
	k8sclient.Client
	Scheme     *runtime.Scheme
	ClientPool netconfclient.ClientPool
}

//+kubebuilder:rbac:groups=biz.usmp.io,resources=businessvlans,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=biz.usmp.io,resources=businessvlans/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=biz.usmp.io,resources=businessvlans/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *BusinessVlanReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 1. 获取 BusinessVlan CR
	businessVlan := &bizv1.BusinessVlan{}
	if err := r.Get(ctx, req.NamespacedName, businessVlan); err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get BusinessVlan")
		return ctrl.Result{}, err
	}

	// 2. 检查是否被删除（DeletionTimestamp 非空）
	if !businessVlan.DeletionTimestamp.IsZero() {
		// 包含 Finalizer，执行删除逻辑
		if controllerutil.ContainsFinalizer(businessVlan, BusinessVlanFinalizer) {
			logger.Info("Deleting VLAN from device", "vlanID", businessVlan.Spec.VlanID, "device", businessVlan.Spec.DeviceID)
			if err := r.deleteVlanFromDevice(ctx, businessVlan); err != nil {
				logger.Error(err, "Failed to delete VLAN from device")
				businessVlan.Status.Phase = bizv1.PhaseFailed
				businessVlan.Status.Message = fmt.Sprintf("VLAN 删除失败: %v", err)
				r.Status().Update(ctx, businessVlan)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, err
			}

			// 移除 Finalizer，允许 K8s 删除 CR
			controllerutil.RemoveFinalizer(businessVlan, BusinessVlanFinalizer)
			if err := r.Update(ctx, businessVlan); err != nil {
				logger.Error(err, "Failed to remove finalizer")
				return ctrl.Result{}, err
			}
			logger.Info("Successfully deleted VLAN", "vlanID", businessVlan.Spec.VlanID)
		}
		return ctrl.Result{}, nil
	}

	// 3. 添加 Finalizer（如果不存在）
	if !controllerutil.ContainsFinalizer(businessVlan, BusinessVlanFinalizer) {
		logger.Info("Adding finalizer for BusinessVlan")
		controllerutil.AddFinalizer(businessVlan, BusinessVlanFinalizer)
		if err := r.Update(ctx, businessVlan); err != nil {
			logger.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
	}

	// 4. 初始化状态
	if businessVlan.Status.Phase == "" {
		businessVlan.Status.Phase = bizv1.PhasePending
		if err := r.Status().Update(ctx, businessVlan); err != nil {
			logger.Error(err, "Failed to update initial status")
			return ctrl.Result{}, err
		}
	}

	// 5. 为该设备创建 VLAN Actor
	vlanActor, err := r.createVlanActor(businessVlan.Spec.DeviceID)
	if err != nil {
		return r.handleReconcileError(ctx, businessVlan, fmt.Errorf("Actor 初始化失败: %w", err))
	}
	defer vlanActor.Stop()

	// 6. 将业务配置翻译为华为 YANG 格式
	err = r.translateBusinessVlanToHuawei(ctx, vlanActor, businessVlan)
	if err != nil {
		return r.handleReconcileError(ctx, businessVlan, fmt.Errorf("配置翻译失败: %w", err))
	}

	// 7. 更新状态为同步中
	businessVlan.Status.Phase = bizv1.PhaseSyncing
	if err := r.Status().Update(ctx, businessVlan); err != nil {
		logger.Error(err, "Failed to update syncing status")
		return ctrl.Result{}, err
	}

	// 8. Prepare 阶段 - 校验配置并写入 Candidate
	prepareCmd := &actor.PrepareCmd{
		BaseMessage: actor.NewBaseMessageWithContext(fmt.Sprintf("prepare-%d", businessVlan.Spec.VlanID), actor.MsgPrepare, ctx),
		DryRun:      false,
	}

	promise, err := vlanActor.Send(prepareCmd)
	if err != nil {
		return r.handleReconcileError(ctx, businessVlan, fmt.Errorf("发送 Prepare 失败: %w", err))
	}

	result := <-promise
	if !result.Success {
		return r.handleReconcileError(ctx, businessVlan, fmt.Errorf("prepare 失败: %v", result.Error))
	}

	// 9. Commit 阶段 - 应用配置到 Running
	commitCmd := &actor.CommitCmd{
		BaseMessage: actor.NewBaseMessageWithContext(fmt.Sprintf("commit-%d", businessVlan.Spec.VlanID), actor.MsgCommit, ctx),
		ForceCommit: false,
	}

	promise, err = vlanActor.Send(commitCmd)
	if err != nil {
		logger.Error(err, "Failed to send Commit command")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	}

	result = <-promise
	if !result.Success {
		return r.handleReconcileError(ctx, businessVlan, fmt.Errorf("commit 失败: %v", result.Error))
	}

	// 10. 从设备读取真实状态回填
	actualStatus, err := r.fetchActualVlanStatus(ctx, vlanActor, businessVlan.Spec.VlanID)
	if err != nil {
		logger.Error(err, "Failed to fetch actual status, but config was applied")
	} else {
		businessVlan.Status.Actual = actualStatus
	}

	// 11. 同步成功，清零重试次数
	businessVlan.Status.RetryCount = 0
	businessVlan.Status.ErrorType = ""
	businessVlan.Status.Phase = bizv1.PhaseSynced
	businessVlan.Status.LastSyncTime = metav1.Now()
	businessVlan.Status.Message = "华为交换机 VLAN 配置同步成功"
	businessVlan.Status.ConfigVersion = businessVlan.Generation

	if err := r.Status().Update(ctx, businessVlan); err != nil {
		logger.Error(err, "Failed to update final status")
		return ctrl.Result{}, err
	}

	logger.Info("成功同步华为交换机 VLAN 配置",
		"vlanID", businessVlan.Spec.VlanID,
		"device", businessVlan.Spec.DeviceID)

	// 12. 5分钟后重同步，确保配置一致性
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// createVlanActor 为华为设备创建 VLAN Actor
func (r *BusinessVlanReconciler) createVlanActor(deviceID string) (*actor.ModelActor[*huawei.HuaweiVlan_Vlan_Vlans], error) {
	// 使用反射 Translator
	translator := actor.NewReflectTranslator[*huawei.HuaweiVlan_Vlan_Vlans]()

	vlanActor := actor.NewModelActor[*huawei.HuaweiVlan_Vlan_Vlans](
		fmt.Sprintf("huawei-vlan-%s", deviceID),
		deviceID,
		r.ClientPool,
		translator,
	)

	// 启动 Actor
	if err := vlanActor.Start(); err != nil {
		return nil, fmt.Errorf("启动 VLAN Actor 失败: %w", err)
	}

	// 等待 Actor 初始化完成
	time.Sleep(100 * time.Millisecond)

	return vlanActor, nil
}

// translateBusinessVlanToHuawei 将业务配置翻译为华为 YANG 格式
// 使用统一的翻译引擎进行 CRD Spec → 厂商 YANG 转换
func (r *BusinessVlanReconciler) translateBusinessVlanToHuawei(
	ctx context.Context,
	vlanActor *actor.ModelActor[*huawei.HuaweiVlan_Vlan_Vlans],
	businessVlan *bizv1.BusinessVlan,
) error {
	// 使用统一翻译引擎进行配置转换
	yangConfig, err := translator.TranslateConfig(
		translator.VendorHuawei,
		translator.ConfigTypeVlan,
		businessVlan.Spec,
	)
	if err != nil {
		return fmt.Errorf("配置翻译失败: %w", err)
	}

	// 将翻译结果转换为 map 格式发送给 Actor
	// 由于当前 ModelActor 使用反射处理，这里直接发送翻译后的 YANG 结构体
	_ = yangConfig // 暂时标记为使用，后续完善发送逻辑

	// 注意：当前实现使用旧的 payload 方式，后续需要重构为直接使用翻译引擎
	// 临时兼容实现：构建华为 VLAN Payload
	payload := map[string]interface{}{
		"Id":          businessVlan.Spec.VlanID,
		"Name":        businessVlan.Spec.Name,
		"Description": businessVlan.Spec.Description,
	}

	// 映射 VLAN 类型 (使用数值枚举)
	switch businessVlan.Spec.Type {
	case bizv1.VlanTypeCommon, "":
		payload["Type"] = 1 // Common
	case bizv1.VlanTypeSuper:
		payload["Type"] = 2 // Super
	case bizv1.VlanTypeSub:
		payload["Type"] = 3 // Sub
	}

	// MAC 地址学习开关
	if businessVlan.Spec.MacLearningEnabled != nil {
		if *businessVlan.Spec.MacLearningEnabled {
			payload["MacLearning"] = 1 // Enable
		} else {
			payload["MacLearning"] = 2 // Disable
		}
	}

	// 统计开关
	if businessVlan.Spec.StatisticEnabled != nil {
		if *businessVlan.Spec.StatisticEnabled {
			payload["StatisticEnable"] = 1 // Enable
		} else {
			payload["StatisticEnable"] = 2 // Disable
		}
	}

	// 广播丢弃开关
	if businessVlan.Spec.BroadcastDiscardEnabled != nil {
		if *businessVlan.Spec.BroadcastDiscardEnabled {
			payload["BroadcastDiscard"] = 1 // Enable
		} else {
			payload["BroadcastDiscard"] = 2 // Disable
		}
	}

	// 发送 Translate 命令
	translateCmd := &actor.TranslateCmd{
		BaseMessage: actor.NewBaseMessageWithContext(
			fmt.Sprintf("translate-vlan-%d", businessVlan.Spec.VlanID),
			actor.MsgTranslate,
			ctx,
		),
		Payload:   payload,
		Operation: actor.OperationMerge,
	}

	promise, err := vlanActor.Send(translateCmd)
	if err != nil {
		return err
	}

	result := <-promise
	if !result.Success {
		return fmt.Errorf("translate 失败: %v", result.Error)
	}

	return nil
}


// deleteVlanFromDevice 从华为交换机删除指定 VLAN
func (r *BusinessVlanReconciler) deleteVlanFromDevice(
	ctx context.Context,
	businessVlan *bizv1.BusinessVlan,
) error {
	// 1. 创建 VLAN Actor
	vlanActor, err := r.createVlanActor(businessVlan.Spec.DeviceID)
	if err != nil {
		return fmt.Errorf("创建 Actor 失败: %w", err)
	}
	defer vlanActor.Stop()

	// 2. 翻译为删除操作 (VLAN ID 作为标识)
	deletePayload := map[string]interface{}{
		"Id": businessVlan.Spec.VlanID,
	}

	translateCmd := &actor.TranslateCmd{
		BaseMessage: actor.NewBaseMessageWithContext(
			fmt.Sprintf("translate-delete-vlan-%d", businessVlan.Spec.VlanID),
			actor.MsgTranslate,
			ctx,
		),
		Payload:   deletePayload,
		Operation: actor.OperationDelete,
	}

	promise, err := vlanActor.Send(translateCmd)
	if err != nil {
		return err
	}

	result := <-promise
	if !result.Success {
		return fmt.Errorf("translate 失败: %v", result.Error)
	}

	// 3. Prepare 阶段
	prepareCmd := &actor.PrepareCmd{
		BaseMessage: actor.NewBaseMessageWithContext(
			fmt.Sprintf("prepare-delete-%d", businessVlan.Spec.VlanID),
			actor.MsgPrepare,
			ctx,
		),
		DryRun: false,
	}

	promise, err = vlanActor.Send(prepareCmd)
	if err != nil {
		return err
	}

	result = <-promise
	if !result.Success {
		return fmt.Errorf("prepare 失败: %v", result.Error)
	}

	// 4. Commit 阶段
	commitCmd := &actor.CommitCmd{
		BaseMessage: actor.NewBaseMessageWithContext(
			fmt.Sprintf("commit-delete-%d", businessVlan.Spec.VlanID),
			actor.MsgCommit,
			ctx,
		),
		ForceCommit: false,
	}

	promise, err = vlanActor.Send(commitCmd)
	if err != nil {
		return err
	}

	result = <-promise
	if !result.Success {
		return fmt.Errorf("commit 失败: %v", result.Error)
	}

	return nil
}

// fetchActualVlanStatus 从华为交换机获取指定 VLAN 的真实状态
func (r *BusinessVlanReconciler) fetchActualVlanStatus(
	ctx context.Context,
	vlanActor *actor.ModelActor[*huawei.HuaweiVlan_Vlan_Vlans],
	vlanID uint16,
) (*bizv1.VlanStatus, error) {
	// 发送 StatusQuery 命令获取实际配置
	statusCmd := &actor.StatusQueryCmd{
		BaseMessage: actor.NewBaseMessageWithContext(
			fmt.Sprintf("status-vlan-%d", vlanID),
			actor.MsgStatusQuery,
			ctx,
		),
		IncludeDetails: true, // 触发实际配置读取
	}

	promise, err := vlanActor.Send(statusCmd)
	if err != nil {
		return nil, fmt.Errorf("获取状态失败: %w", err)
	}

	result := <-promise
	if !result.Success {
		return nil, fmt.Errorf("状态查询失败: %v", result.Error)
	}

	// 从返回数据中解析 VLAN 状态
	status := &bizv1.VlanStatus{
		VlanID: vlanID,
	}

	// 从 actual_config 中提取 VLAN 信息
	if actualConfig, ok := result.Data["actual_config"]; ok {
		// 提取 HuaweiVlan_Vlan_Vlans 结构体信息
		// 简化处理，先返回基本状态
		status.OperStatus = "SyncOk"
		_ = actualConfig // 标记为已使用，后续可扩展
	}
	if fetchErr, ok := result.Data["fetch_error"].(string); ok {
		status.OperStatus = "FetchError: " + fetchErr
	}

	return status, nil
}

// classifyError 分类错误类型: 临时错误 vs 永久错误
func classifyError(err error) string {
	// 网络连接错误通常是临时的
	if strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "i/o timeout") ||
		strings.Contains(err.Error(), "connection reset") ||
		strings.Contains(err.Error(), "no route to host") ||
		strings.Contains(err.Error(), "network is unreachable") {
		return ErrorTypeTemporary
	}

	// 认证错误通常是配置问题（永久）
	if strings.Contains(err.Error(), "authentication failed") ||
		strings.Contains(err.Error(), "permission denied") ||
		strings.Contains(err.Error(), "unauthorized") {
		return ErrorTypePermanent
	}

	// NETCONF 协议错误通常是配置问题（永久）
	if strings.Contains(err.Error(), "rpc-error") ||
		strings.Contains(err.Error(), "invalid value") ||
		strings.Contains(err.Error(), "bad attribute") ||
		strings.Contains(err.Error(), "unknown element") {
		return ErrorTypePermanent
	}

	// DNS 解析错误可能是临时的
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return ErrorTypeTemporary
	}

	// 默认视为临时错误，给予重试机会
	return ErrorTypeTemporary
}

// calculateBackoff 计算指数退避的重队列时间
func calculateBackoff(retryCount int) time.Duration {
	if retryCount <= 0 {
		return baseRetryInterval
	}
	// 指数退避: 5s, 10s, 20s, 40s, 60s (最大值)
	backoff := baseRetryInterval * time.Duration(1<<retryCount)
	maxBackoff := 60 * time.Second
	if backoff > maxBackoff {
		return maxBackoff
	}
	return backoff
}

// handleReconcileError 处理 Reconcile 错误，更新状态并计算重队列时间
func (r *BusinessVlanReconciler) handleReconcileError(
	ctx context.Context,
	businessVlan *bizv1.BusinessVlan,
	err error,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 分类错误类型
	errorType := classifyError(err)
	businessVlan.Status.ErrorType = errorType

	// 更新重试次数
	businessVlan.Status.RetryCount++
	retryCount := businessVlan.Status.RetryCount

	// 计算下一次重试时间
	var requeueAfter time.Duration
	var result ctrl.Result

	if errorType == ErrorTypePermanent {
		// 永久错误: 标记为失败，不再重试（除非配置变更）
		businessVlan.Status.Phase = bizv1.PhaseFailed
		businessVlan.Status.Message = fmt.Sprintf("配置失败(永久错误): %v", err)
		logger.Error(err, "Permanent error - will not retry automatically")
		result = ctrl.Result{} // 不再自动重试
	} else if retryCount >= maxRetryCount {
		// 达到最大重试次数
		businessVlan.Status.Phase = bizv1.PhaseFailed
		businessVlan.Status.Message = fmt.Sprintf("配置失败(已达最大重试次数): %v", err)
		logger.Error(err, "Max retry count reached")
		result = ctrl.Result{} // 不再自动重试
	} else {
		// 临时错误: 指数退避重试
		businessVlan.Status.Phase = bizv1.PhasePending
		businessVlan.Status.Message = fmt.Sprintf("临时错误，将重试: %v (重试次数: %d)", err, retryCount)
		requeueAfter = calculateBackoff(retryCount)
		logger.Info("Temporary error, will retry with backoff",
			"error", err, "retryCount", retryCount, "requeueAfter", requeueAfter)
		result = ctrl.Result{RequeueAfter: requeueAfter}
	}

	// 更新 CR 状态
	if updateErr := r.Status().Update(ctx, businessVlan); updateErr != nil {
		logger.Error(updateErr, "Failed to update error status")
	}

	return result, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *BusinessVlanReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&bizv1.BusinessVlan{}).
		Complete(r)
}
