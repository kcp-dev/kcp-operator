/*
Copyright 2026 The KCP Authors.

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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// BundleSpec defines the desired state of Bundle.
type BundleSpec struct {
	// Target configured which object bundle is targeting.
	Target BundleTarget `json:"target"`
}

// BundleTarget defines configuration bundle target.
type BundleTarget struct {
	RootShardRef  *corev1.LocalObjectReference `json:"rootShardRef,omitempty"`
	ShardRef      *corev1.LocalObjectReference `json:"shardRef,omitempty"`
	FrontProxyRef *corev1.LocalObjectReference `json:"frontProxyRef,omitempty"`
}

// String returns a string representation of the BundleTarget for display purposes.
func (b BundleTarget) String() string {
	switch {
	case b.RootShardRef != nil:
		return fmt.Sprintf("RootShard/%s", b.RootShardRef.Name)
	case b.ShardRef != nil:
		return fmt.Sprintf("Shard/%s", b.ShardRef.Name)
	case b.FrontProxyRef != nil:
		return fmt.Sprintf("FrontProxy/%s", b.FrontProxyRef.Name)
	default:
		return ""
	}
}

// BundleStatus defines the observed state of Bundle.
type BundleStatus struct {
	// State is bundle state
	State BundleState `json:"state,omitempty"`

	// TargetName is the name of the target object for display purposes
	TargetName string `json:"targetName,omitempty"`

	// Objects are list of objects with their state for this bundle.
	Objects []BundleObjectStatus `json:"objects"`

	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type BundleState string

const (
	// BundleStateProvisioning indicates state where bundle assets are still being collected.
	BundleStateProvisioning BundleState = "Provisioning"
	// BundleStateReady indicates state where bundle assets are ready to be used.
	BundleStateReady BundleState = "Ready"
	// BundleStateDeleting bundle being deleted.
	BundleStateDeleting BundleState = "Deleting"
)

type BundleObjectState string

const (
	// BundleObjectStateReady indicates ready state.
	BundleObjectStateReady BundleObjectState = "Ready"
	// BundleObjectStateNotReady indicates not ready state, where object is not available or corrupted. See message for more details.
	BundleObjectStateNotReady BundleObjectState = "NotReady"
)

// BundleObjectStatus represents individual object status for the bundle.
type BundleObjectStatus struct {
	Object  string            `json:"object"`
	State   BundleObjectState `json:"state"`
	Message string            `json:"message,omitempty"`
}

// BundleObject represents a Kubernetes object that should be included in a Bundle.
type BundleObject struct {
	// GVR is the GroupVersionResource for this object
	GVR schema.GroupVersionResource
	// Name is the name of the object
	Name string
	// Namespace is the namespace of the object (if namespaced)
	Namespace string
}

// String returns string representation of the BundleObject, used in the status.
func (b *BundleObject) String() string {
	group := b.GVR.Group
	if group == "" {
		group = "core"
	}
	// format: resource.group.version:namespace/name
	return fmt.Sprintf("%s.%s.%s:%s/%s", b.GVR.Resource, group, b.GVR.Version, b.Namespace, b.Name)
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=".status.targetName",name="Target",type="string"
// +kubebuilder:printcolumn:JSONPath=".status.phase",name="Phase",type="string"
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// Bundle is the Schema for the configuration bundle, intended to provide assets for shard deployments outside operator
// managed cluster.
type Bundle struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BundleSpec   `json:"spec,omitempty"`
	Status BundleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BundleList contains a list of Bundle.
type BundleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Bundle `json:"items"`
}
