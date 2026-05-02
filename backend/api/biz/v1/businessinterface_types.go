package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InterfaceMode represents the operating mode of an interface.
type InterfaceMode string

const (
	InterfaceModeAccess  InterfaceMode = "access"
	InterfaceModeTrunk   InterfaceMode = "trunk"
	InterfaceModeHybrid  InterfaceMode = "hybrid"
)

// InterfaceAdminStatus represents the administrative status of an interface.
type InterfaceAdminStatus string

const (
	InterfaceAdminStatusUp   InterfaceAdminStatus = "up"
	InterfaceAdminStatusDown InterfaceAdminStatus = "down"
)

// BusinessInterfaceSpec defines the desired state of BusinessInterface.
type BusinessInterfaceSpec struct {
	// DeviceID is the identifier of the target device (format: ip:port)
	// +kubebuilder:validation:Required
	DeviceID string `json:"deviceID"`

	// IfName is the name of the interface
	// +kubebuilder:validation:Required
	IfName string `json:"ifName"`

	// Description is a human-readable description for the interface
	Description string `json:"description,omitempty"`

	// AdminStatus indicates the intended administrative state
	// +kubebuilder:validation:Enum=up;down
	// +kubebuilder:default=up
	AdminStatus InterfaceAdminStatus `json:"adminStatus,omitempty"`

	// Mode is the interface operating mode
	// +kubebuilder:validation:Enum=access;trunk;hybrid
	// +kubebuilder:default=access
	Mode InterfaceMode `json:"mode,omitempty"`

	// AccessVlan is the VLAN ID for access mode
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=4094
	// +kubebuilder:default=0
	AccessVlan uint16 `json:"accessVlan,omitempty"`

	// TrunkVlans is the list of VLAN IDs allowed on trunk mode
	TrunkVlans []uint16 `json:"trunkVlans,omitempty"`

	// MTU is the maximum transmission unit for the interface
	// +kubebuilder:validation:Minimum=64
	// +kubebuilder:validation:Maximum=9216
	// +kubebuilder:default=1500
	MTU uint32 `json:"mtu,omitempty"`

	// EnableLldp enables LLDP protocol on the interface
	// +kubebuilder:default=true
	EnableLldp bool `json:"enableLldp,omitempty"`

	// EnableStormControl enables broadcast/multicast storm control
	// +kubebuilder:default=false
	EnableStormControl bool `json:"enableStormControl,omitempty"`
}

// BusinessInterfaceStatus defines the observed state of BusinessInterface.
type BusinessInterfaceStatus struct {
	// OperStatus is the current operational status
	// +kubebuilder:validation:Enum=up;down
	OperStatus InterfaceAdminStatus `json:"operStatus,omitempty"`

	// PhysicalStatus indicates whether the interface is physically present
	PhysicalStatus string `json:"physicalStatus,omitempty"`

	// InSpeed is the inbound bandwidth in Mbps
	InSpeed uint64 `json:"inSpeed,omitempty"`

	// OutSpeed is the outbound bandwidth in Mbps
	OutSpeed uint64 `json:"outSpeed,omitempty"`

	// InErrors is the count of inbound errors
	InErrors uint64 `json:"inErrors,omitempty"`

	// OutErrors is the count of outbound errors
	OutErrors uint64 `json:"outErrors,omitempty"`

	// SyncState indicates the result of the last synchronization
	// +kubebuilder:validation:Enum=Success;Failed;Syncing;Timeout
	SyncState SyncState `json:"syncState,omitempty"`

	// LastFlapped is the time the interface last changed state
	LastFlapped metav1.Time `json:"lastFlapped,omitempty"`

	// Conditions represents the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Interface",type="string",JSONPath=".spec.ifName"
// +kubebuilder:printcolumn:name="Mode",type="string",JSONPath=".spec.mode"
// +kubebuilder:printcolumn:name="DeviceID",type="string",JSONPath=".spec.deviceID"
// +kubebuilder:printcolumn:name="OperState",type="string",JSONPath=".status.operStatus"
// +kubebuilder:printcolumn:name="Sync",type="string",JSONPath=".status.syncState"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// BusinessInterface is the Schema for the businessinterfaces API.
type BusinessInterface struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BusinessInterfaceSpec   `json:"spec,omitempty"`
	Status BusinessInterfaceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BusinessInterfaceList contains a list of BusinessInterface.
type BusinessInterfaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BusinessInterface `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BusinessInterface{}, &BusinessInterfaceList{})
}
