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
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	DefaultCADuration          = metav1.Duration{Duration: time.Hour * 24 * 365 * 10}
	DefaultCARenewal           = metav1.Duration{Duration: time.Hour * 24 * 30}
	DefaultCertificateDuration = metav1.Duration{Duration: time.Hour * 24 * 365}
	DefaultCertificateRenewal  = metav1.Duration{Duration: time.Hour * 24 * 7}
)

// ImageSpec defines settings for using a specific image and overwriting the default images used.
type ImageSpec struct {
	// Repository is the container image repository to use for KCP containers. Defaults to `ghcr.io/kcp-dev/kcp`.
	Repository string `json:"repository,omitempty"`
	// Tag is the container image tag to use for KCP containers. Defaults to the latest kcp release that the operator supports.
	Tag string `json:"tag,omitempty"`
	// Optional: ImagePullSecrets is a list of secret references that should be used as image pull secrets (e.g. when a private registry is used).
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
}

type RootShardConfig struct {
	// Reference references a local RootShard object.
	Reference *corev1.LocalObjectReference `json:"ref,omitempty"`
}

type EtcdConfig struct {
	// Endpoints is a list of http urls at which etcd nodes are available. The expected format is "https://etcd-hostname:2379".
	Endpoints []string `json:"endpoints"`
	// ClientCert configures the client certificate used to access etcd.
	// +optional
	TLSConfig *EtcdTLSConfig `json:"tlsConfig,omitempty"`
}

type EtcdTLSConfig struct {
	// SecretRef is the reference to a v1.Secret object that contains the TLS certificate.
	SecretRef corev1.LocalObjectReference `json:"secretRef"`
}

// ObjectReference is a reference to an object with a given name, kind and group.
type ObjectReference struct {
	// Name of the object being referred to.
	Name string `json:"name"`
	// Kind of the object being referred to.
	// +optional
	Kind string `json:"kind,omitempty"`
	// Group of the object being referred to.
	// +optional
	Group string `json:"group,omitempty"`
}

type Certificate string

const (
	// ServerCertificate is a generic server certificate for serving HTTPS.
	ServerCertificate Certificate = "server"
	// ClientCertificate is a generic client certificate.
	ClientCertificate Certificate = "client"

	ServiceAccountCertificate              Certificate = "service-account"
	VirtualWorkspacesCertificate           Certificate = "virtual-workspaces"
	RequestHeaderClientCertificate         Certificate = "requestheader"
	KubeconfigCertificate                  Certificate = "kubeconfig"
	AdminKubeconfigClientCertificate       Certificate = "admin-kubeconfig"
	LogicalClusterAdminCertificate         Certificate = "logical-cluster-admin"
	ExternalLogicalClusterAdminCertificate Certificate = "external-logical-cluster-admin"
)

type CA string

const (
	RootCA                CA = "root"
	ServerCA              CA = "server"
	ServiceAccountCA      CA = "service-account"
	ClientCA              CA = "client"
	FrontProxyClientCA    CA = "front-proxy-client"
	RequestHeaderClientCA CA = "requestheader-client"
)

type CertificateTemplateMap map[string]CertificateTemplate

func (m CertificateTemplateMap) CertificateTemplate(cert Certificate) CertificateTemplate {
	return m[string(cert)]
}

func (m CertificateTemplateMap) CATemplate(ca CA) CertificateTemplate {
	return m[string(ca)+"-ca"]
}

type CertificateTemplate struct {
	Metadata *CertificateMetadataTemplate `json:"metadata,omitempty"`
	Spec     *CertificateSpecTemplate     `json:"spec,omitempty"`
}

