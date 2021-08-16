// +build !ignore_autogenerated

/*


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

package v1alpha3

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
	cluster_apiapiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	apiv1alpha3 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1alpha3"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BottlerocketConfig) DeepCopyInto(out *BottlerocketConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BottlerocketConfig.
func (in *BottlerocketConfig) DeepCopy() *BottlerocketConfig {
	if in == nil {
		return nil
	}
	out := new(BottlerocketConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CloudConfigConfig) DeepCopyInto(out *CloudConfigConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CloudConfigConfig.
func (in *CloudConfigConfig) DeepCopy() *CloudConfigConfig {
	if in == nil {
		return nil
	}
	out := new(CloudConfigConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EtcdadmConfig) DeepCopyInto(out *EtcdadmConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EtcdadmConfig.
func (in *EtcdadmConfig) DeepCopy() *EtcdadmConfig {
	if in == nil {
		return nil
	}
	out := new(EtcdadmConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EtcdadmConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EtcdadmConfigList) DeepCopyInto(out *EtcdadmConfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]EtcdadmConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EtcdadmConfigList.
func (in *EtcdadmConfigList) DeepCopy() *EtcdadmConfigList {
	if in == nil {
		return nil
	}
	out := new(EtcdadmConfigList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EtcdadmConfigList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EtcdadmConfigSpec) DeepCopyInto(out *EtcdadmConfigSpec) {
	*out = *in
	if in.Users != nil {
		in, out := &in.Users, &out.Users
		*out = make([]apiv1alpha3.User, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.EtcdadmInstallCommands != nil {
		in, out := &in.EtcdadmInstallCommands, &out.EtcdadmInstallCommands
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.PreEtcdadmCommands != nil {
		in, out := &in.PreEtcdadmCommands, &out.PreEtcdadmCommands
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.PostEtcdadmCommands != nil {
		in, out := &in.PostEtcdadmCommands, &out.PostEtcdadmCommands
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.BottlerocketConfig != nil {
		in, out := &in.BottlerocketConfig, &out.BottlerocketConfig
		*out = new(BottlerocketConfig)
		**out = **in
	}
	if in.CloudConfigConfig != nil {
		in, out := &in.CloudConfigConfig, &out.CloudConfigConfig
		*out = new(CloudConfigConfig)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EtcdadmConfigSpec.
func (in *EtcdadmConfigSpec) DeepCopy() *EtcdadmConfigSpec {
	if in == nil {
		return nil
	}
	out := new(EtcdadmConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EtcdadmConfigStatus) DeepCopyInto(out *EtcdadmConfigStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make(cluster_apiapiv1alpha3.Conditions, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.DataSecretName != nil {
		in, out := &in.DataSecretName, &out.DataSecretName
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EtcdadmConfigStatus.
func (in *EtcdadmConfigStatus) DeepCopy() *EtcdadmConfigStatus {
	if in == nil {
		return nil
	}
	out := new(EtcdadmConfigStatus)
	in.DeepCopyInto(out)
	return out
}
