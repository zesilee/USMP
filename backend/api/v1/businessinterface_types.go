package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// InterfaceMode 接口模式
type InterfaceMode string

const (
	InterfaceModeAccess  InterfaceMode = "Access"  // Access 模式
	InterfaceModeTrunk   InterfaceMode = "Trunk"   // Trunk 模式
	InterfaceModeHybrid  InterfaceMode = "Hybrid"  // Hybrid 模式（华为）
	InterfaceModeL3      InterfaceMode = "L3"      // 三层接口
	InterfaceModeL2      InterfaceMode = "L2"      // 二层接口
)

// InterfaceAdminStatus 接口管理状态
type InterfaceAdminStatus string

const (
	InterfaceAdminStatusUp   InterfaceAdminStatus = "Up"
	InterfaceAdminStatusDown InterfaceAdminStatus = "Down"
)

// VlanConfig VLAN 配置
type VlanConfig struct {
	// VLAN ID
	VlanID uint16 `json:"vlanID"`
	// 是否是 Native VLAN (仅 Trunk 模式有效)
	IsNative bool `json:"isNative,omitempty"`
}

// BusinessInterfaceSpec 定义 BusinessInterface 的期望状态
type BusinessInterfaceSpec struct {
	// 所属交换机设备 ID
	DeviceID string `json:"deviceID"`

	// 接口名称（如 GigabitEthernet0/0/1）
	InterfaceName string `json:"interfaceName"`

	// 接口描述
	Description string `json:"description,omitempty"`

	// 接口模式
	// +kubebuilder:validation:Enum=Access;Trunk;Hybrid;L3;L2
	Mode InterfaceMode `json:"mode,omitempty"`

	// 管理状态
	// +kubebuilder:validation:Enum=Up;Down
	AdminStatus InterfaceAdminStatus `json:"adminStatus,omitempty"`

	// Access VLAN（仅 Access 模式有效）
	AccessVlan uint16 `json:"accessVlan,omitempty"`

	// Trunk 允许通过的 VLAN 列表（仅 Trunk/Hybrid 模式有效）
	TrunkAllowedVlans []VlanConfig `json:"trunkAllowedVlans,omitempty"`

	// Native VLAN ID（仅 Trunk/Hybrid 模式有效，等价于 TrunkAllowedVlans 中 IsNative=true 的条目）
	NativeVlan uint16 `json:"nativeVlan,omitempty"`

	// 三层接口 IP 地址（仅 L3 模式有效）
	IpAddress string `json:"ipAddress,omitempty"`

	// 子网掩码（仅 L3 模式有效）
	Netmask string `json:"netmask,omitempty"`

	// MTU 值
	MTU uint32 `json:"mtu,omitempty"`

	// 速率配置（Mbps），0 表示自动协商
	Speed uint32 `json:"speed,omitempty"`

	// 双工模式：auto/full/half
	Duplex string `json:"duplex,omitempty"`

	// 是否启用 LLDP
	LldpEnabled *bool `json:"lldpEnabled,omitempty"`

	// 是否启用风暴控制
	StormControlEnabled *bool `json:"stormControlEnabled,omitempty"`
}

// InterfaceOperStatus 接口运行状态
type InterfaceOperStatus string

const (
	InterfaceOperStatusUp        InterfaceOperStatus = "Up"
	InterfaceOperStatusDown      InterfaceOperStatus = "Down"
	InterfaceOperStatusTesting   InterfaceOperStatus = "Testing"
	InterfaceOperStatusUnknown   InterfaceOperStatus = "Unknown"
	InterfaceOperStatusDormant   InterfaceOperStatus = "Dormant"
	InterfaceOperStatusNotPresent InterfaceOperStatus = "NotPresent"
)

// InterfaceCounters 接口统计信息
type InterfaceCounters struct {
	// 入方向字节数
	InOctets uint64 `json:"inOctets,omitempty"`
	// 出方向字节数
	OutOctets uint64 `json:"outOctets,omitempty"`
	// 入方向数据包数
	InPackets uint64 `json:"inPackets,omitempty"`
	// 出方向数据包数
	OutPackets uint64 `json:"outPackets,omitempty"`
	// 入方向错误数
	InErrors uint32 `json:"inErrors,omitempty"`
	// 出方向错误数
	OutErrors uint32 `json:"outErrors,omitempty"`
	// 入方向丢弃数
	InDiscards uint32 `json:"inDiscards,omitempty"`
	// 出方向丢弃数
	OutDiscards uint32 `json:"outDiscards,omitempty"`
}

