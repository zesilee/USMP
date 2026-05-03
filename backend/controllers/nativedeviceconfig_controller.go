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
	"crypto/sha256"
	"encoding/hex"
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
	netconfclient "github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
)

const (
	// NativeDeviceConfigFinalizer Finalizer 名称
	NativeDeviceConfigFinalizer = "biz.usmp.io/nativedeviceconfig-finalizer"
)

// NativeDeviceConfigReconciler reconciles a NativeDeviceConfig object
type NativeDeviceConfigReconciler struct {
	k8sclient.Client
	Scheme     *runtime.Scheme
	ClientPool netconfclient.ClientPool
}

//+kubebuilder:rbac:groups=biz.usmp.io,resources=nativedeviceconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=biz.usmp.io,resources=nativedeviceconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=biz.usmp.io,resources=nativedeviceconfigs/finalizers,verbs=update

// Reconcile 原生配置调和循环
func (r *NativeDeviceConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 1. 获取 NativeDeviceConfig CR
	nativeConfig := &bizv1.NativeDeviceConfig{}
	if err := r.Get(ctx, req.NamespacedName, nativeConfig); err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get NativeDeviceConfig")
		return ctrl.Result{}, err
	}

	// 2. 检查是否被删除
	if !nativeConfig.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(nativeConfig, NativeDeviceConfigFinalizer) {
			logger.Info("清理原生配置",
				"device", nativeConfig.Spec.DeviceID,
				"format", nativeConfig.Spec.Format)
			// 移除 Finalizer
			controllerutil.RemoveFinalizer(nativeConfig, NativeDeviceConfigFinalizer)
			if err := r.Update(ctx, nativeConfig); err != nil {
				logger.Error(err, "Failed to remove finalizer")
				return ctrl.Result{}, err
			}
			logger.Info("Successfully deleted NativeDeviceConfig")
		}
		return ctrl.Result{}, nil
	}

	// 3. 添加 Finalizer（如果不存在）
	if !controllerutil.ContainsFinalizer(nativeConfig, NativeDeviceConfigFinalizer) {
		logger.Info("Adding finalizer for NativeDeviceConfig")
		controllerutil.AddFinalizer(nativeConfig, NativeDeviceConfigFinalizer)
		if err := r.Update(ctx, nativeConfig); err != nil {
			logger.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
	}

	// 4. 初始化状态
	if nativeConfig.Status.Phase == "" {
		nativeConfig.Status.Phase = bizv1.PhasePending
		nativeConfig.Status.ExecutionStatus = bizv1.ExecutionStatusPending
		if err := r.Status().Update(ctx, nativeConfig); err != nil {
			logger.Error(err, "Failed to update initial status")
			return ctrl.Result{}, err
		}
	}

	// 5. 对于 Once 模式，如果已经成功执行过且配置没有变化，跳过
	configHash := calculateConfigHash(nativeConfig.Spec.Content)
	if nativeConfig.Spec.ExecutionMode == bizv1.ExecutionModeOnce &&
		nativeConfig.Status.ExecutionStatus == bizv1.ExecutionStatusSucceeded &&
		nativeConfig.Status.ConfigHash == configHash {
		logger.Info("配置已成功执行且无变化，跳过",
			"configHash", configHash)
		return ctrl.Result{}, nil
	}

	// 6. 验证配置
	if err := r.validateNativeConfig(nativeConfig); err != nil {
		nativeConfig.Status.Phase = bizv1.PhaseFailed
		nativeConfig.Status.Message = fmt.Sprintf("配置验证失败: %v", err)
		nativeConfig.Status.ErrorType = ErrorTypePermanent
		nativeConfig.Status.ExecutionStatus = bizv1.ExecutionStatusFailed
		r.Status().Update(ctx, nativeConfig)
		return ctrl.Result{}, err
	}

	// 7. 执行配置下发
	logger.Info("执行原生配置下发",
		"device", nativeConfig.Spec.DeviceID,
		"format", nativeConfig.Spec.Format,
		"executionMode", nativeConfig.Spec.ExecutionMode)

	nativeConfig.Status.Phase = bizv1.PhaseSyncing
	nativeConfig.Status.ExecutionStatus = bizv1.ExecutionStatusRunning
	nativeConfig.Status.ExecutionStartTime = metav1.Now()
	if err := r.Status().Update(ctx, nativeConfig); err != nil {
		logger.Error(err, "Failed to update running status")
		return ctrl.Result{}, err
	}

	// 执行配置下发（模拟）
	startTime := time.Now()
	if err := r.applyNativeConfig(ctx, nativeConfig); err != nil {
		return r.handleReconcileError(ctx, nativeConfig, err, startTime)
	}

	// 8. 执行成功，更新状态
	durationMs := time.Since(startTime).Milliseconds()

	nativeConfig.Status.Phase = bizv1.PhaseSynced
	nativeConfig.Status.LastSyncTime = metav1.Now()
	nativeConfig.Status.Message = fmt.Sprintf("配置下发成功，格式: %s", nativeConfig.Spec.Format)
	nativeConfig.Status.RetryCount = 0
	nativeConfig.Status.ErrorType = ""
	nativeConfig.Status.ExecutionStatus = bizv1.ExecutionStatusSucceeded
	nativeConfig.Status.ExecutionEndTime = metav1.Now()
	nativeConfig.Status.ExecutionDurationMs = durationMs
	nativeConfig.Status.ConfigHash = configHash
	nativeConfig.Status.AppliedOnDevice = true
	nativeConfig.Status.DeviceResponse = "Configuration applied successfully"

	if err := r.Status().Update(ctx, nativeConfig); err != nil {
		logger.Error(err, "Failed to update synced status")
		return ctrl.Result{}, err
	}

	logger.Info("原生配置下发成功",
		"device", nativeConfig.Spec.DeviceID,
		"format", nativeConfig.Spec.Format,
		"durationMs", durationMs)

	// 9. 根据执行模式决定是否重同步
	if nativeConfig.Spec.ExecutionMode == bizv1.ExecutionModePersistent {
		// 持久化模式：每 30 分钟检查一次配置是否仍然存在
		return ctrl.Result{RequeueAfter: 30 * time.Minute}, nil
	}

	// Once 模式：不再重同步
	return ctrl.Result{}, nil
}

