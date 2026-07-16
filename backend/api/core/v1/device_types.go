package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeviceSpec 定义设备连接元信息（DS-01/DS-04）。凭据不进 CR：仅存同
// namespace Secret 的引用（etcd 不落明文，SC-02/SC-06）。
type DeviceSpec struct {
	// ManagementIP 是设备管理 IP，也是 DeviceStore 的 DeviceID（裸 IP 键）。
	// +kubebuilder:validation:Required
	ManagementIP string `json:"managementIP"`

	// Port 是协议端口（830=NETCONF, 9339/9340=gNMI）；0/缺省按消费方语义处理。
	Port int `json:"port,omitempty"`

	// Protocol 取值 netconf|gnmi|auto，缺省 auto（按端口选择）。
	// +kubebuilder:validation:Enum=netconf;gnmi;auto;""
	Protocol string `json:"protocol,omitempty"`

	// TimeoutSeconds 是建连/RPC 超时秒数；0 表示走客户端缺省。
	TimeoutSeconds int `json:"timeoutSeconds,omitempty"`

	// Vendor 是厂商标识（缺省语义 huawei，DS-01 零值缺省在消费方）。
	Vendor string `json:"vendor,omitempty"`

	// CredentialsSecretRef 指向同 namespace 的凭据 Secret（key: username/password）。
	// 缺失/悬空引用时 Get 降级空凭据（DS-04，R08 clean fail）。
	CredentialsSecretRef *LocalSecretRef `json:"credentialsSecretRef,omitempty"`
}

// LocalSecretRef 是同 namespace Secret 的名字引用。
type LocalSecretRef struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="IP",type="string",JSONPath=".spec.managementIP"
// +kubebuilder:printcolumn:name="Protocol",type="string",JSONPath=".spec.protocol"
// +kubebuilder:printcolumn:name="Vendor",type="string",JSONPath=".spec.vendor"

// Device 是设备连接元信息 CR（CRD 仅当持久化载体 + watch 事件源，SC-02）。
type Device struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec DeviceSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// DeviceList contains a list of Device.
type DeviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Device `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Device{}, &DeviceList{})
}
