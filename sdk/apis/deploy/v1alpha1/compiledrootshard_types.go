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

// CompiledRootShardSpec defines the fully resolved render input for a kcp root shard.
type CompiledRootShardSpec struct {
	// RootShard is the resolved spec of the RootShard this object was compiled from.
	RootShard operatorv1alpha1.RootShardSpec `json:"rootShard"`

	// Optional: VirtualWorkspace is the resolved spec of the VirtualWorkspace associated with the root shard.
	VirtualWorkspace *NamedVirtualWorkspaceSpec `json:"virtualWorkspace,omitempty"`

	// Optional: Shards are the names of all Shards belonging to the root shard.
	Shards []string `json:"shards,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=".status.phase",name="Phase",type="string"
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// CompiledRootShard is the fully resolved render input for a kcp root shard.
type CompiledRootShard struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CompiledRootShardSpec            `json:"spec,omitempty"`
	Status operatorv1alpha1.RootShardStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CompiledRootShardList contains a list of CompiledRootShard.
type CompiledRootShardList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CompiledRootShard `json:"items"`
}
