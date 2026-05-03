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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	bizv1 "github.com/leezesi/usmp/backend/api/v1"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/actor"
	netconfclient "github.com/leezesi/usmp/backend/pkg/yang-runtime/client"
	"github.com/leezesi/usmp/backend/internal/generated/huawei"
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
		if errors.IsNotFound(err) {
			// CR 已删除，不处理（VLAN 删除逻辑可后续通过 Finalizer 实现）
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get BusinessVlan")
		return ctrl.Result{}, err
	}

	// 2. 初始化状态
	if businessVlan.Status.Phase == "" {
		businessVlan.Status.Phase = bizv1.PhasePending
		if err := r.Status().Update(ctx, businessVlan); err != nil {
			logger.Error(err, "Failed to update initial status")
			return ctrl.Result{}, err
		}
	}

	// 3. 为该设备创建 VLAN Actor
	vlanActor, err := r.createVlanActor(businessVlan.Spec.DeviceID)
	if err != nil {
		logger.Error(err, "Failed to create VLAN Actor")
		businessVlan.Status.Phase = bizv1.PhaseFailed
		businessVlan.Status.Message = fmt.Sprintf("Actor 初始化失败: %v", err)
		r.Status().Update(ctx, businessVlan)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}
	defer vlanActor.Stop()

	// 4. 将业务配置翻译为华为 YANG 格式
	err = r.translateBusinessVlanToHuawei(ctx, vlanActor, businessVlan)
	if err != nil {
		logger.Error(err, "Failed to translate business config")
		businessVlan.Status.Phase = bizv1.PhaseFailed
		businessVlan.Status.Message = fmt.Sprintf("配置翻译失败: %v", err)
		r.Status().Update(ctx, businessVlan)
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// 5. 更新状态为同步中
	businessVlan.Status.Phase = bizv1.PhaseSyncing
	if err := r.Status().Update(ctx, businessVlan); err != nil {
		logger.Error(err, "Failed to update syncing status")
		return ctrl.Result{}, err
	}

	// 6. Prepare 阶段 - 校验配置并写入 Candidate
	prepareCmd := &actor.PrepareCmd{
		BaseMessage: actor.NewBaseMessageWithContext(fmt.Sprintf("prepare-%d", businessVlan.Spec.VlanID), actor.MsgPrepare, ctx),
		DryRun:      false,
	}

	promise, err := vlanActor.Send(prepareCmd)
	if err != nil {
		logger.Error(err, "Failed to send Prepare command")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	}

	result := <-promise
	if !result.Success {
		err = fmt.Errorf("prepare 失败: %v", result.Error)
		logger.Error(err, "Prepare phase failed")
		businessVlan.Status.Phase = bizv1.PhaseFailed
		businessVlan.Status.Message = err.Error()
		r.Status().Update(ctx, businessVlan)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	// 7. Commit 阶段 - 应用配置到 Running
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
		err = fmt.Errorf("commit 失败: %v", result.Error)
		logger.Error(err, "Commit phase failed")
		businessVlan.Status.Phase = bizv1.PhaseFailed
		businessVlan.Status.Message = err.Error()
		r.Status().Update(ctx, businessVlan)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	// 8. 更新最终状态
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

	// 10. 5分钟后重同步，确保配置一致性
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
func (r *BusinessVlanReconciler) translateBusinessVlanToHuawei(
	ctx context.Context,
	vlanActor *actor.ModelActor[*huawei.HuaweiVlan_Vlan_Vlans],
	businessVlan *bizv1.BusinessVlan,
) error {
	// 构建华为 VLAN Payload
	payload := map[string]interface{}{
		"Id":          businessVlan.Spec.VlanID,
		"Name":        businessVlan.Spec.Name,
		"Description": businessVlan.Spec.Description,
	}

	// 映射 VLAN 类型 (华为 VLAN 类型枚举)
	switch businessVlan.Spec.Type {
	case bizv1.VlanTypeCommon:
		payload["Type"] = huawei.HuaweiVlan_VlanType_common
	case bizv1.VlanTypeSuper:
		payload["Type"] = huawei.HuaweiVlan_VlanType_super
	case bizv1.VlanTypeSub:
		payload["Type"] = huawei.HuaweiVlan_VlanType_sub
	}

	// MAC 地址学习开关
	if businessVlan.Spec.MacLearningEnabled != nil {
		if *businessVlan.Spec.MacLearningEnabled {
			payload["MacLearning"] = huawei.HuaweiVlan_EnableStatus_enable
		} else {
			payload["MacLearning"] = huawei.HuaweiVlan_EnableStatus_disable
		}
	}

	// 统计开关
	if businessVlan.Spec.StatisticEnabled != nil {
		if *businessVlan.Spec.StatisticEnabled {
			payload["StatisticEnable"] = huawei.HuaweiVlan_EnableStatus_enable
		} else {
			payload["StatisticEnable"] = huawei.HuaweiVlan_EnableStatus_disable
		}
	}

	// 广播丢弃开关
	if businessVlan.Spec.BroadcastDiscardEnabled != nil {
		if *businessVlan.Spec.BroadcastDiscardEnabled {
			payload["BroadcastDiscard"] = huawei.HuaweiVlan_EnableStatus_enable
		} else {
			payload["BroadcastDiscard"] = huawei.HuaweiVlan_EnableStatus_disable
		}
	}

	// 端口配置（Tagged/Untagged）- 简化处理，实际华为端口配置在 Interface 模型
	// 这里只记录到 Status Diff 中，实际下发由 Interface CRD 处理

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


// SetupWithManager sets up the controller with the Manager.
func (r *BusinessVlanReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&bizv1.BusinessVlan{}).
		Complete(r)
}
