package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigPhase represents the reconciliation phase of a configuration.
type ConfigPhase string

const (
	PhasePending  ConfigPhase = "Pending"
	PhaseUpdating ConfigPhase = "Updating"
	PhaseReady    ConfigPhase = "Ready"
	PhaseFailed   ConfigPhase = "Failed"
)

// NativeDeviceConfigSpec defines the desired state of NativeDeviceConfig.
type NativeDeviceConfigSpec struct {
	// DeviceID is the identifier of the target device (format: ip:port)
	// +kubebuilder:validation:Required
	// +custom:label="设备 ID"
	// +custom:group="基本信息"
	DeviceID string `json:"deviceID"`

	// Module is the YANG module name (e.g., huawei-ifm, huawei-vlan)
	// +kubebuilder:validation:Required
	// +custom:label="YANG 模块"
	// +custom:readonly=true
	// +custom:group="基本信息"
	Module string `json:"module"`

	// Config contains the raw YANG configuration (JSON format)
	// Schema is dynamically loaded based on Module from backend YANG library
	// +kubebuilder:validation:Required
	// +custom:label="配置内容"
	// +custom:dynamic=true
	Config map[string]interface{} `json:"config"`
}

// NativeDeviceConfigStatus defines the observed state of NativeDeviceConfig.
type NativeDeviceConfigStatus struct {
	// Phase indicates the current reconciliation phase
	// +kubebuilder:validation:Enum=Pending;Updating;Ready;Failed
	// +custom:label="同步状态"
	Phase ConfigPhase `json:"phase,omitempty"`

	// LastSyncTime is the timestamp of the last successful synchronization
	// +custom:label="最后同步时间"
	LastSyncTime metav1.Time `json:"lastSyncTime,omitempty"`

	// ActualConfigChecksum is the checksum of the actual device configuration
	ActualConfigChecksum string `json:"actualConfigChecksum,omitempty"`

	// Error contains error details if the last sync failed
	// +custom:label="错误信息"
	Error string `json:"error,omitempty"`

	// Conditions represents the latest available observations of the config's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DeviceID",type="string",JSONPath=".spec.deviceID"
// +kubebuilder:printcolumn:name="Module",type="string",JSONPath=".spec.module"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// NativeDeviceConfig is the Schema for the nativedeviceconfigs API.
type NativeDeviceConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NativeDeviceConfigSpec   `json:"spec,omitempty"`
	Status NativeDeviceConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NativeDeviceConfigList contains a list of NativeDeviceConfig.
type NativeDeviceConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NativeDeviceConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NativeDeviceConfig{}, &NativeDeviceConfigList{})
}