// validateNativeConfig 验证原生配置
func (r *NativeDeviceConfigReconciler) validateNativeConfig(config *bizv1.NativeDeviceConfig) error {
	// 验证设备 ID
	if config.Spec.DeviceID == "" {
		return fmt.Errorf("deviceID 不能为空")
	}

	// 验证配置格式
	switch config.Spec.Format {
	case bizv1.ConfigFormatCLI, bizv1.ConfigFormatYANG, bizv1.ConfigFormatXML, bizv1.ConfigFormatJSON:
		// 支持的格式
	default:
		return fmt.Errorf("不支持的配置格式: %s", config.Spec.Format)
	}

	// 验证配置内容
	if config.Spec.Content == "" {
		return fmt.Errorf("配置内容不能为空")
	}

	// 加密配置必须提供密钥引用
	if config.Spec.Encrypted && config.Spec.KeySecretRef == "" {
		return fmt.Errorf("加密配置必须提供密钥引用 (keySecretRef)")
	}

	// 超时范围验证
	if config.Spec.TimeoutSeconds < 0 {
		return fmt.Errorf("超时时间不能为负数")
	}

	// 重试次数验证
	if config.Spec.MaxRetries < 0 {
		return fmt.Errorf("重试次数不能为负数")
	}

	return nil
}

// applyNativeConfig 下发原生配置到设备
func (r *NativeDeviceConfigReconciler) applyNativeConfig(
	ctx context.Context,
	config *bizv1.NativeDeviceConfig,
) error {
	// TODO: 实际 NETCONF/CLI 下发逻辑
	// 根据不同格式选择不同的下发方式：
	// - CLI: 通过 SSH 或 Console 执行命令
	// - YANG: 通过 NETCONF edit-config
	// - XML: 直接作为 NETCONF payload
	// - JSON: 转换为 XML 后下发

	// 模拟执行延时
	time.Sleep(100 * time.Millisecond)

	return nil
}

// calculateConfigHash 计算配置内容的哈希值
func calculateConfigHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// handleReconcileError 处理调和错误
func (r *NativeDeviceConfigReconciler) handleReconcileError(
	ctx context.Context,
	nativeConfig *bizv1.NativeDeviceConfig,
	err error,
	startTime time.Time,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	durationMs := time.Since(startTime).Milliseconds()
	errorType := classifyError(err)
	maxRetries := nativeConfig.Spec.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3 // 默认 3 次
	}

	nativeConfig.Status.ErrorType = errorType
	nativeConfig.Status.RetryCount++
	nativeConfig.Status.ExecutionEndTime = metav1.Now()
	nativeConfig.Status.ExecutionDurationMs = durationMs

	var requeueAfter time.Duration
	var result ctrl.Result

	if errorType == ErrorTypePermanent {
		nativeConfig.Status.Phase = bizv1.PhaseFailed
		nativeConfig.Status.Message = fmt.Sprintf("配置下发失败(永久错误): %v", err)
		nativeConfig.Status.ExecutionStatus = bizv1.ExecutionStatusFailed
		logger.Error(err, "Permanent error applying native config")
		result = ctrl.Result{RequeueAfter: 60 * time.Minute}
	} else if nativeConfig.Status.RetryCount >= maxRetries {
		nativeConfig.Status.Phase = bizv1.PhaseFailed
		nativeConfig.Status.Message = fmt.Sprintf("配置下发失败(已达最大重试次数 %d): %v", maxRetries, err)
		nativeConfig.Status.ExecutionStatus = bizv1.ExecutionStatusFailed
		logger.Error(err, "Max retry count reached for native config")
		result = ctrl.Result{RequeueAfter: 60 * time.Minute}
	} else {
		nativeConfig.Status.Phase = bizv1.PhasePending
		nativeConfig.Status.Message = fmt.Sprintf("临时错误，将重试: %v (重试次数: %d/%d)", err, nativeConfig.Status.RetryCount, maxRetries)
		nativeConfig.Status.ExecutionStatus = bizv1.ExecutionStatusPending
		requeueAfter = calculateBackoff(nativeConfig.Status.RetryCount)
		logger.Info("Temporary error applying native config, will retry with backoff",
			"error", err, "retryCount", nativeConfig.Status.RetryCount, "maxRetries", maxRetries, "requeueAfter", requeueAfter)
		result = ctrl.Result{RequeueAfter: requeueAfter}
	}

	if updateErr := r.Status().Update(ctx, nativeConfig); updateErr != nil {
		logger.Error(updateErr, "Failed to update error status")
	}

	return result, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *NativeDeviceConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&bizv1.NativeDeviceConfig{}).
		Complete(r)
}