type CertificateMetadataTemplate struct {
	// Annotations is a key value map to be copied to the target Certificate.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels is a key value map to be copied to the target Certificate.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

type CertificateSpecTemplate struct {
	// Requested set of X509 certificate subject attributes.
	// More info: https://datatracker.ietf.org/doc/html/rfc5280#section-4.1.2.6
	//
	// +optional
	Subject *X509Subject `json:"subject,omitempty"`

	// Requested DNS subject alternative names. The values given here will be merged into the
	// DNS names determined automatically by the kcp-operator.
	//
	// +optional
	DNSNames []string `json:"dnsNames,omitempty"`

	// Requested IP address subject alternative names. The values given here will be merged into the
	// DNS names determined automatically by the kcp-operator.
	//
	// +optional
	IPAddresses []string `json:"ipAddresses,omitempty"`

	// Defines annotations and labels to be copied to the Certificate's Secret.
	// Labels and annotations on the Secret will be changed as they appear on the
	// SecretTemplate when added or removed. SecretTemplate annotations are added
	// in conjunction with, and cannot overwrite, the base set of annotations
	// cert-manager sets on the Certificate's Secret.
	// +optional
	SecretTemplate *CertificateSecretTemplate `json:"secretTemplate,omitempty"`

	// Requested 'duration' (i.e. lifetime) of the Certificate. Note that the
	// issuer may choose to ignore the requested duration, just like any other
	// requested attribute.
	//
	// If unset, this defaults to 90 days.
	// Minimum accepted duration is 1 hour.
	// Value must be in units accepted by Go time.ParseDuration https://golang.org/pkg/time/#ParseDuration.
	// +optional
	Duration *metav1.Duration `json:"duration,omitempty"`

	// How long before the currently issued certificate's expiry cert-manager should
	// renew the certificate. For example, if a certificate is valid for 60 minutes,
	// and `renewBefore=10m`, cert-manager will begin to attempt to renew the certificate
	// 50 minutes after it was issued (i.e. when there are 10 minutes remaining until
	// the certificate is no longer valid).
	//
	// NOTE: The actual lifetime of the issued certificate is used to determine the
	// renewal time. If an issuer returns a certificate with a different lifetime than
	// the one requested, cert-manager will use the lifetime of the issued certificate.
	//
	// If unset, this defaults to 1/3 of the issued certificate's lifetime.
	// Minimum accepted value is 5 minutes.
	// Value must be in units accepted by Go time.ParseDuration https://golang.org/pkg/time/#ParseDuration.
	// Cannot be set if the `renewBeforePercentage` field is set.
	// +optional
	RenewBefore *metav1.Duration `json:"renewBefore,omitempty"`

	// Private key options. These include the key algorithm and size, the used
	// encoding and the rotation policy.
	// +optional
	PrivateKey *CertificatePrivateKeyTemplate `json:"privateKey,omitempty"`
}

type CertificatePrivateKeyTemplate struct {
	// RotationPolicy controls how private keys should be regenerated when a
	// re-issuance is being processed.
	//
	// If set to `Never`, a private key will only be generated if one does not
	// already exist in the target `spec.secretName`. If one does exist but it
	// does not have the correct algorithm or size, a warning will be raised
	// to await user intervention.
	// If set to `Always`, a private key matching the specified requirements
	// will be generated whenever a re-issuance occurs.
	// Default is `Never` for backward compatibility.
	// +optional
	RotationPolicy PrivateKeyRotationPolicy `json:"rotationPolicy,omitempty"`

	// The private key cryptography standards (PKCS) encoding for this
	// certificate's private key to be encoded in.
	//
	// If provided, allowed values are `PKCS1` and `PKCS8` standing for PKCS#1
	// and PKCS#8, respectively.
	// Defaults to `PKCS1` if not specified.
	// +optional
	Encoding PrivateKeyEncoding `json:"encoding,omitempty"`

	// Algorithm is the private key algorithm of the corresponding private key
	// for this certificate.
	//
	// If provided, allowed values are either `RSA`, `ECDSA` or `Ed25519`.
	// If `algorithm` is specified and `size` is not provided,
	// key size of 2048 will be used for `RSA` key algorithm and
	// key size of 256 will be used for `ECDSA` key algorithm.
	// key size is ignored when using the `Ed25519` key algorithm.
	// +optional
	Algorithm PrivateKeyAlgorithm `json:"algorithm,omitempty"`

	// Size is the key bit size of the corresponding private key for this certificate.
	//
	// If `algorithm` is set to `RSA`, valid values are `2048`, `4096` or `8192`,
	// and will default to `2048` if not specified.
	// If `algorithm` is set to `ECDSA`, valid values are `256`, `384` or `521`,
	// and will default to `256` if not specified.
	// If `algorithm` is set to `Ed25519`, Size is ignored.
	// No other values are allowed.
	// +optional
	Size int `json:"size,omitempty"`
}

// CertificateSecretTemplate defines the default labels and annotations
// to be copied to the Kubernetes Secret resource named in `CertificateSpec.secretName`.
type CertificateSecretTemplate struct {
	// Annotations is a key value map to be copied to the target Kubernetes Secret.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels is a key value map to be copied to the target Kubernetes Secret.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

type ConditionType string

const (
	ConditionTypeAvailable ConditionType = "Available"
	ConditionTypeRootShard ConditionType = "RootShard"
)

type ConditionReason string

const (
	// reasons for ConditionTypeAvailable

	ConditionReasonDeploymentUnavailable ConditionReason = "DeploymentUnavailable"
	ConditionReasonReplicasUp            ConditionReason = "ReplicasUp"
	ConditionReasonReplicasUnavailable   ConditionReason = "ReplicasUnavailable"

	// reasons for ConditionTypeRootShard

	ConditionReasonRootShardRefInvalid  ConditionReason = "InvalidReference"
	ConditionReasonRootShardRefNotFound ConditionReason = "RootShardNotFound"
	ConditionReasonRootShardRefValid    ConditionReason = "Valid"
)

type ServiceTemplate struct {
	Metadata *ServiceMetadataTemplate `json:"metadata,omitempty"`
	Spec     *ServiceSpecTemplate     `json:"spec,omitempty"`
}

// ServiceMetadataTemplate defines the default labels and annotations
// to be copied to the Kubernetes Service resource.
type ServiceMetadataTemplate struct {
	// Annotations is a key value map to be copied to the target Kubernetes Service.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels is a key value map to be copied to the target Kubernetes Service.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

type ServiceSpecTemplate struct {
	Type      corev1.ServiceType `json:"type,omitempty"`
	ClusterIP string             `json:"clusterIP,omitempty"`
}

type DeploymentTemplate struct {
	Metadata *DeploymentMetadataTemplate `json:"metadata,omitempty"`
	Spec     *DeploymentSpecTemplate     `json:"spec,omitempty"`
}

type DeploymentMetadataTemplate struct {
	// Annotations is a key value map to be copied to the target Deployment.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels is a key value map to be copied to the target Deployment.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

type DeploymentSpecTemplate struct {
	// Template describes the pods that will be created.
	Template *PodTemplateSpec `json:"template,omitempty"`
}

type PodTemplateSpec struct {
	Metadata *PodMetadataTemplate `json:"metadata,omitempty"`
	Spec     *PodSpecTemplate     `json:"spec,omitempty"`
}

type PodMetadataTemplate struct {
	// Annotations is a key value map to be copied to the Pod.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels is a key value map to be copied to the Pod.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

type PodSpecTemplate struct {
	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	// +mapType=atomic
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// If specified, the pod's scheduling constraints
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// If specified, the pod's tolerations.
	// +optional
	// +listType=atomic
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// HostAliases is an optional list of hosts and IPs that will be injected into the pod's hosts
	// file if specified.
	// +optional
	// +patchMergeKey=ip
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=ip
	HostAliases []corev1.HostAlias `json:"hostAliases,omitempty"`

	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use.
	// More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=name
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
}
