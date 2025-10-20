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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CacheServerSpec defines the desired state of CacheServer.
type CacheServerSpec struct {
	// Etcd configures the etcd cluster that this cache server should be using.
	Etcd EtcdConfig `json:"etcd"`

	// Optional: Image overwrites the container image used to deploy the cache server.
	Image *ImageSpec `json:"image,omitempty"`

	// Optional: Logging configures the logging settings for the cache server.
	Logging *LoggingSpec `json:"logging,omitempty"`
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
