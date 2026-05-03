package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// VlanAdminStatus 表示 VLAN 管理状态
type VlanAdminStatus string

const (
	VlanAdminStatusUp   VlanAdminStatus = "Up"
	VlanAdminStatusDown VlanAdminStatus = "Down"
)

// VlanType 表示 VLAN 类型（华为交换机支持的类型）
type VlanType string

const (
	VlanTypeCommon VlanType = "Common" // 普通 VLAN
	VlanTypeSuper  VlanType = "Super"  // Super VLAN
	VlanTypeSub    VlanType = "Sub"    // Sub VLAN
)

// PortConfig 表示端口配置
type PortConfig struct {
	// 端口名称
	Name string `json:"name"`
	// 是否为 Tagged 端口
	Tagged bool `json:"tagged"`
}

// BusinessVlanSpec 定义 BusinessVlan 的期望状态
type BusinessVlanSpec struct {
	// VLAN ID (1-4094)
	VlanID uint16 `json:"vlanID"`

	// 所属交换机设备 ID
	DeviceID string `json:"deviceID"`

	// VLAN 名称
	Name string `json:"name,omitempty"`

	// VLAN 描述
	Description string `json:"description,omitempty"`

	// VLAN 类型
	// +kubebuilder:validation:Enum=Common;Super;Sub;Principal;Separate;Group
	Type VlanType `json:"type,omitempty"`

	// 管理状态
	// +kubebuilder:validation:Enum=Up;Down
	AdminStatus VlanAdminStatus `json:"adminStatus,omitempty"`

	// Tagged 端口列表
	TaggedPorts []string `json:"taggedPorts,omitempty"`

	// Untagged 端口列表
	UntaggedPorts []string `json:"untaggedPorts,omitempty"`

	// 是否启用 MAC 地址学习
	MacLearningEnabled *bool `json:"macLearningEnabled,omitempty"`

	// 统计收集开关
	StatisticEnabled *bool `json:"statisticEnabled,omitempty"`

	// 广播丢弃开关
	BroadcastDiscardEnabled *bool `json:"broadcastDiscardEnabled,omitempty"`
}

// SyncPhase 表示同步阶段
type SyncPhase string

const (
	// PhasePending 等待同步
	PhasePending SyncPhase = "Pending"
	// PhaseSyncing 同步中
	PhaseSyncing SyncPhase = "Syncing"
	// PhaseSynced 已同步
	PhaseSynced SyncPhase = "Synced"
	// PhaseFailed 同步失败
	PhaseFailed SyncPhase = "Failed"
)

// PortStatus 表示端口实际状态
type PortStatus struct {
	// 端口名称
	Name string `json:"name"`
	// 是否为 Tagged 端口
	Tagged bool `json:"tagged"`
	// 实际状态
	Active bool `json:"active"`
}

// VlanStatus 表示设备上实际的 VLAN 状态
type VlanStatus struct {
	// VLAN ID
	VlanID uint16 `json:"vlanID"`
	// VLAN 名称
	Name string `json:"name,omitempty"`
	// 描述
	Description string `json:"description,omitempty"`
	// 类型
	Type string `json:"type,omitempty"`
	// 操作状态
	OperStatus string `json:"operStatus,omitempty"`
	// 端口状态
	Ports []PortStatus `json:"ports,omitempty"`
	// MAC 地址数量
	MacCount int `json:"macCount,omitempty"`
}

// BusinessVlanStatus 定义 BusinessVlan 的实际状态
type BusinessVlanStatus struct {
	// 同步阶段
	// +kubebuilder:validation:Enum=Pending;Syncing;Synced;Failed
	Phase SyncPhase `json:"phase,omitempty"`

	// 最后同步时间
	LastSyncTime metav1.Time `json:"lastSyncTime,omitempty"`

	// 同步消息
	Message string `json:"message,omitempty"`

	// 配置版本（用于乐观锁）
	ConfigVersion int64 `json:"configVersion,omitempty"`

	// 设备上实际的 VLAN 状态
	Actual *VlanStatus `json:"actual,omitempty"`

	// 期望状态与实际状态的差异
	Diff []string `json:"diff,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="VlanID",type="integer",JSONPath=".spec.vlanID"
// +kubebuilder:printcolumn:name="Device",type="string",JSONPath=".spec.deviceID"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="LastSync",type="date",JSONPath=".status.lastSyncTime",priority=1

// BusinessVlan 是 businessvlans API 的 Schema
type BusinessVlan struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BusinessVlanSpec   `json:"spec,omitempty"`
	Status BusinessVlanStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BusinessVlanList 包含 BusinessVlan 列表
type BusinessVlanList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BusinessVlan `json:"items"`
}

// DeepCopyObject implements runtime.Object
func (in *BusinessVlan) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(BusinessVlan)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver, writing into out.
func (in *BusinessVlan) DeepCopyInto(out *BusinessVlan) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopyObject implements runtime.Object
func (in *BusinessVlanList) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(BusinessVlanList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver, writing into out.
func (in *BusinessVlanList) DeepCopyInto(out *BusinessVlanList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]BusinessVlan, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopyInto copies the receiver, writing into out.
func (in *BusinessVlanSpec) DeepCopyInto(out *BusinessVlanSpec) {
	*out = *in
	if in.TaggedPorts != nil {
		t := make([]string, len(in.TaggedPorts))
		copy(t, in.TaggedPorts)
		out.TaggedPorts = t
	}
	if in.UntaggedPorts != nil {
		t := make([]string, len(in.UntaggedPorts))
		copy(t, in.UntaggedPorts)
		out.UntaggedPorts = t
	}
	if in.MacLearningEnabled != nil {
		t := *in.MacLearningEnabled
		out.MacLearningEnabled = &t
	}
	if in.StatisticEnabled != nil {
		t := *in.StatisticEnabled
		out.StatisticEnabled = &t
	}
	if in.BroadcastDiscardEnabled != nil {
		t := *in.BroadcastDiscardEnabled
		out.BroadcastDiscardEnabled = &t
	}
}

// DeepCopyInto copies the receiver, writing into out.
func (in *BusinessVlanStatus) DeepCopyInto(out *BusinessVlanStatus) {
	*out = *in
	in.LastSyncTime.DeepCopyInto(&out.LastSyncTime)
	if in.Actual != nil {
		in, out := &in.Actual, &out.Actual
		*out = new(VlanStatus)
		(*in).DeepCopyInto(*out)
	}
	if in.Diff != nil {
		t := make([]string, len(in.Diff))
		copy(t, in.Diff)
		out.Diff = t
	}
}

// DeepCopyInto copies the receiver, writing into out.
func (in *PortConfig) DeepCopyInto(out *PortConfig) {
	*out = *in
}

// DeepCopyInto copies the receiver, writing into out.
func (in *PortStatus) DeepCopyInto(out *PortStatus) {
	*out = *in
}

// DeepCopyInto copies the receiver, writing into out.
func (in *VlanStatus) DeepCopyInto(out *VlanStatus) {
	*out = *in
	if in.Ports != nil {
		t := make([]PortStatus, len(in.Ports))
		copy(t, in.Ports)
		out.Ports = t
	}
}

func init() {
	SchemeBuilder.Register(&BusinessVlan{}, &BusinessVlanList{})
}
