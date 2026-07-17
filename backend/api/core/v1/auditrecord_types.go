package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// AuditRecordSpec 承载一条操作审计记录（OA-02：每条记录一个 CR，CRD 仅当
// 持久化载体）。字段对齐 audit.Record；对账结局不存于此（查询时 live-join）。
type AuditRecordSpec struct {
	// Timestamp 是操作发生时间。
	// +kubebuilder:validation:Required
	Timestamp metav1.Time `json:"timestamp"`

	// DeviceIP 是目标设备（同 label usmp.io/device-ip，供筛选）。
	// +kubebuilder:validation:Required
	DeviceIP string `json:"deviceIP"`

	// Path 是 YANG 配置路径。
	Path string `json:"path,omitempty"`

	// Summary 是提交内容摘要（如 list keys）。
	Summary string `json:"summary,omitempty"`

	// Triggered 表示是否有 controller 接管对账。
	Triggered bool `json:"triggered,omitempty"`

	// Actor 是操作来源（无鉴权后端默认 system）。
	Actor string `json:"actor,omitempty"`

	// Forced 表示该下发经 force 覆盖了业务意图的归属硬锁（OA-01 二期）。
	Forced bool `json:"forced,omitempty"`

	// ForcedOwners 是被覆盖认领的意图 CR 名单（namespace/name）。
	ForcedOwners []string `json:"forcedOwners,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Device",type="string",JSONPath=".spec.deviceIP"
// +kubebuilder:printcolumn:name="Path",type="string",JSONPath=".spec.path"
// +kubebuilder:printcolumn:name="When",type="string",JSONPath=".spec.timestamp"

// AuditRecord 是操作审计记录 CR（global-ha-multi-instance W4，OA-01~05）。
type AuditRecord struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AuditRecordSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// AuditRecordList contains a list of AuditRecord.
type AuditRecordList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AuditRecord `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AuditRecord{}, &AuditRecordList{})
}

// 手写 deepcopy（同 device_deepcopy.go：仓库无 controller-gen regen 管线）。

// DeepCopyInto copies the receiver into out.
func (in *AuditRecord) DeepCopyInto(out *AuditRecord) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy creates a new AuditRecord deep copy.
func (in *AuditRecord) DeepCopy() *AuditRecord {
	if in == nil {
		return nil
	}
	out := new(AuditRecord)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject implements runtime.Object.
func (in *AuditRecord) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto copies the receiver into out.
func (in *AuditRecordSpec) DeepCopyInto(out *AuditRecordSpec) {
	*out = *in
	in.Timestamp.DeepCopyInto(&out.Timestamp)
	if in.ForcedOwners != nil {
		out.ForcedOwners = make([]string, len(in.ForcedOwners))
		copy(out.ForcedOwners, in.ForcedOwners)
	}
}

// DeepCopy creates a new AuditRecordSpec deep copy.
func (in *AuditRecordSpec) DeepCopy() *AuditRecordSpec {
	if in == nil {
		return nil
	}
	out := new(AuditRecordSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver into out.
func (in *AuditRecordList) DeepCopyInto(out *AuditRecordList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]AuditRecord, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

// DeepCopy creates a new AuditRecordList deep copy.
func (in *AuditRecordList) DeepCopy() *AuditRecordList {
	if in == nil {
		return nil
	}
	out := new(AuditRecordList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject implements runtime.Object.
func (in *AuditRecordList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
