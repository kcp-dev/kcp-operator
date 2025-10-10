/*
Copyright 2025 The KCP Authors.

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

package resources

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

const (
	ImageRepository = "ghcr.io/kcp-dev/kcp"
	ImageTag        = "v0.28.3"

	appNameLabel      = "app.kubernetes.io/name"
	appInstanceLabel  = "app.kubernetes.io/instance"
	appManagedByLabel = "app.kubernetes.io/managed-by"
	appComponentLabel = "app.kubernetes.io/component"

	// RootShardLabel is placed on Secrets created for Certificates so that
	// the Secrets can be more easily mapped to their RootShards.
	RootShardLabel  = "operator.kcp.io/rootshard"
	ShardLabel      = "operator.kcp.io/shard"
	FrontProxyLabel = "operator.kcp.io/front-proxy"
	KubeconfigLabel = "operator.kcp.io/kubeconfig"

	// OperatorUsername is the common name embedded in the operator's admin certificate
	// that is created for each RootShard. This name alone has no special meaning, as
	// the certificate also has system:masters as an organization, which is what ultimately
	// grants the operator its permissions.
	OperatorUsername = "system:kcp-operator"
)

func GetImageSettings(imageSpec *operatorv1alpha1.ImageSpec) (string, []corev1.LocalObjectReference) {
	repository := ImageRepository
	if imageSpec != nil && imageSpec.Repository != "" {
		repository = imageSpec.Repository
	}

	tag := ImageTag
	if imageSpec != nil && imageSpec.Tag != "" {
		tag = imageSpec.Tag
	}

	imagePullSecrets := []corev1.LocalObjectReference{}
	if imageSpec != nil && len(imageSpec.ImagePullSecrets) > 0 {
		imagePullSecrets = imageSpec.ImagePullSecrets
	}

	return fmt.Sprintf("%s:%s", repository, tag), imagePullSecrets
}

func GetRootShardDeploymentName(r *operatorv1alpha1.RootShard) string {
	return fmt.Sprintf("%s-kcp", r.Name)
}

func GetRootShardProxyDeploymentName(r *operatorv1alpha1.RootShard) string {
	return fmt.Sprintf("%s-proxy", r.Name)
}

func GetShardDeploymentName(s *operatorv1alpha1.Shard) string {
	return fmt.Sprintf("%s-shard-kcp", s.Name)
}

func GetRootShardServiceName(r *operatorv1alpha1.RootShard) string {
	return fmt.Sprintf("%s-kcp", r.Name)
}

func GetShardServiceName(s *operatorv1alpha1.Shard) string {
	return fmt.Sprintf("%s-shard-kcp", s.Name)
}

func getResourceLabels(instance, component string) map[string]string {
	return map[string]string{
		appManagedByLabel: "kcp-operator",
		appNameLabel:      "kcp",
		appInstanceLabel:  instance,
		appComponentLabel: component,
	}
}

func GetRootShardResourceLabels(r *operatorv1alpha1.RootShard) map[string]string {
	return getResourceLabels(r.Name, "rootshard")
}

func GetRootShardProxyResourceLabels(r *operatorv1alpha1.RootShard) map[string]string {
	return getResourceLabels(r.Name, "rootshard-proxy")
}

func GetShardResourceLabels(s *operatorv1alpha1.Shard) map[string]string {
	return getResourceLabels(s.Name, "shard")
}

func GetRootShardBaseHost(r *operatorv1alpha1.RootShard) string {
	clusterDomain := r.Spec.ClusterDomain
	if clusterDomain == "" {
		clusterDomain = "cluster.local"
	}

	return fmt.Sprintf("%s-kcp.%s.svc.%s", r.Name, r.Namespace, clusterDomain)
}

func GetRootShardBaseURL(r *operatorv1alpha1.RootShard) string {
	if r.Spec.ShardBaseURL != "" {
		return r.Spec.ShardBaseURL
	}
	return fmt.Sprintf("https://%s:6443", GetRootShardBaseHost(r))
}

func GetShardBaseHost(s *operatorv1alpha1.Shard) string {
	clusterDomain := s.Spec.ClusterDomain
	if clusterDomain == "" {
		clusterDomain = "cluster.local"
	}

	return fmt.Sprintf("%s-shard-kcp.%s.svc.%s", s.Name, s.Namespace, clusterDomain)
}

func GetShardBaseURL(s *operatorv1alpha1.Shard) string {
	if s.Spec.ShardBaseURL != "" {
		return s.Spec.ShardBaseURL
	}
	return fmt.Sprintf("https://%s:6443", GetShardBaseHost(s))
}

func GetRootShardCertificateName(r *operatorv1alpha1.RootShard, certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s", r.Name, certName)
}

func GetRootShardProxyCertificateName(r *operatorv1alpha1.RootShard, certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-proxy-%s", r.Name, certName)
}

func GetShardCertificateName(s *operatorv1alpha1.Shard, certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s", s.Name, certName)
}

func GetRootShardCAName(r *operatorv1alpha1.RootShard, caName operatorv1alpha1.CA) string {
	if caName == operatorv1alpha1.RootCA {
		return fmt.Sprintf("%s-ca", r.Name)
	}
	return fmt.Sprintf("%s-%s-ca", r.Name, caName)
}

func GetFrontProxyResourceLabels(f *operatorv1alpha1.FrontProxy) map[string]string {
	return getResourceLabels(f.Name, "front-proxy")
}

func GetFrontProxyDeploymentName(f *operatorv1alpha1.FrontProxy) string {
	return fmt.Sprintf("%s-front-proxy", f.Name)
}

func GetFrontProxyCertificateName(r *operatorv1alpha1.RootShard, f *operatorv1alpha1.FrontProxy, certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s-%s", r.Name, f.Name, certName)
}

func GetRootShardProxyDynamicKubeconfigName(r *operatorv1alpha1.RootShard) string {
	return fmt.Sprintf("%s-proxy-dynamic-kubeconfig", r.Name)
}

func GetFrontProxyDynamicKubeconfigName(r *operatorv1alpha1.RootShard, f *operatorv1alpha1.FrontProxy) string {
	return fmt.Sprintf("%s-%s-dynamic-kubeconfig", r.Name, f.Name)
}

func GetRootShardProxyConfigName(r *operatorv1alpha1.RootShard) string {
	return fmt.Sprintf("%s-proxy-config", r.Name)
}

func GetFrontProxyConfigName(f *operatorv1alpha1.FrontProxy) string {
	return fmt.Sprintf("%s-config", f.Name)
}

func GetFrontProxyServiceName(f *operatorv1alpha1.FrontProxy) string {
	return fmt.Sprintf("%s-front-proxy", f.Name)
}

func GetRootShardProxyServiceName(r *operatorv1alpha1.RootShard) string {
	return fmt.Sprintf("%s-proxy", r.Name)
}
