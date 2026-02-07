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

// ShardSpec defines the desired state of Shard
type ShardSpec struct {
	CommonShardSpec `json:",inline"`

	RootShard RootShardConfig `json:"rootShard"`

	// Optional: Configure an external cache server for this shard. If not configured, the cache
	// settings of the RootShard will be used.
	Cache *ShardCacheConfig `json:"cache,omitempty"`
}

type CommonShardSpec struct {
	// ClusterDomain is the DNS domain for services in the cluster. Defaults to "cluster.local" if not set.
	// +optional
	ClusterDomain string `json:"clusterDomain,omitempty"`

	// ShardBaseURL is the base URL under which this shard should be reachable. This is used to configure
	// the external URL. If not provided, the operator will use kubernetes service address to generate it.
	// +optional
	ShardBaseURL string `json:"shardBaseURL,omitempty"`

	// Etcd configures the etcd cluster that this shard should be using.
	Etcd EtcdConfig `json:"etcd"`

	Image *ImageSpec `json:"image,omitempty"`

	// Replicas configures how many instances of this shard run in parallel. Defaults to 2 if not set.
	Replicas *int32 `json:"replicas,omitempty"`

	// Resources overrides the default resource requests and limits.
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	Audit         *AuditSpec         `json:"audit,omitempty"`
	Authorization *AuthorizationSpec `json:"authorization,omitempty"`

	// Optional: Auth configures various aspects of Authentication and Authorization for this shard.
	Auth *AuthSpec `json:"auth,omitempty"`

	// CertificateTemplates allows to customize the properties on the generated
	// certificates for this shard.
	CertificateTemplates CertificateTemplateMap `json:"certificateTemplates,omitempty"`

	// Optional: ServiceTemplate configures the Kubernetes Service created for this shard.
	ServiceTemplate *ServiceTemplate `json:"serviceTemplate,omitempty"`

	// Optional: DeploymentTemplate configures the Kubernetes Deployment created for this shard.
	DeploymentTemplate *DeploymentTemplate `json:"deploymentTemplate,omitempty"`

	// CABundle references a v1.Secret object that contains the CA bundle
	// that should be used to validate the API server's TLS certificate.
	// The secret must contain a key named `tls.crt` that holds the PEM encoded CA certificate.
	// It will be merged into the "external-logical-cluster-admin-kubeconfig" kubeconfig under the `certificate-authority-data` field.
	// If not specified, the kubeconfig will use the CA bundle of the root shard or front-proxy referenced in the Target field.
	// It will NOT be used to configure the API server's own TLS certificate or any other component.
	// +optional
	CABundleSecretRef *corev1.LocalObjectReference `json:"caBundleSecretRef,omitempty"`

	// Optional: ExtraArgs defines additional command line arguments to pass to the shard container.
	ExtraArgs []string `json:"extraArgs,omitempty"`

	// Optional: Logging configures the logging settings for the shard.
	Logging *LoggingSpec `json:"logging,omitempty"`
}

type AuditSpec struct {
	Webhook *AuditWebhookSpec `json:"webhook,omitempty"`
}

type ShardCacheConfig struct {
	// Reference references a local CacheServer object.
	Reference *corev1.LocalObjectReference `json:"ref,omitempty"`
}

// +kubebuilder:validation:Enum="";batch;blocking;blocking-strict

type AuditWebhookMode string

const (
	AuditWebhookBatchMode          AuditWebhookMode = "batch"
	AuditWebhookBlockingMode       AuditWebhookMode = "blocking"
	AuditWebhookBlockingStrictMode AuditWebhookMode = "blocking-strict"
)

type AuditWebhookSpec struct {
	// The size of the buffer to store events before batching and writing. Only used in batch mode.
	BatchBufferSize int `json:"batchBufferSize,omitempty"`
	// The maximum size of a batch. Only used in batch mode.
	BatchMaxSize int `json:"batchMaxSize,omitempty"`
	// The amount of time to wait before force writing the batch that hadn't reached the max size.
	// Only used in batch mode.
	BatchMaxWait *metav1.Duration `json:"batchMaxWait,omitempty"`
	// Maximum number of requests sent at the same moment if ThrottleQPS was not utilized before.
	// Only used in batch mode.
	BatchThrottleBurst int `json:"batchThrottleBurst,omitempty"`
	// Whether batching throttling is enabled. Only used in batch mode.
	BatchThrottleEnable bool `json:"batchThrottleEnable,omitempty"`
	// Maximum average number of batches per second. Only used in batch mode.
	// This value is a floating point number, stored as a string (e.g. "3.1").
	BatchThrottleQPS string `json:"batchThrottleQPS,omitempty"`

	// Name of a Kubernetes Secret that contains a kubeconfig formatted file that defines the
	// audit webhook configuration.
	ConfigSecretName string `json:"configSecretName,omitempty"`
	// The amount of time to wait before retrying the first failed request.
	InitialBackoff *metav1.Duration `json:"initialBackoff,omitempty"`
	// Strategy for sending audit events. Blocking indicates sending events should block server
	// responses. Batch causes the backend to buffer and write events asynchronously.
	Mode AuditWebhookMode `json:"mode,omitempty"`
	// Whether event and batch truncating is enabled.
	TruncateEnabled bool `json:"truncateEnabled,omitempty"`
	// Maximum size of the batch sent to the underlying backend. Actual serialized size can be
	// several hundreds of bytes greater. If a batch exceeds this limit, it is split into several
	// batches of smaller size.
	TruncateMaxBatchSize int `json:"truncateMaxBatchSize,omitempty"`
	// Maximum size of the audit event sent to the underlying backend. If the size of an event
	// is greater than this number, first request and response are removed, and if this doesn't
	// reduce the size enough, event is discarded.
	TruncateMaxEventSize int `json:"truncateMaxEventSize,omitempty"`
	// API group and version used for serializing audit events written to webhook.
	Version string `json:"version,omitempty"`
}

type AuthorizationSpec struct {
	Webhook *AuthorizationWebhookSpec `json:"webhook,omitempty"`
}

type AuthorizationWebhookSpec struct {
	// A list of HTTP paths to skip during authorization, i.e. these are authorized without contacting the 'core' kubernetes server.
	// If specified, completely overwrites the default of [/healthz,/readyz,/livez].
	AllowPaths []string `json:"allowPaths,omitempty"`
	// The duration to cache 'authorized' responses from the webhook authorizer.
	CacheAuthorizedTTL *metav1.Duration `json:"cacheAuthorizedTTL,omitempty"`
	// The duration to cache 'unauthorized' responses from the webhook authorizer.
	CacheUnauthorizedTTL *metav1.Duration `json:"cacheUnauthorizedTTL,omitempty"`
	// Name of a Kubernetes Secret that contains a kubeconfig formatted file that defines the
	// authorization webhook configuration.
	ConfigSecretName string `json:"configSecretName,omitempty"`
	// The API version of the authorization.k8s.io SubjectAccessReview to send to and expect from the webhook.
	Version string `json:"version,omitempty"`
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
	ShardPhaseBundled      ShardPhase = "Bundled"
	ShardPhaseDeleting     ShardPhase = "Deleting"
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
