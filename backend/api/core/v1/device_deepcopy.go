package v1

// 手写 deepcopy：仓库无 controller-gen regen 管线（zz_generated.deepcopy.go 为
// 历史一次性生成物），Device 字段均为标量 + 一个指针 struct，手写维护成本可控。

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto copies the receiver into out.
func (in *Device) DeepCopyInto(out *Device) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy creates a new Device deep copy.
func (in *Device) DeepCopy() *Device {
	if in == nil {
		return nil
	}
	out := new(Device)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject implements runtime.Object.
func (in *Device) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto copies the receiver into out.
func (in *DeviceSpec) DeepCopyInto(out *DeviceSpec) {
	*out = *in
	if in.CredentialsSecretRef != nil {
		out.CredentialsSecretRef = &LocalSecretRef{Name: in.CredentialsSecretRef.Name}
	}
}

// DeepCopy creates a new DeviceSpec deep copy.
func (in *DeviceSpec) DeepCopy() *DeviceSpec {
	if in == nil {
		return nil
	}
	out := new(DeviceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver into out.
func (in *LocalSecretRef) DeepCopyInto(out *LocalSecretRef) {
	*out = *in
}

// DeepCopy creates a new LocalSecretRef deep copy.
func (in *LocalSecretRef) DeepCopy() *LocalSecretRef {
	if in == nil {
		return nil
	}
	out := new(LocalSecretRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver into out.
func (in *DeviceList) DeepCopyInto(out *DeviceList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]Device, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

// DeepCopy creates a new DeviceList deep copy.
func (in *DeviceList) DeepCopy() *DeviceList {
	if in == nil {
		return nil
	}
	out := new(DeviceList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject implements runtime.Object.
func (in *DeviceList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
