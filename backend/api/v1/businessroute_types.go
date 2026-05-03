package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// RouteType 路由类型
type RouteType string

const (
	RouteTypeStatic     RouteType = "Static"     // 静态路由
	RouteTypeDefault    RouteType = "Default"    // 默认路由
	RouteTypeBlackhole RouteType = "Blackhole" // 黑洞路由
)

// NextHopType 下一跳类型
type NextHopType string

const (
	NextHopTypeIPAddress NextHopType = "IPAddress" // IP 地址下一跳
	NextHopTypeIFName NextHopType = "Interface" // 出接口下一跳
)

// BusinessRouteSpec 定义 BusinessRoute 的期望状态
type BusinessRouteSpec struct {
	// 所属交换机设备 ID
	DeviceID string `json:"deviceID"`

	// 路由类型
	// +kubebuilder:validation:Enum=Static;Default;Blackhole
	Type RouteType `json:"type,omitempty"`

	// 目标网络（CIDR 格式，如 192.168.1.0/24
	DestinationCIDR string `json:"destinationCIDR"`

	// 下一跳类型
	// +kubebuilder:validation:Enum=IPAddress;Interface
	NextHopType NextHopType `json:"nextHopType,omitempty"`

	// 下一跳 IP 地址（当 NextHopType=IPAddress 时必填）
	NextHopIP string `json:"nextHopIP,omitempty"`

	// 出接口名称（当 NextHopType=Interface 时必填，如 Vlanif10、GigabitEthernet0/0/1）
	OutInterface string `json:"outInterface,omitempty"`

	// 路由优先级（值越小优先级越高，1-255，默认 60）
	Preference uint8 `json:"preference,omitempty"`

	// 路由标签（用于路由策略匹配用）
	Tag uint32 `json:"tag,omitempty"`

	// 描述信息
	Description string `json:"description,omitempty"`

	// 是否启用 BFD 检测
	BfdEnabled *bool `json:"bfdEnabled,omitempty"`

	// BFD 会话名称
	BfdSessionName string `json:"bfdSessionName,omitempty"`

	// 是否为永久路由（接口 Down 时不删除）
	Permanent *bool `json:"permanent,omitempty"`

	// 路由是否发布到其他协议
	Advertise *bool `json:"advertise,omitempty"`
}

// RouteStatusType 路由实际状态
type RouteStatusType string

const (
	RouteStatusActive  RouteStatusType = "Active"   // 活跃
	RouteStatusInactive RouteStatusType = "Inactive" // 不活跃
	RouteStatusFailed   RouteStatusType = "Failed"   // 下发失败
)

// BusinessRouteStatus 定义 BusinessRoute 的实际状态
type BusinessRouteStatus struct {
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

	// 路由实际状态
	// +kubebuilder:validation:Enum=Active;Inactive;Failed
	RouteStatus RouteStatusType `json:"routeStatus,omitempty"`

	// 出接口状态（Up/Down）
	OutInterfaceStatus string `json:"outInterfaceStatus,omitempty"`

	// 下一跳可达性
	NextHopReachable *bool `json:"nextHopReachable,omitempty"`

	// 路由的实际优先级
	ActualPreference uint8 `json:"actualPreference,omitempty"`

	// 下一跳的 MAC 地址
	NextHopMac string `json:"nextHopMac,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Device",type="string",JSONPath=".spec.deviceID"
// +kubebuilder:printcolumn:name="Destination",type="string",JSONPath=".spec.destinationCIDR"
// +kubebuilder:printcolumn:name="NextHop",type="string",JSONPath=".spec.nextHopIP"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.routeStatus"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// BusinessRoute 是 businessroutes API 的 Schema
type BusinessRoute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BusinessRouteSpec   `json:"spec,omitempty"`
	Status BusinessRouteStatus `json:"status,omitempty"`
}

// DeepCopyObject implements runtime.Object
func (in *BusinessRoute) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(BusinessRoute)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver, writing into out.
func (in *BusinessRoute) DeepCopyInto(out *BusinessRoute) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// +kubebuilder:object:root=true

// BusinessRouteList 包含 BusinessRoute 列表
type BusinessRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BusinessRoute `json:"items"`
}

// DeepCopyObject implements runtime.Object
func (in *BusinessRouteList) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(BusinessRouteList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver, writing into out.
func (in *BusinessRouteList) DeepCopyInto(out *BusinessRouteList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]BusinessRoute, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopyInto copies the receiver, writing into out.
func (in *BusinessRouteSpec) DeepCopyInto(out *BusinessRouteSpec) {
	*out = *in
	if in.BfdEnabled != nil {
		t := *in.BfdEnabled
		out.BfdEnabled = &t
	}
	if in.Permanent != nil {
		t := *in.Permanent
		out.Permanent = &t
	}
	if in.Advertise != nil {
		t := *in.Advertise
		out.Advertise = &t
	}
}

// DeepCopyInto copies the receiver, writing into out.
func (in *BusinessRouteStatus) DeepCopyInto(out *BusinessRouteStatus) {
	*out = *in
	in.LastSyncTime.DeepCopyInto(&out.LastSyncTime)
	if in.NextHopReachable != nil {
		t := *in.NextHopReachable
		out.NextHopReachable = &t
	}
}

func init() {
	SchemeBuilder.Register(&BusinessRoute{}, &BusinessRouteList{})
}
