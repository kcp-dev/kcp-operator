/*
Copyright 2026 The kcp Authors.

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

package naming

import (
	"fmt"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

const (
	appNameLabel      = "app.kubernetes.io/name"
	appInstanceLabel  = "app.kubernetes.io/instance"
	appManagedByLabel = "app.kubernetes.io/managed-by"
	appComponentLabel = "app.kubernetes.io/component"

	defaultClusterDomain = "cluster.local"
)

type version1 struct{}

func NewVersion1() Scheme {
	return &version1{}
}

func (v *version1) getResourceLabels(instance, component string) map[string]string {
	return map[string]string{
		appManagedByLabel: "kcp-operator",
		appNameLabel:      "kcp",
		appInstanceLabel:  instance,
		appComponentLabel: component,
	}
}

// RootShard naming

func (v *version1) GetRootShardDeploymentName(r *operatorv1alpha1.RootShard) string {
	return fmt.Sprintf("%s-kcp", r.Name)
}

func (v *version1) GetRootShardProxyDeploymentName(r *operatorv1alpha1.RootShard) string {
	return fmt.Sprintf("%s-proxy", r.Name)
}

func (v *version1) GetRootShardServiceName(r *operatorv1alpha1.RootShard) string {
	return fmt.Sprintf("%s-kcp", r.Name)
}

func (v *version1) GetRootShardResourceLabels(r *operatorv1alpha1.RootShard) map[string]string {
	return v.getResourceLabels(r.Name, "rootshard")
}

func (v *version1) GetRootShardProxyResourceLabels(r *operatorv1alpha1.RootShard) map[string]string {
	return v.getResourceLabels(r.Name, "rootshard-proxy")
}

func (v *version1) GetRootShardBaseHost(r *operatorv1alpha1.RootShard) string {
	clusterDomain := r.Spec.ClusterDomain
	if clusterDomain == "" {
		clusterDomain = defaultClusterDomain
	}

	return fmt.Sprintf("%s-kcp.%s.svc.%s", r.Name, r.Namespace, clusterDomain)
}

func (v *version1) GetRootShardBaseURL(r *operatorv1alpha1.RootShard) string {
	if r.Spec.ShardBaseURL != "" {
		return r.Spec.ShardBaseURL
	}
	return fmt.Sprintf("https://%s:6443", v.GetRootShardBaseHost(r))
}

func (v *version1) GetRootShardCertificateName(r *operatorv1alpha1.RootShard, certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s", r.Name, certName)
}

func (v *version1) GetRootShardProxyCertificateName(r *operatorv1alpha1.RootShard, certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-proxy-%s", r.Name, certName)
}

func (v *version1) GetRootShardCAName(r *operatorv1alpha1.RootShard, caName operatorv1alpha1.CA) string {
	if caName == operatorv1alpha1.RootCA {
		return fmt.Sprintf("%s-ca", r.Name)
	}
	return fmt.Sprintf("%s-%s-ca", r.Name, caName)
}

func (v *version1) GetRootShardProxyDynamicKubeconfigName(r *operatorv1alpha1.RootShard) string {
	return fmt.Sprintf("%s-proxy-dynamic-kubeconfig", r.Name)
}

func (v *version1) GetRootShardProxyConfigName(r *operatorv1alpha1.RootShard) string {
	return fmt.Sprintf("%s-proxy-config", r.Name)
}

func (v *version1) GetRootShardProxyServiceName(r *operatorv1alpha1.RootShard) string {
	return fmt.Sprintf("%s-proxy", r.Name)
}

func (v *version1) GetRootShardKubeconfigSecret(r *operatorv1alpha1.RootShard, cert operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s-kubeconfig", r.Name, cert)
}

// Shard naming

func (v *version1) GetShardDeploymentName(s *operatorv1alpha1.Shard) string {
	return fmt.Sprintf("%s-shard-kcp", s.Name)
}

func (v *version1) GetShardServiceName(s *operatorv1alpha1.Shard) string {
	return fmt.Sprintf("%s-shard-kcp", s.Name)
}

func (v *version1) GetShardResourceLabels(s *operatorv1alpha1.Shard) map[string]string {
	return v.getResourceLabels(s.Name, "shard")
}

func (v *version1) GetShardBaseHost(s *operatorv1alpha1.Shard) string {
	clusterDomain := s.Spec.ClusterDomain
	if clusterDomain == "" {
		clusterDomain = defaultClusterDomain
	}

	return fmt.Sprintf("%s-shard-kcp.%s.svc.%s", s.Name, s.Namespace, clusterDomain)
}

func (v *version1) GetShardBaseURL(s *operatorv1alpha1.Shard) string {
	if s.Spec.ShardBaseURL != "" {
		return s.Spec.ShardBaseURL
	}
	return fmt.Sprintf("https://%s:6443", v.GetShardBaseHost(s))
}

func (v *version1) GetShardCertificateName(s *operatorv1alpha1.Shard, certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s", s.Name, certName)
}

func (v *version1) GetShardKubeconfigSecret(shard *operatorv1alpha1.Shard, cert operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s-kubeconfig", shard.Name, cert)
}

// CacheServer naming

func (v *version1) GetCacheServerDeploymentName(s *operatorv1alpha1.CacheServer) string {
	return fmt.Sprintf("%s-cache-server", s.Name)
}

func (v *version1) GetCacheServerServiceName(s *operatorv1alpha1.CacheServer) string {
	return fmt.Sprintf("%s-cache-server", s.Name)
}

func (v *version1) GetCacheServerResourceLabels(s *operatorv1alpha1.CacheServer) map[string]string {
	return v.getResourceLabels(s.Name, "cache-server")
}

func (v *version1) GetCacheServerBaseHost(s *operatorv1alpha1.CacheServer) string {
	clusterDomain := s.Spec.ClusterDomain
	if clusterDomain == "" {
		clusterDomain = defaultClusterDomain
	}

	return fmt.Sprintf("%s-cache-server.%s.svc.%s", s.Name, s.Namespace, clusterDomain)
}

func (v *version1) GetCacheServerBaseURL(s *operatorv1alpha1.CacheServer) string {
	return fmt.Sprintf("https://%s:6443", v.GetCacheServerBaseHost(s))
}

func (v *version1) GetCacheServerCertificateName(s *operatorv1alpha1.CacheServer, certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s", s.Name, certName)
}

func (v *version1) GetCacheServerCAName(cacheServerName string, caName operatorv1alpha1.CA) string {
	if caName == operatorv1alpha1.RootCA {
		return fmt.Sprintf("%s-ca", cacheServerName)
	}
	return fmt.Sprintf("%s-%s-ca", cacheServerName, caName)
}

func (v *version1) GetCacheServerClientCertificateName(s *operatorv1alpha1.CacheServer) string {
	return fmt.Sprintf("%s-client-certificate", s.Name)
}

func (v *version1) GetCacheServerKubeconfigName(cacheServerName string) string {
	return fmt.Sprintf("%s-kubeconfig", cacheServerName)
}

// VirtualWorkspace naming

func (v *version1) GetVirtualWorkspaceDeploymentName(vw *operatorv1alpha1.VirtualWorkspace) string {
	return fmt.Sprintf("%s-virtual-workspace", vw.Name)
}

func (v *version1) GetVirtualWorkspaceResourceLabels(vw *operatorv1alpha1.VirtualWorkspace) map[string]string {
	return v.getResourceLabels(vw.Name, "virtual-workspace")
}

func (v *version1) GetVirtualWorkspaceBaseHost(s *operatorv1alpha1.VirtualWorkspace) string {
	clusterDomain := s.Spec.ClusterDomain
	if clusterDomain == "" {
		clusterDomain = defaultClusterDomain
	}

	return fmt.Sprintf("%s-virtual-workspace.%s.svc.%s", s.Name, s.Namespace, clusterDomain)
}

func (v *version1) GetVirtualWorkspaceBaseURL(s *operatorv1alpha1.VirtualWorkspace) string {
	return fmt.Sprintf("https://%s:6443", v.GetVirtualWorkspaceBaseHost(s))
}

func (v *version1) GetVirtualWorkspaceCertificateName(vw *operatorv1alpha1.VirtualWorkspace, certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s", vw.Name, certName)
}

// FrontProxy naming

func (v *version1) GetFrontProxyResourceLabels(f *operatorv1alpha1.FrontProxy) map[string]string {
	return v.getResourceLabels(f.Name, "front-proxy")
}

func (v *version1) GetFrontProxyDeploymentName(f *operatorv1alpha1.FrontProxy) string {
	return fmt.Sprintf("%s-front-proxy", f.Name)
}

func (v *version1) GetFrontProxyCertificateName(r *operatorv1alpha1.RootShard, f *operatorv1alpha1.FrontProxy, certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s-%s", r.Name, f.Name, certName)
}

func (v *version1) GetFrontProxyDynamicKubeconfigName(r *operatorv1alpha1.RootShard, f *operatorv1alpha1.FrontProxy) string {
	return fmt.Sprintf("%s-%s-dynamic-kubeconfig", r.Name, f.Name)
}

func (v *version1) GetFrontProxyConfigName(f *operatorv1alpha1.FrontProxy) string {
	return fmt.Sprintf("%s-config", f.Name)
}

func (v *version1) GetFrontProxyServiceName(f *operatorv1alpha1.FrontProxy) string {
	return fmt.Sprintf("%s-front-proxy", f.Name)
}

// Bundle naming

func (v *version1) GetBundleName(ownerName string) string {
	return fmt.Sprintf("%s-bundle", ownerName)
}

func (v *version1) GetMergedClientCAName(ownerName string) string {
	return fmt.Sprintf("%s-merged-client-ca", ownerName)
}
