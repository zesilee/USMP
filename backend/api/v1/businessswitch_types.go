package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// VendorType 交换机厂商
type VendorType string

const (
	VendorHuawei   VendorType = "Huawei"
	VendorCisco    VendorType = "Cisco"
	VendorH3C      VendorType = "H3C"
	VendorJuniper  VendorType = "Juniper"
	VendorUnknown  VendorType = "Unknown"
)

// DeviceOnlineStatus 设备在线状态
type DeviceOnlineStatus string

const (
	DeviceOnline  DeviceOnlineStatus = "Online"
	DeviceOffline DeviceOnlineStatus = "Offline"
	DeviceUnknown DeviceOnlineStatus = "Unknown"
)

// Credentials 设备认证信息
type Credentials struct {
	// 用户名
	Username string `json:"username,omitempty"`
	// 密码（明文，建议使用 Secret）
	Password string `json:"password,omitempty"`
	// Secret 引用格式: secret-name/key
	PasswordSecretRef string `json:"passwordSecretRef,omitempty"`
}

// BusinessSwitchSpec 定义交换机的期望状态
type BusinessSwitchSpec struct {
	// 设备 IP 地址（必填）
	DeviceIP string `json:"deviceIP"`

	// 厂商
	// +kubebuilder:validation:Enum=Huawei;Cisco;H3C;Juniper;Unknown
	Vendor VendorType `json:"vendor,omitempty"`

	// 设备型号
	Model string `json:"model,omitempty"`

	// NETCONF 端口，默认 830
	Port int `json:"port,omitempty"`

	// 认证信息
	Credentials Credentials `json:"credentials,omitempty"`

	// 是否启用自动同步
	Enabled bool `json:"enabled,omitempty"`

	// 自动同步间隔（分钟），默认 5
	SyncInterval int `json:"syncInterval,omitempty"`

	// 设备描述
	Description string `json:"description,omitempty"`

	// 位置信息
	Location string `json:"location,omitempty"`

	// 负责人
	Owner string `json:"owner,omitempty"`
}

// DeviceHardwareStatus 硬件状态
type DeviceHardwareStatus struct {
	// 序列号
	SerialNumber string `json:"serialNumber,omitempty"`
	// 硬件版本
	HardwareVersion string `json:"hardwareVersion,omitempty"`
	// 软件版本
	SoftwareVersion string `json:"softwareVersion,omitempty"`
	// 运行时间
	Uptime string `json:"uptime,omitempty"`
	// CPU 使用率
	CPUUsage int `json:"cpuUsage,omitempty"`
	// 内存使用率
	MemoryUsage int `json:"memoryUsage,omitempty"`
	// 温度
	Temperature int `json:"temperature,omitempty"`
}

// BusinessSwitchStatus 定义交换机的实际状态
type BusinessSwitchStatus struct {
	// 同步阶段
	Phase SyncPhase `json:"phase,omitempty"`

	// 设备在线状态
	// +kubebuilder:validation:Enum=Online;Offline;Unknown
	OnlineStatus DeviceOnlineStatus `json:"onlineStatus,omitempty"`

	// 最后探活时间
	LastSeenTime metav1.Time `json:"lastSeenTime,omitempty"`

	// 最后同步时间
	LastSyncTime metav1.Time `json:"lastSyncTime,omitempty"`

	// 消息/错误信息
	Message string `json:"message,omitempty"`

	// 重试次数
	RetryCount int `json:"retryCount,omitempty"`

	// 错误类型
	ErrorType string `json:"errorType,omitempty"`

	// 硬件状态
	Hardware *DeviceHardwareStatus `json:"hardware,omitempty"`

	// VLAN 数量
	VlanCount int `json:"vlanCount,omitempty"`

	// 接口数量
	InterfaceCount int `json:"interfaceCount,omitempty"`

	// 在线接口数量
	OnlineInterfaceCount int `json:"onlineInterfaceCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="IP",type="string",JSONPath=".spec.deviceIP"
// +kubebuilder:printcolumn:name="Vendor",type="string",JSONPath=".spec.vendor"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.onlineStatus"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// BusinessSwitch 是交换机设备配置的 Schema
type BusinessSwitch struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BusinessSwitchSpec   `json:"spec,omitempty"`
	Status BusinessSwitchStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BusinessSwitchList 包含 BusinessSwitch 列表
type BusinessSwitchList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BusinessSwitch `json:"items"`
}

// DeepCopyObject implements runtime.Object
func (in *BusinessSwitch) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(BusinessSwitch)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver, writing into out.
func (in *BusinessSwitch) DeepCopyInto(out *BusinessSwitch) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopyObject implements runtime.Object
func (in *BusinessSwitchList) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(BusinessSwitchList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver, writing into out.
func (in *BusinessSwitchList) DeepCopyInto(out *BusinessSwitchList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]BusinessSwitch, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopyInto copies the receiver, writing into out.
func (in *Credentials) DeepCopyInto(out *Credentials) {
	*out = *in
}

// DeepCopyInto copies the receiver, writing into out.
func (in *BusinessSwitchSpec) DeepCopyInto(out *BusinessSwitchSpec) {
	*out = *in
	in.Credentials.DeepCopyInto(&out.Credentials)
}

// DeepCopyInto copies the receiver, writing into out.
func (in *DeviceHardwareStatus) DeepCopyInto(out *DeviceHardwareStatus) {
	*out = *in
}

// DeepCopyInto copies the receiver, writing into out.
func (in *BusinessSwitchStatus) DeepCopyInto(out *BusinessSwitchStatus) {
	*out = *in
	in.LastSeenTime.DeepCopyInto(&out.LastSeenTime)
	in.LastSyncTime.DeepCopyInto(&out.LastSyncTime)
	if in.Hardware != nil {
		in, out := &in.Hardware, &out.Hardware
		*out = new(DeviceHardwareStatus)
		(*in).DeepCopyInto(*out)
	}
}

func init() {
	SchemeBuilder.Register(&BusinessSwitch{}, &BusinessSwitchList{})
}
