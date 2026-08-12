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

	// Optional: Command overrides the entrypoint of the server container. Set this when the
	// image is not kcp's own virtual-workspace server, whose binary lives at
	// `/virtual-workspaces`.
	//
	// Setting this also changes which command line arguments the operator generates: only the
	// arguments any aggregated apiserver understands are passed (TLS serving certificate,
	// client and requestheader CA configuration, bind address and port, and `--kubeconfig`).
	// The arguments only kcp's own server accepts (`--shard-external-url` and
	// `--cache-kubeconfig`) are omitted, as is the cache server's kubeconfig mount, because a
	// custom server would reject the unknown flags and refuse to start. Use ExtraArgs for
	// whatever else the custom server needs.
	//
	// +optional
	Command []string `json:"command,omitempty"`

	// Optional: InitContainers are run in order before the server container starts. They are
	// meant for the bootstrapping a custom virtual workspace has to perform against kcp before
	// it can serve, like creating its own workspace and installing its APIExport.
	//
	// Each init container inherits the server container's volume mounts, so it can reach the
	// same certificates without having to know the operator's mount paths.
	//
	// +optional
	// +listType=map
	// +listMapKey=name
	InitContainers []VirtualWorkspaceInitContainer `json:"initContainers,omitempty"`

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

	// ClientCABundleRef references a v1.Secret containing an additional client CA bundle
	// for client certificate authentication. The secret must contain a key named `tls.crt`.
	// This CA bundle will be merged with the root shard's client CA and its ClientCABundleRef
	// (if configured).
	//
	// +optional
	ClientCABundleRef *corev1.LocalObjectReference `json:"clientCABundleRef,omitempty"`

	// KubeconfigSecretRef references a v1.Secret containing a `kubeconfig` key that the server
	// container uses to connect to kcp, mounted at `/etc/kcp/server-kubeconfig/kubeconfig`. When
	// set, the server's `--kubeconfig` points at it instead of the target's logical-cluster-admin
	// kubeconfig, which is what lets a server run with an identity other than the broadly
	// privileged one the operator provisions by default.
	//
	// Init containers are unaffected and select their own credentials, see the init container's
	// own KubeconfigSecretRef.
	//
	// The Secret is usually produced by a Kubeconfig object, whose kubeconfig is self-contained
	// (the client certificate, key and CA are embedded) and already scoped to
	// spec.targetWorkspace.
	//
	// +optional
	KubeconfigSecretRef *corev1.LocalObjectReference `json:"kubeconfigSecretRef,omitempty"`

	// Optional: ExtraArgs defines additional command line arguments to pass to the server container.
	ExtraArgs []string `json:"extraArgs,omitempty"`

	// Optional: ExtraVolumes defines additional volumes that should be added to the virtual workspace Pod.
	//
	// +optional
	// +listType=map
	// +listMapKey=name
	ExtraVolumes []corev1.Volume `json:"extraVolumes,omitempty"`

	// Optional: ExtraVolumeMounts defines additional volume mounts that should be added to the
	// virtual workspace server container. Each entry's `name` must match the `name` of a volume
	// defined in ExtraVolumes (or one of the volumes the operator manages). Init containers do not
	// inherit these; they carry their own VolumeMounts.
	//
	// +optional
	// +listType=map
	// +listMapKey=name
	ExtraVolumeMounts []corev1.VolumeMount `json:"extraVolumeMounts,omitempty"`

	// Optional: Logging configures the logging settings for the server.
	Logging *LoggingSpec `json:"logging,omitempty"`

	// ClusterDomain is the DNS domain for services in the cluster. Defaults to "cluster.local" if not set.
	//
	// +optional
	ClusterDomain string `json:"clusterDomain,omitempty"`
}

// VirtualWorkspaceInitContainer describes a container that runs to completion before the virtual
// workspace server container is started.
type VirtualWorkspaceInitContainer struct {
	// Name of the init container. Must be unique within the VirtualWorkspace.
	//
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Optional: Image overwrites the container image used for this init container. If not
	// specified, the same image as the server container is used, which is the common case for
	// servers that ship their bootstrapping binary alongside the server binary.
	Image *ImageSpec `json:"image,omitempty"`

	// Optional: Command overrides the entrypoint of the init container image.
	Command []string `json:"command,omitempty"`

	// Optional: Args are the command line arguments passed to the init container.
	Args []string `json:"args,omitempty"`

	// KubeconfigSecretRef references a v1.Secret containing a `kubeconfig` key, mounted into this
	// init container at `/etc/kcp/init-kubeconfig/kubeconfig`.
	//
	// Bootstrapping usually has to work across the workspace tree, which only resolves through
	// the front-proxy, and needs administrative rights the server itself should not hold. Both
	// are covered by pointing this at a Kubeconfig object that targets a FrontProxy with the
	// `kcp-admin` username in the `system:kcp:admin` group.
	//
	// +optional
	KubeconfigSecretRef *corev1.LocalObjectReference `json:"kubeconfigSecretRef,omitempty"`

	// Optional: Resources overrides the resource requests and limits, which default to the same
	// values the server container uses.
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// Optional: Volumes are added to the virtual workspace Pod for this init container's benefit.
	// Volumes are pod-scoped, so declaring one here makes it mountable by every container, but it
	// is only mounted into this init container. If the same volume name is declared more than once
	// across the VirtualWorkspace and its init containers, the first declaration wins.
	//
	// +optional
	// +listType=map
	// +listMapKey=name
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// Optional: VolumeMounts are mounted into this init container alone; they are not applied to
	// the server container or to any other init container. Each mount must reference a volume
	// defined in this init container's Volumes, in the VirtualWorkspace's ExtraVolumes, or one of
	// the volumes the operator manages. The certificates and kubeconfigs the operator mounts into
	// every container are added on top of these.
	//
	// +optional
	// +listType=map
	// +listMapKey=name
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`
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
