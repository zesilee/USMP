package vlan

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	bizv1 "github.com/leezesi/usmp/backend/api/biz/v1"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/actor"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/reconcile"
)

const vlanFinalizer = "vlan.biz.usmp.io/finalizer"

// ActorBasedVlanReconciler reconciles a BusinessVlan object using the Actor system
type ActorBasedVlanReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	ActorManager *actor.ActorManager
}

// NewActorBasedVlanReconciler creates a new ActorBasedVlanReconciler
func NewActorBasedVlanReconciler(
	client client.Client,
	scheme *runtime.Scheme,
	actorManager *actor.ActorManager,
) *ActorBasedVlanReconciler {
	return &ActorBasedVlanReconciler{
		Client:       client,
		Scheme:       scheme,
		ActorManager: actorManager,
	}
}

//+kubebuilder:rbac:groups=biz.usmp.io,resources=businessvlans,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=biz.usmp.io,resources=businessvlans/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=biz.usmp.io,resources=businessvlans/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ActorBasedVlanReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// Fetch the BusinessVlan instance
	vlan := &bizv1.BusinessVlan{}
	if err := r.Get(ctx, req.NamespacedName, vlan); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Get device ID from spec
	deviceIP := vlan.Spec.DeviceID
	if deviceIP == "" {
		// No device specified, update status and return
		return r.updateStatus(ctx, vlan, metav1.ConditionFalse, "NoDevice", "No device reference specified")
	}

	// Get or create module actor for this device
	deviceActor := r.ActorManager.GetDeviceActor(deviceIP)

	// Ensure actor is started
	if err := deviceActor.Start(); err != nil {
		return r.updateStatus(ctx, vlan, metav1.ConditionFalse, "ActorError", fmt.Sprintf("Failed to start actor: %v", err))
	}

	// Check if the VLAN is being deleted
	if vlan.DeletionTimestamp != nil {
		// Handle deletion
		if controllerutil.ContainsFinalizer(vlan, vlanFinalizer) {
			// Run finalization logic
			if err := r.finalizeVlan(ctx, vlan, deviceActor); err != nil {
				return ctrl.Result{Requeue: true}, err
			}

			// Remove the finalizer
			controllerutil.RemoveFinalizer(vlan, vlanFinalizer)
			if err := r.Update(ctx, vlan); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(vlan, vlanFinalizer) {
		controllerutil.AddFinalizer(vlan, vlanFinalizer)
		if err := r.Update(ctx, vlan); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Convert CR Spec to payload map
	payload := map[string]interface{}{
		"Id":                     &vlan.Spec.VlanID,
		"Name":                   &vlan.Spec.Name,
		"Description":            &vlan.Spec.Description,
		"AdminStatus":            vlan.Spec.AdminStatus,
		"BroadcastDiscard":       vlan.Spec.BroadcastDiscard,
		"UnknownMulticastDiscard": vlan.Spec.UnknownMulticastDiscard,
		"MacLearning":            vlan.Spec.MacLearning,
	}

	// Store desired state in config store
	configStore := &k8sConfigStore{Client: r.Client}
	if err := configStore.Set(deviceIP, "/vlans", payload); err != nil {
		return r.updateStatus(ctx, vlan, metav1.ConditionFalse, "ConfigStoreError", fmt.Sprintf("Failed to store config: %v", err))
	}

	// Get module actor (already registered by ActorManager)
	moduleActor, exists := deviceActor.GetModuleActor("vlans")
	if !exists {
		return r.updateStatus(ctx, vlan, metav1.ConditionFalse, "ModuleNotReady", "VLAN module actor not available")
	}

	// Create actor-based reconciler
	actorReconciler := actor.NewActorReconciler(moduleActor, configStore)

	// Run reconciliation
	reconcileReq := reconcile.Request{
		DeviceID: deviceIP,
		Path:     "/vlans",
	}
	result := actorReconciler.Reconcile(ctx, reconcileReq)

	if result.Error != nil {
		return r.updateStatus(ctx, vlan, metav1.ConditionFalse, "ReconcileFailed", fmt.Sprintf("Reconciliation failed: %v", result.Error))
	}

	if result.Requeue {
		return ctrl.Result{Requeue: true, RequeueAfter: result.RequeueAfter}, nil
	}

	// Update status to indicate success
	return r.updateStatus(ctx, vlan, metav1.ConditionTrue, "Reconciled", "Configuration successfully applied")
}

// SetupWithManager sets up the controller with the Manager.
func (r *ActorBasedVlanReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&bizv1.BusinessVlan{}).
		Complete(r)
}

// updateStatus updates the VLAN CR status
func (r *ActorBasedVlanReconciler) updateStatus(
	ctx context.Context,
	vlan *bizv1.BusinessVlan,
	status metav1.ConditionStatus,
	reason, message string,
) (ctrl.Result, error) {
	// Map condition status to Phase
	switch status {
	case metav1.ConditionTrue:
		vlan.Status.Phase = bizv1.PhaseReady
	case metav1.ConditionFalse:
		vlan.Status.Phase = bizv1.PhaseFailed
	case metav1.ConditionUnknown:
		vlan.Status.Phase = bizv1.PhaseUpdating
	}

	// Update conditions
	vlan.Status.Conditions = []metav1.Condition{
		{
			Type:               "Reconciled",
			Status:             status,
			LastTransitionTime: metav1.Now(),
			Reason:             reason,
			Message:            message,
		},
	}

	if err := r.Status().Update(ctx, vlan); err != nil {
		return ctrl.Result{}, err
	}

	if status == metav1.ConditionFalse {
		return ctrl.Result{Requeue: true}, fmt.Errorf("%s", message)
	}
	return ctrl.Result{}, nil
}

// finalizeVlan handles cleanup when a VLAN is deleted
func (r *ActorBasedVlanReconciler) finalizeVlan(
	ctx context.Context,
	vlan *bizv1.BusinessVlan,
	deviceActor *actor.DeviceActor,
) error {
	// Send delete command to actor
	moduleActor, exists := deviceActor.GetModuleActor("vlans")
	if !exists {
		return nil // Already cleaned up
	}

	// Send translate with delete operation
	cmd := &actor.TranslateCmd{
		BaseMessage: actor.NewBaseMessageWithContext(fmt.Sprintf("delete-%d", vlan.Spec.VlanID), actor.MsgTranslate, ctx),
		Path:        fmt.Sprintf("/vlans/vlan[%d]", vlan.Spec.VlanID),
		Payload:     map[string]interface{}{},
		Operation:   actor.OperationDelete,
	}

	promise, err := moduleActor.Send(cmd)
	if err != nil {
		return err
	}

	result := <-promise
	if !result.Success {
		return result.Error
	}

	// Apply the delete
	applyCmd := &actor.ApplyCmd{
		BaseMessage: actor.NewBaseMessageWithContext(fmt.Sprintf("apply-delete-%d", vlan.Spec.VlanID), actor.MsgApply, ctx),
	}

	promise, err = moduleActor.Send(applyCmd)
	if err != nil {
		return err
	}

	result = <-promise
	if !result.Success {
		return result.Error
	}

	return nil
}

// k8sConfigStore implements reconcile.ConfigStore using Kubernetes CRs as storage
type k8sConfigStore struct {
	client.Client
}

func (s *k8sConfigStore) Get(deviceID, path string) (interface{}, error) {
	// In a real implementation, this would fetch from a ConfigMap or CR specifically for config storage
	// For now, return nil to indicate no cached config
	return nil, fmt.Errorf("config store not fully implemented")
}

func (s *k8sConfigStore) Set(deviceID, path string, value interface{}) error {
	// In a real implementation, this would store in a ConfigMap or CR
	return nil
}

func (s *k8sConfigStore) Delete(deviceID, path string) error {
	return nil
}

func (s *k8sConfigStore) List(deviceID string) ([]string, error) {
	return nil, nil
}

func (s *k8sConfigStore) ListDevices() ([]string, error) {
	return nil, nil
}
