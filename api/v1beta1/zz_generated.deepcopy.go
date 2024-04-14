//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright 2024 Sourav Patnaik.

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

package v1beta1

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExcludeList) DeepCopyInto(out *ExcludeList) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExcludeList.
func (in *ExcludeList) DeepCopy() *ExcludeList {
	if in == nil {
		return nil
	}
	out := new(ExcludeList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecretStatus) DeepCopyInto(out *SecretStatus) {
	*out = *in
	in.Created.DeepCopyInto(&out.Created)
	in.LastModified.DeepCopyInto(&out.LastModified)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecretStatus.
func (in *SecretStatus) DeepCopy() *SecretStatus {
	if in == nil {
		return nil
	}
	out := new(SecretStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StaleSecretToWatch) DeepCopyInto(out *StaleSecretToWatch) {
	*out = *in
	if in.ExcludeList != nil {
		in, out := &in.ExcludeList, &out.ExcludeList
		*out = make([]ExcludeList, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StaleSecretToWatch.
func (in *StaleSecretToWatch) DeepCopy() *StaleSecretToWatch {
	if in == nil {
		return nil
	}
	out := new(StaleSecretToWatch)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StaleSecretWatch) DeepCopyInto(out *StaleSecretWatch) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StaleSecretWatch.
func (in *StaleSecretWatch) DeepCopy() *StaleSecretWatch {
	if in == nil {
		return nil
	}
	out := new(StaleSecretWatch)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *StaleSecretWatch) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StaleSecretWatchCondition) DeepCopyInto(out *StaleSecretWatchCondition) {
	*out = *in
	in.LastUpdateTime.DeepCopyInto(&out.LastUpdateTime)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StaleSecretWatchCondition.
func (in *StaleSecretWatchCondition) DeepCopy() *StaleSecretWatchCondition {
	if in == nil {
		return nil
	}
	out := new(StaleSecretWatchCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StaleSecretWatchList) DeepCopyInto(out *StaleSecretWatchList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]StaleSecretWatch, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StaleSecretWatchList.
func (in *StaleSecretWatchList) DeepCopy() *StaleSecretWatchList {
	if in == nil {
		return nil
	}
	out := new(StaleSecretWatchList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *StaleSecretWatchList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StaleSecretWatchSpec) DeepCopyInto(out *StaleSecretWatchSpec) {
	*out = *in
	in.StaleSecretToWatch.DeepCopyInto(&out.StaleSecretToWatch)
	if in.RefreshInterval != nil {
		in, out := &in.RefreshInterval, &out.RefreshInterval
		*out = new(v1.Duration)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StaleSecretWatchSpec.
func (in *StaleSecretWatchSpec) DeepCopy() *StaleSecretWatchSpec {
	if in == nil {
		return nil
	}
	out := new(StaleSecretWatchSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StaleSecretWatchStatus) DeepCopyInto(out *StaleSecretWatchStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.SecretStatus != nil {
		in, out := &in.SecretStatus, &out.SecretStatus
		*out = make([]SecretStatus, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StaleSecretWatchStatus.
func (in *StaleSecretWatchStatus) DeepCopy() *StaleSecretWatchStatus {
	if in == nil {
		return nil
	}
	out := new(StaleSecretWatchStatus)
	in.DeepCopyInto(out)
	return out
}
