/*
Copyright 2024 The kcp Authors.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CacheServerSpec defines the desired state of CacheServer.
type CacheServerSpec struct {
	// ClusterDomain is the DNS domain for services in the cluster. Defaults to "cluster.local" if not set.
	// +optional
	ClusterDomain string `json:"clusterDomain,omitempty"`

	// Optional: Image overwrites the container image used to deploy the cache server.
	Image *ImageSpec `json:"image,omitempty"`

	// Optional: Logging configures the logging settings for the cache server.
	Logging *LoggingSpec `json:"logging,omitempty"`

	// Certificates configures how the operator should create the kcp root CA, from which it will
	// then create all other sub CAs and leaf certificates.
	Certificates Certificates `json:"certificates"`

	// CertificateTemplates allows to customize the properties on the generated
	// certificates for this cache server.
	CertificateTemplates CertificateTemplateMap `json:"certificateTemplates,omitempty"`

	// Optional: ServiceTemplate configures the Kubernetes Service created for this cache server.
	ServiceTemplate *ServiceTemplate `json:"serviceTemplate,omitempty"`

	// Optional: DeploymentTemplate configures the Kubernetes Deployment created for this cache server.
	DeploymentTemplate *DeploymentTemplate `json:"deploymentTemplate,omitempty"`
}

// CacheServerStatus defines the observed state of CacheServer
type CacheServerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// CacheServer is the Schema for the cacheservers API
type CacheServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CacheServerSpec   `json:"spec,omitempty"`
	Status CacheServerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CacheServerList contains a list of CacheServer
type CacheServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CacheServer `json:"items"`
}
