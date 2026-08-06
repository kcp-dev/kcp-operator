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

// CompiledFrontProxySpec defines the fully resolved render input for a kcp front-proxy.
//
// EXPERIMENTAL: This type is part of an experimental, internal API that
// is subject to change at any time.
type CompiledFrontProxySpec struct {
	// FrontProxy is the resolved spec of the FrontProxy this object was compiled from.
	FrontProxy operatorv1alpha1.FrontProxySpec `json:"frontProxy"`

	// RootShard is the resolved spec of the RootShard the front-proxy belongs to.
	RootShard NamedRootShardSpec `json:"rootShard"`

	// Optional: Shards are the names of all Shards belonging to the root shard.
	Shards []string `json:"shards,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=".status.phase",name="Phase",type="string"
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// CompiledFrontProxy is the fully resolved render input for a kcp front-proxy.
//
// EXPERIMENTAL: This type is part of an experimental, internal API that
// is subject to change at any time.
type CompiledFrontProxy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CompiledFrontProxySpec            `json:"spec,omitempty"`
	Status operatorv1alpha1.FrontProxyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CompiledFrontProxyList contains a list of CompiledFrontProxy.
//
// EXPERIMENTAL: This type is part of an experimental, internal API that
// is subject to change at any time.
type CompiledFrontProxyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CompiledFrontProxy `json:"items"`
}
