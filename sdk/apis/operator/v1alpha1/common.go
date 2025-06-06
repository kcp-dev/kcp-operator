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
