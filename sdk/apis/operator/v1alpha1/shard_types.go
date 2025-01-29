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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ShardSpec defines the desired state of Shard
type ShardSpec struct {
	CommonShardSpec `json:",inline"`

	RootShard RootShardConfig `json:"rootShard"`
}

type CommonShardSpec struct {
	ClusterDomain string `json:"clusterDomain,omitempty"`

	// Etcd configures the etcd cluster that this shard should be using.
	Etcd EtcdConfig `json:"etcd"`

	Image *ImageSpec `json:"image,omitempty"`

	// Replicas configures how many instances of this shard run in parallel. Defaults to 2 if not set.
	Replicas *int32 `json:"replicas,omitempty"`
}

// ShardStatus defines the observed state of Shard
type ShardStatus struct {
	Phase ShardPhase `json:"phase,omitempty"`

	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type ShardPhase string

const (
	ShardPhaseProvisioning ShardPhase = "Provisioning"
	ShardPhaseRunning      ShardPhase = "Running"
	ShardPhaseDeleting     ShardPhase = "Deleting"
)

type ShardConditionType string

const (
	ShardConditionTypeAvailable ShardConditionType = "Available"
)

type ShardConditionReason string

const (
	ShardConditionReasonDeploymentUnavailable ShardConditionReason = "DeploymentUnavailable"
	ShardConditionReasonReplicasUp            ShardConditionReason = "ReplicasUp"
	ShardConditionReasonReplicasUnavailable   ShardConditionReason = "ReplicasUnavailable"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=".spec.rootShard.ref.name",name="RootShard",type="string"
// +kubebuilder:printcolumn:JSONPath=".status.phase",name="Phase",type="string"
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// Shard is the Schema for the shards API
type Shard struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ShardSpec   `json:"spec,omitempty"`
	Status ShardStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ShardList contains a list of Shard
type ShardList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Shard `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Shard{}, &ShardList{})
}
