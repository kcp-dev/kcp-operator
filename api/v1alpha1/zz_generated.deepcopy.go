//go:build !ignore_autogenerated

/*
Copyright 2024 The KCP Authors.

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
	"k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AuthSpec) DeepCopyInto(out *AuthSpec) {
	*out = *in
	if in.OIDC != nil {
		in, out := &in.OIDC, &out.OIDC
		*out = new(OIDCConfiguration)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AuthSpec.
func (in *AuthSpec) DeepCopy() *AuthSpec {
	if in == nil {
		return nil
	}
	out := new(AuthSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CacheConfig) DeepCopyInto(out *CacheConfig) {
	*out = *in
	if in.Embedded != nil {
		in, out := &in.Embedded, &out.Embedded
		*out = new(EmbeddedCacheConfiguration)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CacheConfig.
func (in *CacheConfig) DeepCopy() *CacheConfig {
	if in == nil {
		return nil
	}
	out := new(CacheConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CacheServer) DeepCopyInto(out *CacheServer) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CacheServer.
func (in *CacheServer) DeepCopy() *CacheServer {
	if in == nil {
		return nil
	}
	out := new(CacheServer)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CacheServer) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CacheServerList) DeepCopyInto(out *CacheServerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]CacheServer, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CacheServerList.
func (in *CacheServerList) DeepCopy() *CacheServerList {
	if in == nil {
		return nil
	}
	out := new(CacheServerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CacheServerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CacheServerSpec) DeepCopyInto(out *CacheServerSpec) {
	*out = *in
	in.Etcd.DeepCopyInto(&out.Etcd)
	if in.Image != nil {
		in, out := &in.Image, &out.Image
		*out = new(ImageSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CacheServerSpec.
func (in *CacheServerSpec) DeepCopy() *CacheServerSpec {
	if in == nil {
		return nil
	}
	out := new(CacheServerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CacheServerStatus) DeepCopyInto(out *CacheServerStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CacheServerStatus.
func (in *CacheServerStatus) DeepCopy() *CacheServerStatus {
	if in == nil {
		return nil
	}
	out := new(CacheServerStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Certificates) DeepCopyInto(out *Certificates) {
	*out = *in
	if in.IssuerRef != nil {
		in, out := &in.IssuerRef, &out.IssuerRef
		*out = new(ObjectReference)
		**out = **in
	}
	if in.CASecretRef != nil {
		in, out := &in.CASecretRef, &out.CASecretRef
		*out = new(v1.LocalObjectReference)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Certificates.
func (in *Certificates) DeepCopy() *Certificates {
	if in == nil {
		return nil
	}
	out := new(Certificates)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CommonShardSpec) DeepCopyInto(out *CommonShardSpec) {
	*out = *in
	in.Etcd.DeepCopyInto(&out.Etcd)
	if in.Image != nil {
		in, out := &in.Image, &out.Image
		*out = new(ImageSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CommonShardSpec.
func (in *CommonShardSpec) DeepCopy() *CommonShardSpec {
	if in == nil {
		return nil
	}
	out := new(CommonShardSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EmbeddedCacheConfiguration) DeepCopyInto(out *EmbeddedCacheConfiguration) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EmbeddedCacheConfiguration.
func (in *EmbeddedCacheConfiguration) DeepCopy() *EmbeddedCacheConfiguration {
	if in == nil {
		return nil
	}
	out := new(EmbeddedCacheConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EtcdCertificate) DeepCopyInto(out *EtcdCertificate) {
	*out = *in
	out.SecretRef = in.SecretRef
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EtcdCertificate.
func (in *EtcdCertificate) DeepCopy() *EtcdCertificate {
	if in == nil {
		return nil
	}
	out := new(EtcdCertificate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EtcdConfig) DeepCopyInto(out *EtcdConfig) {
	*out = *in
	if in.Endpoints != nil {
		in, out := &in.Endpoints, &out.Endpoints
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	out.ClientCert = in.ClientCert
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EtcdConfig.
func (in *EtcdConfig) DeepCopy() *EtcdConfig {
	if in == nil {
		return nil
	}
	out := new(EtcdConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FrontProxy) DeepCopyInto(out *FrontProxy) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FrontProxy.
func (in *FrontProxy) DeepCopy() *FrontProxy {
	if in == nil {
		return nil
	}
	out := new(FrontProxy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *FrontProxy) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FrontProxyList) DeepCopyInto(out *FrontProxyList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]FrontProxy, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FrontProxyList.
func (in *FrontProxyList) DeepCopy() *FrontProxyList {
	if in == nil {
		return nil
	}
	out := new(FrontProxyList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *FrontProxyList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FrontProxySpec) DeepCopyInto(out *FrontProxySpec) {
	*out = *in
	in.RootShard.DeepCopyInto(&out.RootShard)
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
	if in.Auth != nil {
		in, out := &in.Auth, &out.Auth
		*out = new(AuthSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FrontProxySpec.
func (in *FrontProxySpec) DeepCopy() *FrontProxySpec {
	if in == nil {
		return nil
	}
	out := new(FrontProxySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FrontProxyStatus) DeepCopyInto(out *FrontProxyStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FrontProxyStatus.
func (in *FrontProxyStatus) DeepCopy() *FrontProxyStatus {
	if in == nil {
		return nil
	}
	out := new(FrontProxyStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageSpec) DeepCopyInto(out *ImageSpec) {
	*out = *in
	if in.ImagePullSecrets != nil {
		in, out := &in.ImagePullSecrets, &out.ImagePullSecrets
		*out = make([]v1.LocalObjectReference, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageSpec.
func (in *ImageSpec) DeepCopy() *ImageSpec {
	if in == nil {
		return nil
	}
	out := new(ImageSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Kubeconfig) DeepCopyInto(out *Kubeconfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Kubeconfig.
func (in *Kubeconfig) DeepCopy() *Kubeconfig {
	if in == nil {
		return nil
	}
	out := new(Kubeconfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Kubeconfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubeconfigList) DeepCopyInto(out *KubeconfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Kubeconfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubeconfigList.
func (in *KubeconfigList) DeepCopy() *KubeconfigList {
	if in == nil {
		return nil
	}
	out := new(KubeconfigList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KubeconfigList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubeconfigSpec) DeepCopyInto(out *KubeconfigSpec) {
	*out = *in
	in.Target.DeepCopyInto(&out.Target)
	if in.Groups != nil {
		in, out := &in.Groups, &out.Groups
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	out.Validity = in.Validity
	out.SecretRef = in.SecretRef
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubeconfigSpec.
func (in *KubeconfigSpec) DeepCopy() *KubeconfigSpec {
	if in == nil {
		return nil
	}
	out := new(KubeconfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubeconfigStatus) DeepCopyInto(out *KubeconfigStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubeconfigStatus.
func (in *KubeconfigStatus) DeepCopy() *KubeconfigStatus {
	if in == nil {
		return nil
	}
	out := new(KubeconfigStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubeconfigTarget) DeepCopyInto(out *KubeconfigTarget) {
	*out = *in
	if in.RootShardRef != nil {
		in, out := &in.RootShardRef, &out.RootShardRef
		*out = new(v1.LocalObjectReference)
		**out = **in
	}
	if in.ShardRef != nil {
		in, out := &in.ShardRef, &out.ShardRef
		*out = new(v1.LocalObjectReference)
		**out = **in
	}
	if in.FrontProxyRef != nil {
		in, out := &in.FrontProxyRef, &out.FrontProxyRef
		*out = new(v1.LocalObjectReference)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubeconfigTarget.
func (in *KubeconfigTarget) DeepCopy() *KubeconfigTarget {
	if in == nil {
		return nil
	}
	out := new(KubeconfigTarget)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OIDCConfiguration) DeepCopyInto(out *OIDCConfiguration) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OIDCConfiguration.
func (in *OIDCConfiguration) DeepCopy() *OIDCConfiguration {
	if in == nil {
		return nil
	}
	out := new(OIDCConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ObjectReference) DeepCopyInto(out *ObjectReference) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ObjectReference.
func (in *ObjectReference) DeepCopy() *ObjectReference {
	if in == nil {
		return nil
	}
	out := new(ObjectReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RootShard) DeepCopyInto(out *RootShard) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RootShard.
func (in *RootShard) DeepCopy() *RootShard {
	if in == nil {
		return nil
	}
	out := new(RootShard)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RootShard) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RootShardConfig) DeepCopyInto(out *RootShardConfig) {
	*out = *in
	if in.Reference != nil {
		in, out := &in.Reference, &out.Reference
		*out = new(v1.ObjectReference)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RootShardConfig.
func (in *RootShardConfig) DeepCopy() *RootShardConfig {
	if in == nil {
		return nil
	}
	out := new(RootShardConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RootShardList) DeepCopyInto(out *RootShardList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]RootShard, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RootShardList.
func (in *RootShardList) DeepCopy() *RootShardList {
	if in == nil {
		return nil
	}
	out := new(RootShardList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RootShardList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RootShardSpec) DeepCopyInto(out *RootShardSpec) {
	*out = *in
	in.CommonShardSpec.DeepCopyInto(&out.CommonShardSpec)
	in.Cache.DeepCopyInto(&out.Cache)
	in.Certificates.DeepCopyInto(&out.Certificates)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RootShardSpec.
func (in *RootShardSpec) DeepCopy() *RootShardSpec {
	if in == nil {
		return nil
	}
	out := new(RootShardSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RootShardStatus) DeepCopyInto(out *RootShardStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RootShardStatus.
func (in *RootShardStatus) DeepCopy() *RootShardStatus {
	if in == nil {
		return nil
	}
	out := new(RootShardStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Shard) DeepCopyInto(out *Shard) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Shard.
func (in *Shard) DeepCopy() *Shard {
	if in == nil {
		return nil
	}
	out := new(Shard)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Shard) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ShardList) DeepCopyInto(out *ShardList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Shard, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ShardList.
func (in *ShardList) DeepCopy() *ShardList {
	if in == nil {
		return nil
	}
	out := new(ShardList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ShardList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ShardSpec) DeepCopyInto(out *ShardSpec) {
	*out = *in
	in.CommonShardSpec.DeepCopyInto(&out.CommonShardSpec)
	in.RootShard.DeepCopyInto(&out.RootShard)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ShardSpec.
func (in *ShardSpec) DeepCopy() *ShardSpec {
	if in == nil {
		return nil
	}
	out := new(ShardSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ShardStatus) DeepCopyInto(out *ShardStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ShardStatus.
func (in *ShardStatus) DeepCopy() *ShardStatus {
	if in == nil {
		return nil
	}
	out := new(ShardStatus)
	in.DeepCopyInto(out)
	return out
}
