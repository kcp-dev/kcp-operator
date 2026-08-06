/*
Copyright 2026 The kcp Authors.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// CompiledCacheServerSpec defines the fully resolved render input for a kcp cache server.
//
// EXPERIMENTAL: This type is part of an experimental, internal API that
// is subject to change at any time.
type CompiledCacheServerSpec struct {
	// CacheServer is the resolved spec of the CacheServer this object was compiled from.
	CacheServer operatorv1alpha1.CacheServerSpec `json:"cacheServer"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// CompiledCacheServer is the fully resolved render input for a kcp cache server.
//
// EXPERIMENTAL: This type is part of an experimental, internal API that
// is subject to change at any time.
type CompiledCacheServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CompiledCacheServerSpec            `json:"spec,omitempty"`
	Status operatorv1alpha1.CacheServerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CompiledCacheServerList contains a list of CompiledCacheServer.
//
// EXPERIMENTAL: This type is part of an experimental, internal API that
// is subject to change at any time.
type CompiledCacheServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CompiledCacheServer `json:"items"`
}
