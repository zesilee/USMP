package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VlanAdminStatus represents the administrative status of a VLAN.
type VlanAdminStatus string

const (
	VlanAdminStatusUp   VlanAdminStatus = "up"
	VlanAdminStatusDown VlanAdminStatus = "down"
)

// MacLearningStatus represents the MAC learning state.
type MacLearningStatus string

const (
	MacLearningEnabled  MacLearningStatus = "enabled"
	MacLearningDisabled MacLearningStatus = "disabled"
)

// BusinessVlanSpec defines the desired state of BusinessVlan.
type BusinessVlanSpec struct {
	// DeviceID is the identifier of the target device (format: ip:port)
	// +kubebuilder:validation:Required
	DeviceID string `json:"deviceID"`

	// VlanID is the VLAN identifier (1-4094)
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=4094
	// +kubebuilder:validation:Required
	VlanID uint16 `json:"vlanID"`

	// Name is the VLAN name
	Name string `json:"name,omitempty"`

	// Description is a human-readable description of the VLAN
	Description string `json:"description,omitempty"`

	// AdminStatus indicates the intended VLAN state
	// +kubebuilder:validation:Enum=up;down
	// +kubebuilder:default=up
	AdminStatus VlanAdminStatus `json:"adminStatus,omitempty"`

	// BroadcastDiscard enables discarding of broadcast packets
	// +kubebuilder:default=false
	BroadcastDiscard bool `json:"broadcastDiscard,omitempty"`

	// UnknownMulticastDiscard enables discarding of unknown multicast packets
	// +kubebuilder:default=false
	UnknownMulticastDiscard bool `json:"unknownMulticastDiscard,omitempty"`

	// MacLearning controls MAC address learning on this VLAN
	// +kubebuilder:validation:Enum=enabled;disabled
	// +kubebuilder:default=enabled
	MacLearning MacLearningStatus `json:"macLearning,omitempty"`
}

// BusinessVlanStatus defines the observed state of BusinessVlan.
type BusinessVlanStatus struct {
	// ActualState is the current operational state of the VLAN
	// +kubebuilder:validation:Enum=up;down
	ActualState VlanAdminStatus `json:"actualState,omitempty"`

	// MemberPorts lists the ports currently assigned to this VLAN
	MemberPorts []string `json:"memberPorts,omitempty"`

	// MacCount is the number of MAC addresses learned on this VLAN
	MacCount int `json:"macCount,omitempty"`

	// SyncState indicates the result of the last synchronization
	// +kubebuilder:validation:Enum=Success;Failed;Syncing;Timeout
	SyncState SyncState `json:"syncState,omitempty"`

	// SyncTime is the timestamp of the last successful sync
	SyncTime metav1.Time `json:"syncTime,omitempty"`

	// ConfigVersion is the snapshot version associated with this state
	ConfigVersion string `json:"configVersion,omitempty"`

	// Conditions represents the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="VlanID",type="integer",JSONPath=".spec.vlanID"
// +kubebuilder:printcolumn:name="Name",type="string",JSONPath=".spec.name"
// +kubebuilder:printcolumn:name="DeviceID",type="string",JSONPath=".spec.deviceID"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.actualState"
// +kubebuilder:printcolumn:name="Sync",type="string",JSONPath=".status.syncState"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// BusinessVlan is the Schema for the businessvlans API.
type BusinessVlan struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BusinessVlanSpec   `json:"spec,omitempty"`
	Status BusinessVlanStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BusinessVlanList contains a list of BusinessVlan.
type BusinessVlanList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BusinessVlan `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BusinessVlan{}, &BusinessVlanList{})
}
