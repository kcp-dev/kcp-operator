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

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package applyconfiguration

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	testing "k8s.io/client-go/testing"

	v1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
	internal "github.com/kcp-dev/kcp-operator/sdk/applyconfiguration/internal"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/applyconfiguration/operator/v1alpha1"
)

// ForKind returns an apply configuration type for the given GroupVersionKind, or nil if no
// apply configuration type exists for the given GroupVersionKind.
func ForKind(kind schema.GroupVersionKind) interface{} {
	switch kind {
	// Group=operator.kcp.io, Version=v1alpha1
	case v1alpha1.SchemeGroupVersion.WithKind("AuditSpec"):
		return &operatorv1alpha1.AuditSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("AuditWebhookSpec"):
		return &operatorv1alpha1.AuditWebhookSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("AuthorizationSpec"):
		return &operatorv1alpha1.AuthorizationSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("AuthorizationWebhookSpec"):
		return &operatorv1alpha1.AuthorizationWebhookSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("AuthSpec"):
		return &operatorv1alpha1.AuthSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("CacheConfig"):
		return &operatorv1alpha1.CacheConfigApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("CacheServer"):
		return &operatorv1alpha1.CacheServerApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("CacheServerSpec"):
		return &operatorv1alpha1.CacheServerSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("Certificates"):
		return &operatorv1alpha1.CertificatesApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("CommonShardSpec"):
		return &operatorv1alpha1.CommonShardSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("EmbeddedCacheConfiguration"):
		return &operatorv1alpha1.EmbeddedCacheConfigurationApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("EtcdConfig"):
		return &operatorv1alpha1.EtcdConfigApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("EtcdTLSConfig"):
		return &operatorv1alpha1.EtcdTLSConfigApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("ExternalConfig"):
		return &operatorv1alpha1.ExternalConfigApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("FrontProxy"):
		return &operatorv1alpha1.FrontProxyApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("FrontProxySpec"):
		return &operatorv1alpha1.FrontProxySpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("FrontProxyStatus"):
		return &operatorv1alpha1.FrontProxyStatusApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("ImageSpec"):
		return &operatorv1alpha1.ImageSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("Kubeconfig"):
		return &operatorv1alpha1.KubeconfigApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("KubeconfigSpec"):
		return &operatorv1alpha1.KubeconfigSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("KubeconfigTarget"):
		return &operatorv1alpha1.KubeconfigTargetApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("ObjectReference"):
		return &operatorv1alpha1.ObjectReferenceApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("OIDCConfiguration"):
		return &operatorv1alpha1.OIDCConfigurationApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("PathMappingEntry"):
		return &operatorv1alpha1.PathMappingEntryApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("RootShard"):
		return &operatorv1alpha1.RootShardApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("RootShardConfig"):
		return &operatorv1alpha1.RootShardConfigApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("RootShardSpec"):
		return &operatorv1alpha1.RootShardSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("RootShardStatus"):
		return &operatorv1alpha1.RootShardStatusApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("ServiceSpec"):
		return &operatorv1alpha1.ServiceSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("Shard"):
		return &operatorv1alpha1.ShardApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("ShardSpec"):
		return &operatorv1alpha1.ShardSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("ShardStatus"):
		return &operatorv1alpha1.ShardStatusApplyConfiguration{}

	}
	return nil
}

func NewTypeConverter(scheme *runtime.Scheme) *testing.TypeConverter {
	return &testing.TypeConverter{Scheme: scheme, TypeResolver: internal.Parser()}
}
