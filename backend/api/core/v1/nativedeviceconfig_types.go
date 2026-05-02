package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigType represents the type of native configuration.
type ConfigType string

const (
	ConfigTypeYangXML ConfigType = "yang-xml"
	ConfigTypeCLI     ConfigType = "cli"
	ConfigTypeJSON    ConfigType = "json"
)

// NativeDeviceConfigSpec defines the desired state of NativeDeviceConfig.
type NativeDeviceConfigSpec struct {
	// DeviceID is the identifier of the target device (format: ip:port)
	// +kubebuilder:validation:Required
	DeviceID string `json:"deviceID"`

	// ConfigType specifies the type of configuration content
	// +kubebuilder:validation:Enum=yang-xml;cli;json
	// +kubebuilder:default=yang-xml
	ConfigType ConfigType `json:"configType,omitempty"`

	// Content is the raw configuration content to be applied directly
	// +kubebuilder:validation:Required
	Content string `json:"content"`

	// Encrypt indicates whether the content should be encrypted at rest
	// +kubebuilder:default=false
	Encrypt bool `json:"encrypt,omitempty"`
}

// NativeDeviceConfigStatus defines the observed state of NativeDeviceConfig.
type NativeDeviceConfigStatus struct {
	// SyncState indicates the result of the last configuration sync
	// +kubebuilder:validation:Enum=Success;Failed;Syncing;Timeout
	SyncState SyncState `json:"syncState,omitempty"`

	// SyncTime is the timestamp of the last successful synchronization
	SyncTime metav1.Time `json:"syncTime,omitempty"`

	// Error contains error details if the last sync failed
	Error string `json:"error,omitempty"`

	// ActualConfigChecksum is the SHA256 checksum of the actual device configuration
	ActualConfigChecksum string `json:"actualConfigChecksum,omitempty"`

	// Conditions represents the latest available observations of the config's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DeviceID",type="string",JSONPath=".spec.deviceID"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.syncState"
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
