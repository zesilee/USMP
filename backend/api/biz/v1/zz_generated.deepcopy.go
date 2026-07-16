package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// --- BusinessVlan ---

func (in *BusinessVlan) DeepCopyInto(out *BusinessVlan) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

func (in *BusinessVlan) DeepCopy() *BusinessVlan {
	if in == nil {
		return nil
	}
	out := new(BusinessVlan)
	in.DeepCopyInto(out)
	return out
}

func (in *BusinessVlan) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *BusinessVlanSpec) DeepCopyInto(out *BusinessVlanSpec) {
	*out = *in
}

func (in *BusinessVlanStatus) DeepCopyInto(out *BusinessVlanStatus) {
	*out = *in
	if in.MemberPorts != nil {
		in, out := &in.MemberPorts, &out.MemberPorts
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		copy(*out, *in)
	}
}

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

func (in *BusinessVlanList) DeepCopy() *BusinessVlanList {
	if in == nil {
		return nil
	}
	out := new(BusinessVlanList)
	in.DeepCopyInto(out)
	return out
}

func (in *BusinessVlanList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// --- BusinessInterface ---

func (in *BusinessInterface) DeepCopyInto(out *BusinessInterface) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

func (in *BusinessInterface) DeepCopy() *BusinessInterface {
	if in == nil {
		return nil
	}
	out := new(BusinessInterface)
	in.DeepCopyInto(out)
	return out
}

func (in *BusinessInterface) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *BusinessInterfaceSpec) DeepCopyInto(out *BusinessInterfaceSpec) {
	*out = *in
	if in.TrunkVlans != nil {
		in, out := &in.TrunkVlans, &out.TrunkVlans
		*out = make([]uint16, len(*in))
		copy(*out, *in)
	}
}

func (in *BusinessInterfaceStatus) DeepCopyInto(out *BusinessInterfaceStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		copy(*out, *in)
	}
}

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

func (in *BusinessInterfaceList) DeepCopy() *BusinessInterfaceList {
	if in == nil {
		return nil
	}
	out := new(BusinessInterfaceList)
	in.DeepCopyInto(out)
	return out
}

func (in *BusinessInterfaceList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