// BusinessInterfaceStatus 定义 BusinessInterface 的实际状态
type BusinessInterfaceStatus struct {
	// 同步阶段
	// +kubebuilder:validation:Enum=Pending;Syncing;Synced;Failed
	Phase SyncPhase `json:"phase,omitempty"`

	// 最后同步时间
	LastSyncTime metav1.Time `json:"lastSyncTime,omitempty"`

	// 同步消息/错误信息
	Message string `json:"message,omitempty"`

	// 重试次数
	RetryCount int `json:"retryCount,omitempty"`

	// 错误类型：Temporary/Permanent
	ErrorType string `json:"errorType,omitempty"`

	// 运行状态
	OperStatus InterfaceOperStatus `json:"operStatus,omitempty"`

	// 接口类型（从设备读取）
	InterfaceType string `json:"interfaceType,omitempty"`

	// 物理地址/MAC
	PhysAddress string `json:"physAddress,omitempty"`

	// 实际速率（Mbps）
	ActualSpeed uint32 `json:"actualSpeed,omitempty"`

	// 实际 MTU
	ActualMTU uint32 `json:"actualMTU,omitempty"`

	// 统计信息
	Counters *InterfaceCounters `json:"counters,omitempty"`

	// 设备上实际配置的 VLAN
	ActualAccessVlan uint16 `json:"actualAccessVlan,omitempty"`
	ActualTrunkVlans []uint16 `json:"actualTrunkVlans,omitempty"`
	ActualNativeVlan uint16 `json:"actualNativeVlan,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Device",type="string",JSONPath=".spec.deviceID"
// +kubebuilder:printcolumn:name="Interface",type="string",JSONPath=".spec.interfaceName"
// +kubebuilder:printcolumn:name="Mode",type="string",JSONPath=".spec.mode"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.operStatus"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// BusinessInterface 是 businessinterfaces API 的 Schema
type BusinessInterface struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BusinessInterfaceSpec   `json:"spec,omitempty"`
	Status BusinessInterfaceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BusinessInterfaceList 包含 BusinessInterface 列表
type BusinessInterfaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BusinessInterface `json:"items"`
}

// DeepCopyObject implements runtime.Object
func (in *BusinessInterface) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(BusinessInterface)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver, writing into out.
func (in *BusinessInterface) DeepCopyInto(out *BusinessInterface) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopyObject implements runtime.Object
func (in *BusinessInterfaceList) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(BusinessInterfaceList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver, writing into out.
func (in *BusinessInterfaceList) DeepCopyInto(out *BusinessInterfaceList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]BusinessInterface, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopyInto copies the receiver, writing into out.
func (in *BusinessInterfaceSpec) DeepCopyInto(out *BusinessInterfaceSpec) {
	*out = *in
	if in.TrunkAllowedVlans != nil {
		in, out := &in.TrunkAllowedVlans, &out.TrunkAllowedVlans
		*out = make([]VlanConfig, len(*in))
		copy(*out, *in)
	}
	if in.LldpEnabled != nil {
		t := *in.LldpEnabled
		out.LldpEnabled = &t
	}
	if in.StormControlEnabled != nil {
		t := *in.StormControlEnabled
		out.StormControlEnabled = &t
	}
}

// DeepCopyInto copies the receiver, writing into out.
func (in *BusinessInterfaceStatus) DeepCopyInto(out *BusinessInterfaceStatus) {
	*out = *in
	in.LastSyncTime.DeepCopyInto(&out.LastSyncTime)
	if in.Counters != nil {
		in, out := &in.Counters, &out.Counters
		*out = new(InterfaceCounters)
		**out = **in
	}
	if in.ActualTrunkVlans != nil {
		in, out := &in.ActualTrunkVlans, &out.ActualTrunkVlans
		*out = make([]uint16, len(*in))
		copy(*out, *in)
	}
}

func init() {
	SchemeBuilder.Register(&BusinessInterface{}, &BusinessInterfaceList{})
}
