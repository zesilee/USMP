package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AdminStatus represents the administrative status of a switch.
type AdminStatus string

const (
	AdminStatusOnline       AdminStatus = "online"
	AdminStatusMaintenance  AdminStatus = "maintenance"
	AdminStatusOffline      AdminStatus = "offline"
)

// BusinessSwitchSpec defines the desired state of BusinessSwitch.
type BusinessSwitchSpec struct {
	// DeviceID is the identifier of the target device (format: ip:port)
	// +kubebuilder:validation:Required
	DeviceID string `json:"deviceID"`

	// Vendor is the device manufacturer
	// +kubebuilder:validation:Enum=huawei;cisco;h3c;juniper
	// +kubebuilder:default=huawei
	Vendor string `json:"vendor,omitempty"`

	// Model is the device model identifier
	Model string `json:"model,omitempty"`

	// ManagementIP is the management IP address of the device
	// +kubebuilder:validation:Required
	ManagementIP string `json:"managementIP"`

	// AdminStatus indicates the intended operational status
	// +kubebuilder:validation:Enum=online;maintenance;offline
	// +kubebuilder:default=online
	AdminStatus AdminStatus `json:"adminStatus,omitempty"`

	// Location describes the physical location of the device
	Location string `json:"location,omitempty"`

	// Tags are user-defined tags for categorization
	Tags []string `json:"tags,omitempty"`
}

// BusinessSwitchStatus defines the observed state of BusinessSwitch.
type BusinessSwitchStatus struct {
	// Online indicates whether the device is currently reachable
	Online bool `json:"online,omitempty"`

	// PlatformVersion is the running OS version
	PlatformVersion string `json:"platformVersion,omitempty"`

	// PatchVersion is the installed patch version
	PatchVersion string `json:"patchVersion,omitempty"`

	// Uptime is the duration the device has been running
	Uptime string `json:"uptime,omitempty"`

	// SyncState indicates the result of the last synchronization
	// +kubebuilder:validation:Enum=Success;Failed;Syncing;Timeout
	SyncState SyncState `json:"syncState,omitempty"`

	// LastSyncTime is the timestamp of the last successful sync
	LastSyncTime metav1.Time `json:"lastSyncTime,omitempty"`

	// Conditions represents the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DeviceID",type="string",JSONPath=".spec.deviceID"
// +kubebuilder:printcolumn:name="Vendor",type="string",JSONPath=".spec.vendor"
// +kubebuilder:printcolumn:name="Online",type="boolean",JSONPath=".status.online"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.syncState"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// BusinessSwitch is the Schema for the businessswitches API.
type BusinessSwitch struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BusinessSwitchSpec   `json:"spec,omitempty"`
	Status BusinessSwitchStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BusinessSwitchList contains a list of BusinessSwitch.
type BusinessSwitchList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BusinessSwitch `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BusinessSwitch{}, &BusinessSwitchList{})
}
