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
	"fmt"

	"github.com/kcp-dev/logicalcluster/v3"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubeconfigSpec defines the desired state of Kubeconfig.
// +kubebuilder:validation:XValidation:rule="!(has(self.targetWorkspace) && has(self.authorization) && has(self.authorization.clusterRoleBindings.cluster))",message="Cannot set both targetWorkspace and authorization.clusterRoleBindings.cluster. Use targetWorkspace only."
type KubeconfigSpec struct {
	// Target configures which kcp-operator object this kubeconfig should be generated for (shard or front-proxy).
	Target KubeconfigTarget `json:"target"`

	// TargetWorkspace specifies the workspace path this kubeconfig targets.
	// Used in the generated kubeconfig server URL and as the default RBAC provisioning target.
	// Accepts kcp workspace paths like "root:org:team".
	// Defaults to "root" if unset.
	// +optional
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(:[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*$`
	TargetWorkspace string `json:"targetWorkspace,omitempty"`

	// Username defines the username embedded in the TLS certificate generated for this kubeconfig.
	Username string `json:"username"`
	// Username defines the groups embedded in the TLS certificate generated for this kubeconfig.
	Groups []string `json:"groups,omitempty"`

	// Validity configures the lifetime of the embedded TLS certificate. The kubeconfig secret will be automatically regenerated when the certificate expires.
	Validity metav1.Duration `json:"validity"`

	// SecretRef defines the v1.Secret object that the resulting kubeconfig should be written to.
	SecretRef corev1.LocalObjectReference `json:"secretRef"`

	// CertificateTemplate allows to customize the properties on the generated
	// certificate for this kubeconfig.
	CertificateTemplate *CertificateTemplate `json:"certificateTemplate,omitempty"`

	// Authorization allows to provision permissions for this kubeconfig.
	Authorization *KubeconfigAuthorization `json:"authorization,omitempty"`
}

type KubeconfigTarget struct {
	RootShardRef  *corev1.LocalObjectReference `json:"rootShardRef,omitempty"`
	ShardRef      *corev1.LocalObjectReference `json:"shardRef,omitempty"`
	FrontProxyRef *corev1.LocalObjectReference `json:"frontProxyRef,omitempty"`
}

type KubeconfigAuthorization struct {
	ClusterRoleBindings KubeconfigClusterRoleBindings `json:"clusterRoleBindings"`
}

type KubeconfigClusterRoleBindings struct {
	// Cluster can be either a cluster name or a workspace path.
	//
	// Deprecated: Use spec.targetWorkspace instead. This field is kept for backward
	// compatibility but cannot be set together with spec.targetWorkspace.
	// +optional
	Cluster string `json:"cluster,omitempty"`

	ClusterRoles []string `json:"clusterRoles"`
}

type KubeconfigPhase string

const (
	KubeconfigPhaseProvisioning KubeconfigPhase = "Provisioning"
	KubeconfigPhaseReady        KubeconfigPhase = "Ready"
	KubeconfigPhaseFailed       KubeconfigPhase = "Failed"
)

// KubeconfigStatus defines the observed state of Kubeconfig
type KubeconfigStatus struct {
	// Phase represents the current phase of kubeconfig lifecycle.
	Phase KubeconfigPhase `json:"phase,omitempty"`

	// TargetName represents the name of the target resource (RootShard, Shard, or FrontProxy).
	TargetName string `json:"targetName,omitempty"`

	Authorization *KubeconfigAuthorizationStatus `json:"authorization,omitempty"`

	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type KubeconfigAuthorizationStatus struct {
	ProvisionedCluster string `json:"provisionedCluster"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=".status.targetName",name="Target",type="string"
// +kubebuilder:printcolumn:JSONPath=".status.phase",name="Phase",type="string"
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"
// Kubeconfig is the Schema for the kubeconfigs API
type Kubeconfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KubeconfigSpec   `json:"spec,omitempty"`
	Status KubeconfigStatus `json:"status,omitempty"`
}

func (k *Kubeconfig) GetCertificateName() string {
	return fmt.Sprintf("kubeconfig-cert-%s", k.Name)
}

// GetTargetWorkspace returns the workspace path for the generated kubeconfig URL.
// Only spec.targetWorkspace is considered; defaults to "root".
// The deprecated authorization.clusterRoleBindings.cluster field does NOT influence
// the URL to avoid breaking existing users.
func (k *Kubeconfig) GetTargetWorkspace() logicalcluster.Path {
	if k.Spec.TargetWorkspace != "" {
		return logicalcluster.NewPath(k.Spec.TargetWorkspace)
	}
	return logicalcluster.NewPath("root")
}

// GetRBACTargetWorkspace returns the workspace path for RBAC provisioning.
// It checks spec.targetWorkspace first, then falls back to
// spec.authorization.clusterRoleBindings.cluster (deprecated), and defaults to "root".
func (k *Kubeconfig) GetRBACTargetWorkspace() logicalcluster.Path {
	if k.Spec.TargetWorkspace != "" {
		return logicalcluster.NewPath(k.Spec.TargetWorkspace)
	}
	if auth := k.Spec.Authorization; auth != nil && auth.ClusterRoleBindings.Cluster != "" {
		return logicalcluster.NewPath(auth.ClusterRoleBindings.Cluster)
	}
	return logicalcluster.NewPath("root")
}

// +kubebuilder:object:root=true

// KubeconfigList contains a list of Kubeconfig
type KubeconfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kubeconfig `json:"items"`
}
