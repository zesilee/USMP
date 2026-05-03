package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RouteType represents the type of route.
type RouteType string

const (
	RouteTypeStatic   RouteType = "static"
	RouteTypeDynamic  RouteType = "dynamic"
	RouteTypeConnected RouteType = "connected"
)

// BusinessRouteSpec defines the desired state of BusinessRoute.
type BusinessRouteSpec struct {
	// DeviceID is the identifier of the target device (format: ip:port)
	// +kubebuilder:validation:Required
	// +custom:label="设备 ID"
	// +custom:group="基本信息"
	DeviceID string `json:"deviceID"`

	// Destination is the destination IP prefix (CIDR notation)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}/\d{1,2}$`
	// +custom:label="目标网络"
	// +custom:placeholder="例如: 192.168.0.0/24"
	// +custom:group="基本信息"
	Destination string `json:"destination"`

	// NextHop is the next-hop IP address for this route
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`
	// +custom:label="下一跳地址"
	// +custom:group="基本信息"
	NextHop string `json:"nextHop"`

	// Preference is the route preference value (lower is better)
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=255
	// +kubebuilder:default=60
	// +custom:label="优先级"
	// +custom:group="高级设置"
	Preference uint8 `json:"preference,omitempty"`

	// Description is a human-readable description of the route
	// +custom:label="描述"
	// +custom:group="基本信息"
	Description string `json:"description,omitempty"`

	// BfdEnabled enables BFD (Bidirectional Forwarding Detection) for this route
	// +kubebuilder:default=false
	// +custom:label="启用 BFD"
	// +custom:group="高级设置"
	BfdEnabled bool `json:"bfdEnabled,omitempty"`
}

// BusinessRouteStatus defines the observed state of BusinessRoute.
type BusinessRouteStatus struct {
	// Phase indicates the current reconciliation phase
	// +kubebuilder:validation:Enum=Pending;Updating;Ready;Failed
	// +custom:label="同步状态"
	Phase ConfigPhase `json:"phase,omitempty"`

	// LastSyncTime is the timestamp of the last successful sync
	// +custom:label="最后同步时间"
	LastSyncTime metav1.Time `json:"lastSyncTime,omitempty"`

	// RouteType indicates how this route was learned
	// +kubebuilder:validation:Enum=static;dynamic;connected
	RouteType RouteType `json:"routeType,omitempty"`

	// Active indicates whether this route is the active best route
	Active bool `json:"active,omitempty"`

	// Error contains error message if synchronization failed
	// +custom:label="错误信息"
	Error string `json:"error,omitempty"`

	// Conditions represents the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Destination",type="string",JSONPath=".spec.destination"
// +kubebuilder:printcolumn:name="NextHop",type="string",JSONPath=".spec.nextHop"
// +kubebuilder:printcolumn:name="DeviceID",type="string",JSONPath=".spec.deviceID"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// BusinessRoute is the Schema for the businessroutes API.
type BusinessRoute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BusinessRouteSpec   `json:"spec,omitempty"`
	Status BusinessRouteStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BusinessRouteList contains a list of BusinessRoute.
type BusinessRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BusinessRoute `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BusinessRoute{}, &BusinessRouteList{})
}
