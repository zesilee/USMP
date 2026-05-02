package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// --- BusinessSwitch ---

func (in *BusinessSwitch) DeepCopyInto(out *BusinessSwitch) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

func (in *BusinessSwitch) DeepCopy() *BusinessSwitch {
	if in == nil {
		return nil
	}
	out := new(BusinessSwitch)
	in.DeepCopyInto(out)
	return out
}

func (in *BusinessSwitch) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *BusinessSwitchSpec) DeepCopyInto(out *BusinessSwitchSpec) {
	*out = *in
	if in.Tags != nil {
		in, out := &in.Tags, &out.Tags
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

func (in *BusinessSwitchStatus) DeepCopyInto(out *BusinessSwitchStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		copy(*out, *in)
	}
}

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

func (in *BusinessSwitchList) DeepCopy() *BusinessSwitchList {
	if in == nil {
		return nil
	}
	out := new(BusinessSwitchList)
	in.DeepCopyInto(out)
	return out
}

func (in *BusinessSwitchList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

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

// --- BusinessRoute ---

func (in *BusinessRoute) DeepCopyInto(out *BusinessRoute) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

func (in *BusinessRoute) DeepCopy() *BusinessRoute {
	if in == nil {
		return nil
	}
	out := new(BusinessRoute)
	in.DeepCopyInto(out)
	return out
}

func (in *BusinessRoute) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *BusinessRouteSpec) DeepCopyInto(out *BusinessRouteSpec) {
	*out = *in
}

func (in *BusinessRouteStatus) DeepCopyInto(out *BusinessRouteStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		copy(*out, *in)
	}
}

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

func (in *BusinessRouteList) DeepCopy() *BusinessRouteList {
	if in == nil {
		return nil
	}
	out := new(BusinessRouteList)
	in.DeepCopyInto(out)
	return out
}

func (in *BusinessRouteList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
