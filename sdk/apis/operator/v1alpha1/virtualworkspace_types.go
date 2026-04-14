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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VirtualWorkspace represents an external virtual workspace server that will be deployed as a
// single Deployment (plus a few auxiliary resources). Creating a per-shard virtual workspace means
// you have to create one VirtualWorkspace object for each shard, plus one for the root shard.
type VirtualWorkspace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualWorkspaceSpec   `json:"spec,omitempty"`
	Status VirtualWorkspaceStatus `json:"status,omitempty"`
}

// VirtualWorkspaceSpec defines the desired state of VirtualWorkspace.
type VirtualWorkspaceSpec struct {
	// Target configures to which shard this virtual workspace belongs.
	Target VirtualWorkspaceTarget `json:"target"`

	// External configures the domain and port under which this virtual workspace should be reachable.
	// This is important to allow kcp to generate the correct URLs for clients of virtual workspaces.
	External ExternalConfig `json:"external"`

	// Optional: Image overwrites the container image used to deploy the server.
	// If not specified, kcp's own virtual-workspace server will be deployed.
	Image *ImageSpec `json:"image,omitempty"`

	// Replicas configures how many instances of this server run in parallel.
	// Defaults to 2 if not set.
	Replicas *int32 `json:"replicas,omitempty"`

	// Resources overrides the default resource requests and limits.
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// CertificateTemplates allows to customize the properties on the generated
	// certificates for this server.
	CertificateTemplates CertificateTemplateMap `json:"certificateTemplates,omitempty"`

	// Optional: ServiceTemplate configures the Kubernetes Service created for this server.
	ServiceTemplate *ServiceTemplate `json:"serviceTemplate,omitempty"`

	// Optional: DeploymentTemplate configures the Kubernetes Deployment created for this server.
	DeploymentTemplate *DeploymentTemplate `json:"deploymentTemplate,omitempty"`

	// CABundle references a v1.Secret object that contains the CA bundle that should be used
	// to validate the API server's TLS certificate. The secret must contain a key named `tls.crt`
	// that holds the PEM encoded CA certificate. It will be merged into the
	// "external-logical-cluster-admin-kubeconfig" kubeconfig under the `certificate-authority-data`
	// field.
	// If not specified, the kubeconfig will use the CA bundle of the root shard or front-proxy
	// referenced in the Target field. It will NOT be used to configure the API server's own TLS
	// certificate or any other component.
	//
	// +optional
	CABundleSecretRef *corev1.LocalObjectReference `json:"caBundleSecretRef,omitempty"`

	// Optional: ExtraArgs defines additional command line arguments to pass to the server container.
	ExtraArgs []string `json:"extraArgs,omitempty"`

	// Optional: Logging configures the logging settings for the server.
	Logging *LoggingSpec `json:"logging,omitempty"`

	// ClusterDomain is the DNS domain for services in the cluster. Defaults to "cluster.local" if not set.
	//
	// +optional
	ClusterDomain string `json:"clusterDomain,omitempty"`
}

// VirtualWorkspaceTarget configures which shard or root shard a virtual workspace is connected to.
// This influences the certificates and CAs mounted to it.
// +kubebuilder:validation:XValidation:rule="(has(self.rootShardRef) && !has(self.shardRef)) || (!has(self.rootShardRef) && has(self.shardRef))",message="Must specify exactly one of rootShardRef or shardRef"
type VirtualWorkspaceTarget struct {
	// RootShardRef is used to connect this virtual workspace to the root shard. This can be used for
	// per-shard VWs, in which case having one VirtualWorkspace object with a rootShardRef is done
	// because the root shard is technically also a shard. But it can also be used for "singleton"
	// virtual workspaces, where one single deployment is meant to connect to all shards. In this
	// case the VW will most likely need to connect to the root shard to query and watch the Shard
	// objects.
	//
	// Mutually exclusive with shardRef.
	RootShardRef *corev1.LocalObjectReference `json:"rootShardRef,omitempty"`

	// ShardRef is used to connect this virtual workspace to one specific shard.
	//
	// Mutually exclusive with rootShardRef.
	ShardRef *corev1.LocalObjectReference `json:"shardRef,omitempty"`
}

// VirtualWorkspaceStatus defines the observed state of VirtualWorkspace
type VirtualWorkspaceStatus struct {
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true

// VirtualWorkspaceList contains a list of VirtualWorkspace
type VirtualWorkspaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualWorkspace `json:"items"`
}
