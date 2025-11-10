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
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type WorkspaceObjectManagementPolicy string

const (
	WorkspaceObjectManagementPolicyAll    WorkspaceObjectManagementPolicy = "*"
	WorkspaceObjectManagementPolicyCreate WorkspaceObjectManagementPolicy = "Create"
	WorkspaceObjectManagementPolicyUpdate WorkspaceObjectManagementPolicy = "Update"
	WorkspaceObjectManagementPolicyDelete WorkspaceObjectManagementPolicy = "Delete"
)

// WorkspaceObjectSpec defines the desired state of WorkspaceObject
type WorkspaceObjectSpec struct {
	// RootShard is a reference to the root shard that holds the workspace in
	// which the manifest should be applied.
	RootShard RootShardConfig `json:"rootShard"`

	// Workspace specifies in which workspace the manifest should be applied.
	Workspace WorkspaceConfig `json:"workspace"`

	// Manifest is the desired state of the object in the workspace.
	Manifest extv1.JSON `json:"manifest"`

	// ManagementPolicies specify the operations that should be executed by the
	// operator. By default, the operator manages creation, update and deletion
	// of objects.
	//
	// +kubebuilder:default={"*"}
	ManagementPolicies []WorkspaceObjectManagementPolicy `json:"managementPolicies"`
}

// WorkspaceObjectStatus defines the observed state of WorkspaceObject
type WorkspaceObjectStatus struct {
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Manifest is the actual state of the object in the workspace.
	Manifest *extv1.JSON `json:"manifest,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=".spec.rootShard.ref.name",name="RootShard",type="string"
// +kubebuilder:printcolumn:JSONPath=".status.conditions[?(@.type=='Available')].reason",name="Status",type="string"
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// WorkspaceObject is the Schema for the WorkspaceObjects API
type WorkspaceObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkspaceObjectSpec   `json:"spec,omitempty"`
	Status WorkspaceObjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WorkspaceObjectList contains a list of WorkspaceObject
type WorkspaceObjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkspaceObject `json:"items"`
}
