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
)

const (
	appNameLabel      = "app.kubernetes.io/name"
	appInstanceLabel  = "app.kubernetes.io/instance"
	appManagedByLabel = "app.kubernetes.io/managed-by"
	appComponentLabel = "app.kubernetes.io/component"

	defaultClusterDomain string = "cluster.local"
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
	Reference *corev1.ObjectReference `json:"ref,omitempty"`
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
	// Name of the resource being referred to.
	Name string `json:"name"`
	// Kind of the resource being referred to.
	// +optional
	Kind string `json:"kind,omitempty"`
	// Group of the resource being referred to.
	// +optional
	Group string `json:"group,omitempty"`
}
