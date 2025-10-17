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

// FrontProxySpec defines the desired state of FrontProxy.
type FrontProxySpec struct {
	// RootShard configures the kcp root shard that this front-proxy instance should connect to.
	RootShard RootShardConfig `json:"rootShard"`
	// Optional: Replicas configures the replica count for the front-proxy Deployment.
	Replicas *int32 `json:"replicas,omitempty"`
	// Resources overrides the default resource requests and limits.
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
	// Optional: Auth configures various aspects of Authentication and Authorization for this front-proxy instance.
	// If OIDC is enabled, it also requires enabling ServiceAccount authentication (as front-proxy will start validating JWT tokens, which includes ServiceAccount tokens).
	// +kubebuilder:validation:XValidation:rule="!has(self.oidc) || (has(self.serviceAccount) && self.serviceAccount.enabled)",message="OIDC requires ServiceAccount auth to be enabled."
	Auth *AuthSpec `json:"auth,omitempty"`
	// Optional: AdditionalPathMappings configures // TODO ?
	AdditionalPathMappings []PathMappingEntry `json:"additionalPathMappings,omitempty"`
	// Optional: Image defines the image to use. Defaults to the latest versioned image during the release of kcp-operator.
	Image *ImageSpec `json:"image,omitempty"`
	// Optional: ExternalHostname under which the FrontProxy can be reached. If empty, the RootShard's external hostname will be used only.
	ExternalHostname string `json:"externalHostname,omitempty"`

	// Optional: ServiceTemplate configures the Kubernetes Service created for this front-proxy instance.
	ServiceTemplate *ServiceTemplate `json:"serviceTemplate,omitempty"`

	// Optional: DeploymentTemplate configures the Kubernetes Deployment created for this shard.
	DeploymentTemplate *DeploymentTemplate `json:"deploymentTemplate,omitempty"`

	// CertificateTemplates allows to customize the properties on the generated
	// certificates for this front-proxy.
	CertificateTemplates CertificateTemplateMap `json:"certificateTemplates,omitempty"`

	// CABundle references a v1.Secret object that contains the CA bundle
	// that should be used to validate the API server's TLS certificate.
	// The secret must contain a key named `tls.crt` that holds the PEM encoded CA certificate.
	// It will be merged into the "external-logical-cluster-admin-kubeconfig" kubeconfig under the `certificate-authority-data` field.
	// If not specified, the kubeconfig will use the CA bundle of the root shard or front-proxy referenced in the Target field.
	// It will NOT be used to configure the API server's own TLS certificate or any other component.
	// +optional
	CABundleSecretRef *corev1.LocalObjectReference `json:"caBundleSecretRef,omitempty"`

	// Optional: ExtraArgs defines additional command line arguments to pass to the front-proxy container.
	ExtraArgs []string `json:"extraArgs,omitempty"`
}

type AuthSpec struct {
	// Optional: OIDC configures OpenID Connect Authentication.
	OIDC *OIDCConfiguration `json:"oidc,omitempty"`

	// Optional: serviceAccountAuthentication configures ServiceAccount Authentication.
	ServiceAccount *ServiceAccountAuthentication `json:"serviceAccount,omitempty"`

	// Optional: DropGroups configures groups to be dropped before forwarding requests to Shards.
	DropGroups []string `json:"dropGroups,omitempty"`

	// Optional: PassOnGroups configures groups to be passed on before forwarding requests to Shards
	PassOnGroups []string `json:"passOnGroups,omitempty"`
}

// ServiceAccountAuthentication configures ServiceAccount Authentication.
type ServiceAccountAuthentication struct {
	// Optional: Enabled enables or disables ServiceAccount Authentication.
	// If set, it will mount every shard's service account certificate to the front-proxy.
	Enabled bool `json:"enabled"`
}

// FrontProxyStatus defines the observed state of FrontProxy
type FrontProxyStatus struct {
	Phase FrontProxyPhase `json:"phase,omitempty"`

	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type FrontProxyPhase string

const (
	FrontProxyPhaseProvisioning FrontProxyPhase = "Provisioning"
	FrontProxyPhaseRunning      FrontProxyPhase = "Running"
	FrontProxyPhaseDeleting     FrontProxyPhase = "Deleting"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=".spec.rootShard.ref.name",name="RootShard",type="string"
// +kubebuilder:printcolumn:JSONPath=".spec.externalHostname",name="ExternalHostname",type="string"
// +kubebuilder:printcolumn:JSONPath=".status.phase",name="Phase",type="string"
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// FrontProxy is the Schema for the frontproxies API
type FrontProxy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FrontProxySpec   `json:"spec,omitempty"`
	Status FrontProxyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FrontProxyList contains a list of FrontProxy
type FrontProxyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FrontProxy `json:"items"`
}

// TODO for now the PathMappingEntry is defined inline at kcp upstream (https://github.com/kcp-dev/kcp/blob/f81a97d0fba951e6ac6f94e8e0f5339f49a9dd92/cmd/sharded-test-server/frontproxy.go#L69),
// so we have to copy the struct type
type PathMappingEntry struct {
	Path            string `json:"path"`
	Backend         string `json:"backend"`
	BackendServerCA string `json:"backend_server_ca"`
	ProxyClientCert string `json:"proxy_client_cert"`
	ProxyClientKey  string `json:"proxy_client_key"`
}
