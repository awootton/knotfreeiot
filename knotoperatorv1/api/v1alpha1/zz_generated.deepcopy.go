//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"github.com/awootton/knotfreeiot/iot"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterState) DeepCopyInto(out *ClusterState) {
	*out = *in
	if in.GuruNames != nil {
		in, out := &in.GuruNames, &out.GuruNames
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Nodes != nil {
		in, out := &in.Nodes, &out.Nodes
		*out = make(map[string]*iot.ExecutiveStats, len(*in))
		for key, val := range *in {
			(*out)[key] = val.DeepCopy()
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterState.
func (in *ClusterState) DeepCopy() *ClusterState {
	if in == nil {
		return nil
	}
	out := new(ClusterState)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Knotoperator) DeepCopyInto(out *Knotoperator) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Knotoperator.
func (in *Knotoperator) DeepCopy() *Knotoperator {
	if in == nil {
		return nil
	}
	out := new(Knotoperator)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Knotoperator) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KnotoperatorList) DeepCopyInto(out *KnotoperatorList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Knotoperator, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KnotoperatorList.
func (in *KnotoperatorList) DeepCopy() *KnotoperatorList {
	if in == nil {
		return nil
	}
	out := new(KnotoperatorList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KnotoperatorList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KnotoperatorSpec) DeepCopyInto(out *KnotoperatorSpec) {
	*out = *in
	if in.Ce != nil {
		in, out := &in.Ce, &out.Ce
		*out = new(ClusterState)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KnotoperatorSpec.
func (in *KnotoperatorSpec) DeepCopy() *KnotoperatorSpec {
	if in == nil {
		return nil
	}
	out := new(KnotoperatorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KnotoperatorStatus) DeepCopyInto(out *KnotoperatorStatus) {
	*out = *in
	if in.Ce != nil {
		in, out := &in.Ce, &out.Ce
		*out = new(ClusterState)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KnotoperatorStatus.
func (in *KnotoperatorStatus) DeepCopy() *KnotoperatorStatus {
	if in == nil {
		return nil
	}
	out := new(KnotoperatorStatus)
	in.DeepCopyInto(out)
	return out
}