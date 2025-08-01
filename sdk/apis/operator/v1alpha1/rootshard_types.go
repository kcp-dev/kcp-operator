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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RootShardSpec defines the desired state of RootShard.
type RootShardSpec struct {
	CommonShardSpec `json:",inline"`

	External ExternalConfig `json:"external"`

	// Cache configures the cache server (with a Kubernetes-like API) used by a sharded kcp instance.
	Cache CacheConfig `json:"cache"`

	// Certificates configures how the operator should create the kcp root CA, from which it will
	// then create all other sub CAs and leaf certificates.
	Certificates Certificates `json:"certificates"`
}

type ExternalConfig struct {
	// Hostname is the external name of the kcp instance. This should be matched by a DNS
	// record pointing to the kcp-front-proxy Service's external IP address.
	Hostname string `json:"hostname"`
	Port     uint32 `json:"port"`
}

// Certificates configures how certificates for kcp should be created.
type Certificates struct {
	// IssuerRef points to a pre-existing cert-manager Issuer or ClusterIssuer that shall be used
	// to acquire new certificates. This field is mutually exclusive with caSecretRef.
	IssuerRef *ObjectReference `json:"issuerRef,omitempty"`

	// CASecretRef can be used as an alternative to the IssuerRef: This field allows to configure
	// a pre-existing CA certificate that should be used as sign kcp certificates.
	// This Secret must contain both the certificate and the private key so that new sub certificates
	// can be signed and created from this CA. This field is mutually exclusive with issuerRef.
	CASecretRef *corev1.LocalObjectReference `json:"caSecretRef,omitempty"`
}

type CacheConfig struct {
	// Embedded configures settings for starting the cache server embedded in the root shard.
	Embedded *EmbeddedCacheConfiguration `json:"embedded,omitempty"`
}

type EmbeddedCacheConfiguration struct {
	// Enabled enables or disables running the cache server as embedded.
	Enabled bool `json:"enabled"`
}

// RootShardStatus defines the observed state of RootShard
type RootShardStatus struct {
	Phase RootShardPhase `json:"phase,omitempty"`

	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Shards is a list of shards that are currently registered with this root shard.
	// +listType=map
	// +listMapKey=name
	// +optional
	Shards []ShardReference `json:"shards,omitempty"`
}

type ShardReference struct {
	// Name is the name of the shard.
	Name string `json:"name"`
}

type RootShardPhase string

const (
	RootShardPhaseProvisioning RootShardPhase = "Provisioning"
	RootShardPhaseRunning      RootShardPhase = "Running"
	RootShardPhaseDeleting     RootShardPhase = "Deleting"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=".spec.external.hostname",name="Hostname",type="string"
// +kubebuilder:printcolumn:JSONPath=".status.phase",name="Phase",type="string"
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// RootShard is the Schema for the kcpinstances API
type RootShard struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RootShardSpec   `json:"spec,omitempty"`
	Status RootShardStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RootShardList contains a list of RootShard
type RootShardList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RootShard `json:"items"`
}
