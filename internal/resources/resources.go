/*
Copyright 2025 The kcp Authors.

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

	"github.com/Masterminds/semver/v3"

	corev1 "k8s.io/api/core/v1"

	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

const (
	// FrontProxyCommonName is the CommonName used in the requestheader client certificate for a FrontProxy.
	FrontProxyCommonName = "kcp-front-proxy"

	// RootShardProxyCommonName is the CommonName used in the requestheader client certificate for a RootShard's built-in proxy.
	RootShardProxyCommonName = "kcp-root-shard-proxy"

	// MountsProxyCommonName is the CommonName used in the client certificate that a shard's
	// local proxy presents when it re-enters the front-proxy to serve a mounted workspace.
	MountsProxyCommonName = "kcp-mounts-proxy"

	ImageRepository = "ghcr.io/kcp-dev/kcp"

	// ImageTag is the default tag to be used for any kcp component.
	//
	// When changing this to a new minor version, you must also update
	// the .prow.yaml accordingly and shift the jobs.
	ImageTag = "v0.32.3"

	appNameLabel      = "app.kubernetes.io/name"
	appInstanceLabel  = "app.kubernetes.io/instance"
	appManagedByLabel = "app.kubernetes.io/managed-by"
	appComponentLabel = "app.kubernetes.io/component"

	// RootShardLabel is placed on Secrets created for Certificates so that
	// the Secrets can be more easily mapped to their RootShards.
	RootShardLabel        = "operator.kcp.io/rootshard"
	ShardLabel            = "operator.kcp.io/shard"
	FrontProxyLabel       = "operator.kcp.io/front-proxy"
	KubeconfigLabel       = "operator.kcp.io/kubeconfig"
	CacheServerLabel      = "operator.kcp.io/cache-server"
	VirtualWorkspaceLabel = "operator.kcp.io/virtual-workspace"

	// BundleAnnotation is placed on RootShard, Shard, or FrontProxy objects to trigger automatic Bundle creation
	BundleAnnotation = "operator.kcp.io/bundle"

	// BundleDesiredReplicasAnnotation is placed on deployments to store the desired replica count before scaling to 0 for bundling
	BundleDesiredReplicasAnnotation = "operator.kcp.io/bundle-desired-replicas"

	// OperatorUsername is the common name embedded in the operator's admin certificate
	// that is created for each RootShard. This name alone has no special meaning, as
	// the certificate also has system:masters as an organization, which is what ultimately
	// grants the operator its permissions.
	OperatorUsername = "system:kcp-operator"

	defaultClusterDomain = "cluster.local"
)

func GetImageSettings(imageSpec *operatorv1alpha1.ImageSpec) (string, []corev1.LocalObjectReference, *semver.Version) {
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

	// try to detect the kcp version, but accept that this might not work for custom image tags
	version, _ := semver.NewVersion(tag)

	return fmt.Sprintf("%s:%s", repository, tag), imagePullSecrets, version
}

func GetRootShardDeploymentName(r *operatorv1alpha1.RootShard) string {
	return fmt.Sprintf("%s-kcp", r.Name)
}

func GetCompiledRootShardDeploymentName(r *deployv1alpha1.CompiledRootShard) string {
	return fmt.Sprintf("%s-kcp", r.Name)
}

func GetNamedRootShardDeploymentName(r deployv1alpha1.NamedRootShardSpec) string {
	return fmt.Sprintf("%s-kcp", r.Name)
}

func GetRootShardProxyDeploymentName(r *operatorv1alpha1.RootShard) string {
	return fmt.Sprintf("%s-proxy", r.Name)
}

func GetCompiledRootShardProxyDeploymentName(r *deployv1alpha1.CompiledRootShard) string {
	return fmt.Sprintf("%s-proxy", r.Name)
}

func GetNamedRootShardProxyDeploymentName(r deployv1alpha1.NamedRootShardSpec) string {
	return fmt.Sprintf("%s-proxy", r.Name)
}

func GetShardDeploymentName(s *operatorv1alpha1.Shard) string {
	return fmt.Sprintf("%s-shard-kcp", s.Name)
}

func GetCompiledShardDeploymentName(s *deployv1alpha1.CompiledShard) string {
	return fmt.Sprintf("%s-shard-kcp", s.Name)
}

func GetNamedShardDeploymentName(s deployv1alpha1.NamedShardSpec) string {
	return fmt.Sprintf("%s-shard-kcp", s.Name)
}

func GetCacheServerDeploymentName(s *operatorv1alpha1.CacheServer) string {
	return fmt.Sprintf("%s-cache-server", s.Name)
}

func GetCompiledCacheServerDeploymentName(s *deployv1alpha1.CompiledCacheServer) string {
	return fmt.Sprintf("%s-cache-server", s.Name)
}

func GetVirtualWorkspaceDeploymentName(vw *operatorv1alpha1.VirtualWorkspace) string {
	return fmt.Sprintf("%s-virtual-workspace", vw.Name)
}

func GetCompiledVirtualWorkspaceDeploymentName(vw *deployv1alpha1.CompiledVirtualWorkspace) string {
	return fmt.Sprintf("%s-virtual-workspace", vw.Name)
}

func GetNamedVirtualWorkspaceDeploymentName(vw deployv1alpha1.NamedVirtualWorkspaceSpec) string {
	return fmt.Sprintf("%s-virtual-workspace", vw.Name)
}

func GetRootShardServiceName(r *operatorv1alpha1.RootShard) string {
	return fmt.Sprintf("%s-kcp", r.Name)
}

func GetCompiledRootShardServiceName(r *deployv1alpha1.CompiledRootShard) string {
	return fmt.Sprintf("%s-kcp", r.Name)
}

func GetNamedRootShardServiceName(r deployv1alpha1.NamedRootShardSpec) string {
	return fmt.Sprintf("%s-kcp", r.Name)
}

func GetShardServiceName(s *operatorv1alpha1.Shard) string {
	return fmt.Sprintf("%s-shard-kcp", s.Name)
}

func GetCompiledShardServiceName(s *deployv1alpha1.CompiledShard) string {
	return fmt.Sprintf("%s-shard-kcp", s.Name)
}

func GetNamedShardServiceName(s deployv1alpha1.NamedShardSpec) string {
	return fmt.Sprintf("%s-shard-kcp", s.Name)
}

func GetCacheServerServiceName(s *operatorv1alpha1.CacheServer) string {
	return fmt.Sprintf("%s-cache-server", s.Name)
}

func GetCompiledCacheServerServiceName(s *deployv1alpha1.CompiledCacheServer) string {
	return fmt.Sprintf("%s-cache-server", s.Name)
}

func GetCompiledVirtualWorkspaceServiceName(vw *deployv1alpha1.CompiledVirtualWorkspace) string {
	return fmt.Sprintf("%s-virtual-workspace", vw.Name)
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

func GetCompiledRootShardResourceLabels(r *deployv1alpha1.CompiledRootShard) map[string]string {
	return getResourceLabels(r.Name, "rootshard")
}

func GetNamedRootShardResourceLabels(r deployv1alpha1.NamedRootShardSpec) map[string]string {
	return getResourceLabels(r.Name, "rootshard")
}

func GetRootShardProxyResourceLabels(r *operatorv1alpha1.RootShard) map[string]string {
	return getResourceLabels(r.Name, "rootshard-proxy")
}

func GetCompiledRootShardProxyResourceLabels(r *deployv1alpha1.CompiledRootShard) map[string]string {
	return getResourceLabels(r.Name, "rootshard-proxy")
}

func GetNamedRootShardProxyResourceLabels(r deployv1alpha1.NamedRootShardSpec) map[string]string {
	return getResourceLabels(r.Name, "rootshard-proxy")
}

func GetShardResourceLabels(s *operatorv1alpha1.Shard) map[string]string {
	return getResourceLabels(s.Name, "shard")
}

func GetCompiledShardResourceLabels(s *deployv1alpha1.CompiledShard) map[string]string {
	return getResourceLabels(s.Name, "shard")
}

func GetNamedShardResourceLabels(s deployv1alpha1.NamedShardSpec) map[string]string {
	return getResourceLabels(s.Name, "shard")
}

func GetCacheServerResourceLabels(s *operatorv1alpha1.CacheServer) map[string]string {
	return getResourceLabels(s.Name, "cache-server")
}

func GetCompiledCacheServerResourceLabels(s *deployv1alpha1.CompiledCacheServer) map[string]string {
	return getResourceLabels(s.Name, "cache-server")
}

func GetVirtualWorkspaceResourceLabels(vw *operatorv1alpha1.VirtualWorkspace) map[string]string {
	return getResourceLabels(vw.Name, "virtual-workspace")
}

func GetCompiledVirtualWorkspaceResourceLabels(vw *deployv1alpha1.CompiledVirtualWorkspace) map[string]string {
	return getResourceLabels(vw.Name, "virtual-workspace")
}

func GetNamedVirtualWorkspaceResourceLabels(vw deployv1alpha1.NamedVirtualWorkspaceSpec) map[string]string {
	return getResourceLabels(vw.Name, "virtual-workspace")
}

func GetRootShardBaseHost(r *operatorv1alpha1.RootShard) string {
	clusterDomain := r.Spec.ClusterDomain
	if clusterDomain == "" {
		clusterDomain = defaultClusterDomain
	}

	return fmt.Sprintf("%s-kcp.%s.svc.%s", r.Name, r.Namespace, clusterDomain)
}

func GetCompiledRootShardBaseHost(r *deployv1alpha1.CompiledRootShard) string {
	clusterDomain := r.Spec.RootShard.ClusterDomain
	if clusterDomain == "" {
		clusterDomain = defaultClusterDomain
	}

	return fmt.Sprintf("%s-kcp.%s.svc.%s", r.Name, r.Namespace, clusterDomain)
}

func GetNamedRootShardBaseHost(r deployv1alpha1.NamedRootShardSpec, namespace string) string {
	clusterDomain := r.Spec.ClusterDomain
	if clusterDomain == "" {
		clusterDomain = defaultClusterDomain
	}

	return fmt.Sprintf("%s-kcp.%s.svc.%s", r.Name, namespace, clusterDomain)
}

func GetRootShardBaseURL(r *operatorv1alpha1.RootShard) string {
	if r.Spec.ShardBaseURL != "" {
		return r.Spec.ShardBaseURL
	}
	return fmt.Sprintf("https://%s:6443", GetRootShardBaseHost(r))
}

func GetCompiledRootShardBaseURL(r *deployv1alpha1.CompiledRootShard) string {
	if r.Spec.RootShard.ShardBaseURL != "" {
		return r.Spec.RootShard.ShardBaseURL
	}
	return fmt.Sprintf("https://%s:6443", GetCompiledRootShardBaseHost(r))
}

func GetNamedRootShardBaseURL(r deployv1alpha1.NamedRootShardSpec, namespace string) string {
	if r.Spec.ShardBaseURL != "" {
		return r.Spec.ShardBaseURL
	}
	return fmt.Sprintf("https://%s:6443", GetNamedRootShardBaseHost(r, namespace))
}

func GetShardBaseHost(s *operatorv1alpha1.Shard) string {
	clusterDomain := s.Spec.ClusterDomain
	if clusterDomain == "" {
		clusterDomain = defaultClusterDomain
	}

	return fmt.Sprintf("%s-shard-kcp.%s.svc.%s", s.Name, s.Namespace, clusterDomain)
}

func GetCompiledShardBaseHost(s *deployv1alpha1.CompiledShard) string {
	clusterDomain := s.Spec.Shard.ClusterDomain
	if clusterDomain == "" {
		clusterDomain = defaultClusterDomain
	}

	return fmt.Sprintf("%s-shard-kcp.%s.svc.%s", s.Name, s.Namespace, clusterDomain)
}

func GetNamedShardBaseHost(s deployv1alpha1.NamedShardSpec, namespace string) string {
	clusterDomain := s.Spec.ClusterDomain
	if clusterDomain == "" {
		clusterDomain = defaultClusterDomain
	}

	return fmt.Sprintf("%s-shard-kcp.%s.svc.%s", s.Name, namespace, clusterDomain)
}

func GetShardBaseURL(s *operatorv1alpha1.Shard) string {
	if s.Spec.ShardBaseURL != "" {
		return s.Spec.ShardBaseURL
	}
	return fmt.Sprintf("https://%s:6443", GetShardBaseHost(s))
}

func GetCompiledShardBaseURL(s *deployv1alpha1.CompiledShard) string {
	if s.Spec.Shard.ShardBaseURL != "" {
		return s.Spec.Shard.ShardBaseURL
	}
	return fmt.Sprintf("https://%s:6443", GetCompiledShardBaseHost(s))
}

func GetNamedShardBaseURL(s deployv1alpha1.NamedShardSpec, namespace string) string {
	if s.Spec.ShardBaseURL != "" {
		return s.Spec.ShardBaseURL
	}
	return fmt.Sprintf("https://%s:6443", GetNamedShardBaseHost(s, namespace))
}

func GetCacheServerBaseHost(s *operatorv1alpha1.CacheServer) string {
	clusterDomain := s.Spec.ClusterDomain
	if clusterDomain == "" {
		clusterDomain = defaultClusterDomain
	}

	return fmt.Sprintf("%s-cache-server.%s.svc.%s", s.Name, s.Namespace, clusterDomain)
}

func GetCompiledCacheServerBaseHost(s *deployv1alpha1.CompiledCacheServer) string {
	clusterDomain := s.Spec.CacheServer.ClusterDomain
	if clusterDomain == "" {
		clusterDomain = defaultClusterDomain
	}

	return fmt.Sprintf("%s-cache-server.%s.svc.%s", s.Name, s.Namespace, clusterDomain)
}

func GetCacheServerBaseURL(s *operatorv1alpha1.CacheServer) string {
	return fmt.Sprintf("https://%s:6443", GetCacheServerBaseHost(s))
}

func GetCompiledCacheServerBaseURL(s *deployv1alpha1.CompiledCacheServer) string {
	return fmt.Sprintf("https://%s:6443", GetCompiledCacheServerBaseHost(s))
}

func GetVirtualWorkspaceBaseHost(s *operatorv1alpha1.VirtualWorkspace) string {
	clusterDomain := s.Spec.ClusterDomain
	if clusterDomain == "" {
		clusterDomain = defaultClusterDomain
	}

	return fmt.Sprintf("%s-virtual-workspace.%s.svc.%s", s.Name, s.Namespace, clusterDomain)
}

func GetCompiledVirtualWorkspaceBaseHost(s *deployv1alpha1.CompiledVirtualWorkspace) string {
	clusterDomain := s.Spec.VirtualWorkspace.ClusterDomain
	if clusterDomain == "" {
		clusterDomain = defaultClusterDomain
	}

	return fmt.Sprintf("%s-virtual-workspace.%s.svc.%s", s.Name, s.Namespace, clusterDomain)
}

func GetNamedVirtualWorkspaceBaseHost(vw deployv1alpha1.NamedVirtualWorkspaceSpec, namespace string) string {
	clusterDomain := vw.Spec.ClusterDomain
	if clusterDomain == "" {
		clusterDomain = defaultClusterDomain
	}

	return fmt.Sprintf("%s-virtual-workspace.%s.svc.%s", vw.Name, namespace, clusterDomain)
}

func GetVirtualWorkspaceBaseURL(s *operatorv1alpha1.VirtualWorkspace) string {
	return fmt.Sprintf("https://%s:6443", GetVirtualWorkspaceBaseHost(s))
}

func GetCompiledVirtualWorkspaceBaseURL(s *deployv1alpha1.CompiledVirtualWorkspace) string {
	return fmt.Sprintf("https://%s:6443", GetCompiledVirtualWorkspaceBaseHost(s))
}

func GetNamedVirtualWorkspaceBaseURL(vw deployv1alpha1.NamedVirtualWorkspaceSpec, namespace string) string {
	return fmt.Sprintf("https://%s:6443", GetNamedVirtualWorkspaceBaseHost(vw, namespace))
}

func GetRootShardCertificateName(r *operatorv1alpha1.RootShard, certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s", r.Name, certName)
}

func GetCompiledRootShardCertificateName(r *deployv1alpha1.CompiledRootShard, certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s", r.Name, certName)
}

func GetNamedRootShardCertificateName(r deployv1alpha1.NamedRootShardSpec, certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s", r.Name, certName)
}

func GetRootShardProxyCertificateName(r *operatorv1alpha1.RootShard, certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-proxy-%s", r.Name, certName)
}

func GetCompiledRootShardProxyCertificateName(r *deployv1alpha1.CompiledRootShard, certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-proxy-%s", r.Name, certName)
}

func GetNamedRootShardProxyCertificateName(r deployv1alpha1.NamedRootShardSpec, certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-proxy-%s", r.Name, certName)
}

func GetShardCertificateName(s *operatorv1alpha1.Shard, certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s", s.Name, certName)
}

func GetCompiledShardCertificateName(s *deployv1alpha1.CompiledShard, certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s", s.Name, certName)
}

func GetNamedShardCertificateName(s deployv1alpha1.NamedShardSpec, certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s", s.Name, certName)
}

func GetCacheServerCertificateName(s *operatorv1alpha1.CacheServer, certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s", s.Name, certName)
}

func GetCompiledCacheServerCertificateName(s *deployv1alpha1.CompiledCacheServer, certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s", s.Name, certName)
}

func GetVirtualWorkspaceCertificateName(vw *operatorv1alpha1.VirtualWorkspace, certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s", vw.Name, certName)
}

func GetCompiledVirtualWorkspaceCertificateName(vw *deployv1alpha1.CompiledVirtualWorkspace, certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s", vw.Name, certName)
}

func GetNamedVirtualWorkspaceCertificateName(vw deployv1alpha1.NamedVirtualWorkspaceSpec, certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s", vw.Name, certName)
}

func GetRootShardCAName(r *operatorv1alpha1.RootShard, caName operatorv1alpha1.CA) string {
	if caName == operatorv1alpha1.RootCA {
		return fmt.Sprintf("%s-ca", r.Name)
	}
	return fmt.Sprintf("%s-%s-ca", r.Name, caName)
}

func GetCompiledRootShardCAName(r *deployv1alpha1.CompiledRootShard, caName operatorv1alpha1.CA) string {
	if caName == operatorv1alpha1.RootCA {
		return fmt.Sprintf("%s-ca", r.Name)
	}
	return fmt.Sprintf("%s-%s-ca", r.Name, caName)
}

func GetNamedRootShardCAName(r deployv1alpha1.NamedRootShardSpec, caName operatorv1alpha1.CA) string {
	if caName == operatorv1alpha1.RootCA {
		return fmt.Sprintf("%s-ca", r.Name)
	}
	return fmt.Sprintf("%s-%s-ca", r.Name, caName)
}

func GetCacheServerCAName(cacheServerName string, caName operatorv1alpha1.CA) string {
	if caName == operatorv1alpha1.RootCA {
		return fmt.Sprintf("%s-ca", cacheServerName)
	}
	return fmt.Sprintf("%s-%s-ca", cacheServerName, caName)
}

func GetFrontProxyResourceLabels(f *operatorv1alpha1.FrontProxy) map[string]string {
	return getResourceLabels(f.Name, "front-proxy")
}

func GetCompiledFrontProxyResourceLabels(f *deployv1alpha1.CompiledFrontProxy) map[string]string {
	return getResourceLabels(f.Name, "front-proxy")
}

func GetFrontProxyDeploymentName(f *operatorv1alpha1.FrontProxy) string {
	return fmt.Sprintf("%s-front-proxy", f.Name)
}

func GetCompiledFrontProxyDeploymentName(f *deployv1alpha1.CompiledFrontProxy) string {
	return fmt.Sprintf("%s-front-proxy", f.Name)
}

func GetFrontProxyCertificateName(r *operatorv1alpha1.RootShard, f *operatorv1alpha1.FrontProxy, certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s-%s", r.Name, f.Name, certName)
}

func GetCompiledFrontProxyCertificateName(f *deployv1alpha1.CompiledFrontProxy, certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s-%s", f.Spec.RootShard.Name, f.Name, certName)
}

func GetRootShardProxyDynamicKubeconfigName(r *operatorv1alpha1.RootShard) string {
	return fmt.Sprintf("%s-proxy-dynamic-kubeconfig", r.Name)
}

func GetCompiledRootShardProxyDynamicKubeconfigName(r *deployv1alpha1.CompiledRootShard) string {
	return fmt.Sprintf("%s-proxy-dynamic-kubeconfig", r.Name)
}

func GetNamedRootShardProxyDynamicKubeconfigName(r deployv1alpha1.NamedRootShardSpec) string {
	return fmt.Sprintf("%s-proxy-dynamic-kubeconfig", r.Name)
}

func GetFrontProxyDynamicKubeconfigName(r *operatorv1alpha1.RootShard, f *operatorv1alpha1.FrontProxy) string {
	return fmt.Sprintf("%s-%s-dynamic-kubeconfig", r.Name, f.Name)
}

func GetCompiledFrontProxyDynamicKubeconfigName(f *deployv1alpha1.CompiledFrontProxy) string {
	return fmt.Sprintf("%s-%s-dynamic-kubeconfig", f.Spec.RootShard.Name, f.Name)
}

func GetCacheServerClientCertificateName(s *operatorv1alpha1.CacheServer) string {
	return fmt.Sprintf("%s-client-certificate", s.Name)
}

func GetCompiledCacheServerClientCertificateName(s *deployv1alpha1.CompiledCacheServer) string {
	return fmt.Sprintf("%s-client-certificate", s.Name)
}

func GetCacheServerKubeconfigName(cacheServerName string) string {
	return fmt.Sprintf("%s-kubeconfig", cacheServerName)
}

func GetRootShardProxyConfigName(r *operatorv1alpha1.RootShard) string {
	return fmt.Sprintf("%s-proxy-config", r.Name)
}

func GetCompiledRootShardProxyConfigName(r *deployv1alpha1.CompiledRootShard) string {
	return fmt.Sprintf("%s-proxy-config", r.Name)
}

func GetNamedRootShardProxyConfigName(r deployv1alpha1.NamedRootShardSpec) string {
	return fmt.Sprintf("%s-proxy-config", r.Name)
}

func GetFrontProxyConfigName(f *operatorv1alpha1.FrontProxy) string {
	return fmt.Sprintf("%s-config", f.Name)
}

func GetCompiledFrontProxyConfigName(f *deployv1alpha1.CompiledFrontProxy) string {
	return fmt.Sprintf("%s-config", f.Name)
}

func GetFrontProxyServiceName(f *operatorv1alpha1.FrontProxy) string {
	return fmt.Sprintf("%s-front-proxy", f.Name)
}

func GetCompiledFrontProxyServiceName(f *deployv1alpha1.CompiledFrontProxy) string {
	return fmt.Sprintf("%s-front-proxy", f.Name)
}

func GetRootShardProxyServiceName(r *operatorv1alpha1.RootShard) string {
	return fmt.Sprintf("%s-proxy", r.Name)
}

func GetCompiledRootShardProxyServiceName(r *deployv1alpha1.CompiledRootShard) string {
	return fmt.Sprintf("%s-proxy", r.Name)
}

func GetNamedRootShardProxyServiceName(r deployv1alpha1.NamedRootShardSpec) string {
	return fmt.Sprintf("%s-proxy", r.Name)
}

func GetRootShardKubeconfigSecret(r *operatorv1alpha1.RootShard, cert operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s-kubeconfig", r.Name, cert)
}

func GetCompiledRootShardKubeconfigSecret(r *deployv1alpha1.CompiledRootShard, cert operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s-kubeconfig", r.Name, cert)
}

func GetNamedRootShardKubeconfigSecret(r deployv1alpha1.NamedRootShardSpec, cert operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s-kubeconfig", r.Name, cert)
}

func GetShardKubeconfigSecret(shard *operatorv1alpha1.Shard, cert operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s-kubeconfig", shard.Name, cert)
}

func GetCompiledShardKubeconfigSecret(shard *deployv1alpha1.CompiledShard, cert operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s-kubeconfig", shard.Name, cert)
}

func GetNamedShardKubeconfigSecret(shard deployv1alpha1.NamedShardSpec, cert operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s-kubeconfig", shard.Name, cert)
}

func GetBundleName(ownerName string) string {
	return fmt.Sprintf("%s-bundle", ownerName)
}

func GetMergedClientCAName(ownerName string) string {
	return fmt.Sprintf("%s-merged-client-ca", ownerName)
}
