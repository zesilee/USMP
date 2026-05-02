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
	DeviceID string `json:"deviceID"`

	// Prefix is the destination IP prefix (CIDR notation)
	// +kubebuilder:validation:Required
	Prefix string `json:"prefix"`

	// NextHop is the next-hop IP address for this route
	// +kubebuilder:validation:Required
	NextHop string `json:"nextHop"`

	// Preference is the route preference value (lower is better)
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=255
	// +kubebuilder:default=60
	Preference uint8 `json:"preference,omitempty"`

	// Description is a human-readable description of the route
	Description string `json:"description,omitempty"`

	// BfdEnabled enables BFD (Bidirectional Forwarding Detection) for this route
	// +kubebuilder:default=false
	BfdEnabled bool `json:"bfdEnabled,omitempty"`
}

// BusinessRouteStatus defines the observed state of BusinessRoute.
type BusinessRouteStatus struct {
	// RouteType indicates how this route was learned
	// +kubebuilder:validation:Enum=static;dynamic;connected
	RouteType RouteType `json:"routeType,omitempty"`

	// Installed indicates whether the route is installed in the FIB
	Installed bool `json:"installed,omitempty"`

	// Active indicates whether this route is the active best route
	Active bool `json:"active,omitempty"`

	// ProtocolPreference is the actual preference value used
	ProtocolPreference uint8 `json:"protocolPreference,omitempty"`

	// SyncState indicates the result of the last synchronization
	// +kubebuilder:validation:Enum=Success;Failed;Syncing;Timeout
	SyncState SyncState `json:"syncState,omitempty"`

	// SyncTime is the timestamp of the last successful sync
	SyncTime metav1.Time `json:"syncTime,omitempty"`

	// Conditions represents the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Prefix",type="string",JSONPath=".spec.prefix"
// +kubebuilder:printcolumn:name="NextHop",type="string",JSONPath=".spec.nextHop"
// +kubebuilder:printcolumn:name="DeviceID",type="string",JSONPath=".spec.deviceID"
// +kubebuilder:printcolumn:name="Active",type="boolean",JSONPath=".status.active"
// +kubebuilder:printcolumn:name="Sync",type="string",JSONPath=".status.syncState"
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
