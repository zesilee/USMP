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
	// +kubebuilder:validation:Pattern=`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}(:\d+)?$`
	// +custom:label="设备 ID"
	// +custom:placeholder="例如: 192.168.1.1:830"
	// +custom:group="基本信息"
	DeviceID string `json:"deviceID"`

	// VlanID is the VLAN identifier (1-4094)
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=4094
	// +kubebuilder:validation:Required
	// +custom:label="VLAN ID"
	// +custom:group="基本信息"
	VlanID uint16 `json:"vlanID"`

	// Name is the VLAN name
	// +custom:label="VLAN 名称"
	// +custom:placeholder="例如: VLAN-100"
	// +custom:group="基本信息"
	Name string `json:"name,omitempty"`

	// Description is a human-readable description of the VLAN
	// +custom:label="描述"
	// +custom:placeholder="描述该 VLAN 用途"
	// +custom:group="基本信息"
	Description string `json:"description,omitempty"`

	// AdminStatus indicates the intended VLAN state
	// +kubebuilder:validation:Enum=up;down
	// +kubebuilder:default=up
	// +custom:label="管理状态"
	// +custom:group="高级设置"
	AdminStatus VlanAdminStatus `json:"adminStatus,omitempty"`

	// BroadcastDiscard enables discarding of broadcast packets
	// +kubebuilder:default=false
	// +custom:label="丢弃广播包"
	// +custom:group="高级设置"
	BroadcastDiscard bool `json:"broadcastDiscard,omitempty"`

	// UnknownMulticastDiscard enables discarding of unknown multicast packets
	// +kubebuilder:default=false
	// +custom:label="丢弃未知组播包"
	// +custom:group="高级设置"
	UnknownMulticastDiscard bool `json:"unknownMulticastDiscard,omitempty"`

	// MacLearning controls MAC address learning on this VLAN
	// +kubebuilder:validation:Enum=enabled;disabled
	// +kubebuilder:default=enabled
	// +custom:label="MAC 地址学习"
	// +custom:group="高级设置"
	MacLearning MacLearningStatus `json:"macLearning,omitempty"`
}

// BusinessVlanStatus defines the observed state of BusinessVlan.
type BusinessVlanStatus struct {
	// Phase indicates the current reconciliation phase
	// +kubebuilder:validation:Enum=Pending;Updating;Ready;Failed
	// +custom:label="同步状态"
	Phase ConfigPhase `json:"phase,omitempty"`

	// LastSyncTime is the timestamp of the last successful sync
	// +custom:label="最后同步时间"
	LastSyncTime metav1.Time `json:"lastSyncTime,omitempty"`

	// ActualState is the current operational state of the VLAN
	// +kubebuilder:validation:Enum=up;down
	ActualState VlanAdminStatus `json:"actualState,omitempty"`

	// MemberPorts lists the ports currently assigned to this VLAN
	MemberPorts []string `json:"memberPorts,omitempty"`

	// MacCount is the number of MAC addresses learned on this VLAN
	MacCount int `json:"macCount,omitempty"`

	// Error contains error message if synchronization failed
	// +custom:label="错误信息"
	Error string `json:"error,omitempty"`

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
