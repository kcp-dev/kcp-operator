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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RootShardSpec defines the desired state of RootShard.
type RootShardSpec struct {
	CommonShardSpec `json:",inline"`

	External ExternalConfig `json:"external"`

	// Cache configures the cache server (with a Kubernetes-like API) used by a sharded kcp instance.
	Cache CacheConfig `json:"cache"`

	Certificates Certificates `json:"certificates"`
}

type ExternalConfig struct {
	// Hostname is the external name of the kcp instance. This should be matched by a DNS
	// record pointing to the kcp-front-proxy Service's external IP address.
	Hostname string `json:"hostname"`
	Port     uint32 `json:"port"`
}

type Certificates struct {
	IssuerRef   *ObjectReference             `json:"issuerRef,omitempty"`
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

type OIDCConfiguration struct {
	Enabled bool `json:"enabled"`

	// IssuerURL is used for the OIDC issuer URL. Only https URLs will be accepted.
	IssuerURL string `json:"issuerURL"`
	// ClientID is the OIDC client ID configured on the issuer side for this KCP instance.
	ClientID string `json:"clientID"`

	// Optionally provide the client secret for the OIDC client. This is not used by KCP itself, but is used to generate
	// a OIDC kubeconfig that can be shared with users to log in via the OIDC provider.
	ClientSecret string `json:"clientSecret,omitempty"`

	// Experimental: Optionally provides a custom claim for fetching groups. The claim must be a string or an array of strings.
	GroupsClaim string `json:"groupsClaim,omitempty"`
	// Optionally uses a custom claim for fetching the username. This defaults to "sub" if unset.
	UsernameClaim string `json:"usernameClaim,omitempty"`

	// Optionally sets a custom groups prefix. This defaults to "oidc:" if unset, which means a group called "group1"
	// on the OIDC side will be recognised as "oidc:group1" in KCP.
	GroupsPrefix string `json:"groupsPrefix,omitempty"`
	// Optionally sets a custom username prefix. This defaults to "oidc:" if unset, which means a user called "user@example.com"
	// on the OIDC side will be recognised as "oidc:user@example.com" in KCP.
	UsernamePrefix string `json:"usernamePrefix,omitempty"`
}

// RootShardStatus defines the observed state of RootShard
type RootShardStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// RootShard is the Schema for the kcpinstances API
type RootShard struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RootShardSpec   `json:"spec,omitempty"`
	Status RootShardStatus `json:"status,omitempty"`
}

func (r *RootShard) GetResourceLabels() map[string]string {
	return map[string]string{
		appNameLabel:      "kcp",
		appInstanceLabel:  r.Name,
		appManagedByLabel: "kcp-operator",
		appComponentLabel: "rootshard",
	}
}

func (r *RootShard) GetShardBaseURL() string {
	clusterDomain := r.Spec.ClusterDomain
	if clusterDomain == "" {
		clusterDomain = "cluster.local"
	}

	return fmt.Sprintf("https://%s-kcp.%s.svc.%s:6443", r.Name, r.Namespace, clusterDomain)
}

type Certificate string

const (
	ServerCertificate            Certificate = "server"
	ServiceAccountCertificate    Certificate = "service-account"
	VirtualWorkspacesCertificate Certificate = "virtual-workspaces"
)

func (r *RootShard) GetCertificateName(certName Certificate) string {
	return fmt.Sprintf("%s-%s", r.Name, certName)
}

type CA string

const (
	RootCA                CA = "root"
	ServerCA              CA = "server"
	ServiceAccountCA      CA = "service-account"
	ClientCA              CA = "client"
	RequestHeaderClientCA CA = "requestheader-client"
)

func (r *RootShard) GetCAName(caName CA) string {
	if caName == RootCA {
		return fmt.Sprintf("%s-ca", r.Name)
	}
	return fmt.Sprintf("%s-%s-ca", r.Name, caName)
}

// +kubebuilder:object:root=true

// RootShardList contains a list of RootShard
type RootShardList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RootShard `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RootShard{}, &RootShardList{})
}
