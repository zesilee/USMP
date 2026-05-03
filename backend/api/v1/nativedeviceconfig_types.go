package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ConfigFormat 配置格式
type ConfigFormat string

const (
	ConfigFormatCLI    ConfigFormat = "CLI"    // 命令行格式
	ConfigFormatYANG   ConfigFormat = "YANG"   // YANG 模型格式
	ConfigFormatXML    ConfigFormat = "XML"    // XML 配置
	ConfigFormatJSON   ConfigFormat = "JSON"   // JSON 配置
)

// ExecutionMode 执行模式
type ExecutionMode string

const (
	ExecutionModeOnce      ExecutionMode = "Once"      // 执行一次
	ExecutionModePersistent ExecutionMode = "Persistent" // 持续同步（确保配置始终存在）
)

// NativeDeviceConfigSpec 定义原生配置的期望状态
type NativeDeviceConfigSpec struct {
	// 所属交换机设备 ID
	DeviceID string `json:"deviceID"`

	// 配置格式
	// +kubebuilder:validation:Enum=CLI;YANG;XML;JSON
	Format ConfigFormat `json:"format"`

	// 配置内容
	Content string `json:"content"`

	// 执行模式
	// +kubebuilder:validation:Enum=Once;Persistent
	ExecutionMode ExecutionMode `json:"executionMode,omitempty"`

	// 配置是否加密
	Encrypted bool `json:"encrypted,omitempty"`

	// 加密算法（如 AES-256）
	EncryptionAlgorithm string `json:"encryptionAlgorithm,omitempty"`

	// 密钥引用（Secret 名称）
	KeySecretRef string `json:"keySecretRef,omitempty"`

	// 配置描述
	Description string `json:"description,omitempty"`

	// 优先级（用于控制下发顺序，数字越小越先执行）
	Priority int `json:"priority,omitempty"`

	// 配置分组（用于批量管理）
	Group string `json:"group,omitempty"`

	// 执行超时时间（秒），默认 60 秒
	TimeoutSeconds int `json:"timeoutSeconds,omitempty"`

	// 失败重试次数，默认 3 次
	MaxRetries int `json:"maxRetries,omitempty"`

	// 是否在下发前保存配置
	SaveBeforeApply bool `json:"saveBeforeApply,omitempty"`

	// 是否在下发后保存配置
	SaveAfterApply bool `json:"saveAfterApply,omitempty"`
}

// ExecutionStatus 执行状态
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "Pending"
	ExecutionStatusRunning   ExecutionStatus = "Running"
	ExecutionStatusSucceeded ExecutionStatus = "Succeeded"
	ExecutionStatusFailed    ExecutionStatus = "Failed"
	ExecutionStatusSkipped   ExecutionStatus = "Skipped"
)

// NativeDeviceConfigStatus 定义原生配置的实际状态
type NativeDeviceConfigStatus struct {
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

	// 执行状态
	// +kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed;Skipped
	ExecutionStatus ExecutionStatus `json:"executionStatus,omitempty"`

	// 配置内容的哈希值（用于检测变化）
	ConfigHash string `json:"configHash,omitempty"`

	// 设备返回的响应
	DeviceResponse string `json:"deviceResponse,omitempty"`

	// 执行开始时间
	ExecutionStartTime metav1.Time `json:"executionStartTime,omitempty"`

	// 执行结束时间
	ExecutionEndTime metav1.Time `json:"executionEndTime,omitempty"`

	// 执行耗时（毫秒）
	ExecutionDurationMs int64 `json:"executionDurationMs,omitempty"`

	// 配置已在设备上生效
	AppliedOnDevice bool `json:"appliedOnDevice,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Device",type="string",JSONPath=".spec.deviceID"
// +kubebuilder:printcolumn:name="Format",type="string",JSONPath=".spec.format"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.executionStatus"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// NativeDeviceConfig 是原生设备配置的 Schema
// 用于直接透传 CLI/YANG/XML 配置到设备，不经过翻译引擎
type NativeDeviceConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NativeDeviceConfigSpec   `json:"spec,omitempty"`
	Status NativeDeviceConfigStatus `json:"status,omitempty"`
}

// DeepCopyObject implements runtime.Object
func (in *NativeDeviceConfig) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(NativeDeviceConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver, writing into out.
func (in *NativeDeviceConfig) DeepCopyInto(out *NativeDeviceConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// +kubebuilder:object:root=true

// NativeDeviceConfigList 包含 NativeDeviceConfig 列表
type NativeDeviceConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NativeDeviceConfig `json:"items"`
}

// DeepCopyObject implements runtime.Object
func (in *NativeDeviceConfigList) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(NativeDeviceConfigList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver, writing into out.
func (in *NativeDeviceConfigList) DeepCopyInto(out *NativeDeviceConfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]NativeDeviceConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopyInto copies the receiver, writing into out.
func (in *NativeDeviceConfigSpec) DeepCopyInto(out *NativeDeviceConfigSpec) {
	*out = *in
}

// DeepCopyInto copies the receiver, writing into out.
func (in *NativeDeviceConfigStatus) DeepCopyInto(out *NativeDeviceConfigStatus) {
	*out = *in
	in.LastSyncTime.DeepCopyInto(&out.LastSyncTime)
	in.ExecutionStartTime.DeepCopyInto(&out.ExecutionStartTime)
	in.ExecutionEndTime.DeepCopyInto(&out.ExecutionEndTime)
}

func init() {
	SchemeBuilder.Register(&NativeDeviceConfig{}, &NativeDeviceConfigList{})
}
