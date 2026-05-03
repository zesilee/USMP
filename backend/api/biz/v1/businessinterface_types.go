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
	// +custom:label="设备 ID"
	// +custom:group="基本信息"
	DeviceID string `json:"deviceID"`

	// IfName is the name of the interface
	// +kubebuilder:validation:Required
	// +custom:label="接口名称"
	// +custom:placeholder="例如: GigabitEthernet0/0/1"
	// +custom:group="基本信息"
	IfName string `json:"ifName"`

	// Description is a human-readable description for the interface
	// +custom:label="描述"
	// +custom:group="基本信息"
	Description string `json:"description,omitempty"`

	// AdminStatus indicates the intended administrative state
	// +kubebuilder:validation:Enum=up;down
	// +kubebuilder:default=up
	// +custom:label="管理状态"
	// +custom:group="基本设置"
	AdminStatus InterfaceAdminStatus `json:"adminStatus,omitempty"`

	// Mode is the interface operating mode
	// +kubebuilder:validation:Enum=access;trunk;hybrid
	// +kubebuilder:default=access
	// +custom:label="接口模式"
	// +custom:group="VLAN 配置"
	Mode InterfaceMode `json:"mode,omitempty"`

	// AccessVlan is the VLAN ID for access mode
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=4094
	// +kubebuilder:default=0
	// +custom:label="Access VLAN"
	// +custom:group="VLAN 配置"
	AccessVlan uint16 `json:"accessVlan,omitempty"`

	// TrunkVlans is the list of VLAN IDs allowed on trunk mode
	// +custom:label="Trunk VLAN 列表"
	// +custom:group="VLAN 配置"
	TrunkVlans []uint16 `json:"trunkVlans,omitempty"`

	// MTU is the maximum transmission unit for the interface
	// +kubebuilder:validation:Minimum=64
	// +kubebuilder:validation:Maximum=9216
	// +kubebuilder:default=1500
	// +custom:label="MTU"
	// +custom:group="高级设置"
	MTU uint32 `json:"mtu,omitempty"`

	// EnableLldp enables LLDP protocol on the interface
	// +kubebuilder:default=true
	// +custom:label="启用 LLDP"
	// +custom:group="高级设置"
	EnableLldp bool `json:"enableLldp,omitempty"`

	// EnableStormControl enables broadcast/multicast storm control
	// +kubebuilder:default=false
	// +custom:label="启用风暴控制"
	// +custom:group="高级设置"
	EnableStormControl bool `json:"enableStormControl,omitempty"`
}

// BusinessInterfaceStatus defines the observed state of BusinessInterface.
type BusinessInterfaceStatus struct {
	// Phase indicates the current reconciliation phase
	// +kubebuilder:validation:Enum=Pending;Updating;Ready;Failed
	// +custom:label="同步状态"
	Phase ConfigPhase `json:"phase,omitempty"`

	// LastSyncTime is the timestamp of the last successful sync
	// +custom:label="最后同步时间"
	LastSyncTime metav1.Time `json:"lastSyncTime,omitempty"`

	// OperStatus is the current operational status
	// +kubebuilder:validation:Enum=up;down
	OperStatus InterfaceAdminStatus `json:"operStatus,omitempty"`

	// PhysicalStatus indicates whether the interface is physically present
	PhysicalStatus string `json:"physicalStatus,omitempty"`

	// InSpeed is the inbound bandwidth in Mbps
	InSpeed uint64 `json:"inSpeed,omitempty"`

	// OutSpeed is the outbound bandwidth in Mbps
	OutSpeed uint64 `json:"outSpeed,omitempty"`

	// Error contains error message if synchronization failed
	// +custom:label="错误信息"
	Error string `json:"error,omitempty"`

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
