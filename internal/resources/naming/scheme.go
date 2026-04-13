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
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

type Scheme interface {
	// RootShard naming
	GetRootShardDeploymentName(r *operatorv1alpha1.RootShard) string
	GetRootShardProxyDeploymentName(r *operatorv1alpha1.RootShard) string
	GetRootShardServiceName(r *operatorv1alpha1.RootShard) string
	GetRootShardResourceLabels(r *operatorv1alpha1.RootShard) map[string]string
	GetRootShardProxyResourceLabels(r *operatorv1alpha1.RootShard) map[string]string
	GetRootShardBaseHost(r *operatorv1alpha1.RootShard) string
	GetRootShardBaseURL(r *operatorv1alpha1.RootShard) string
	GetRootShardCertificateName(r *operatorv1alpha1.RootShard, certName operatorv1alpha1.Certificate) string
	GetRootShardProxyCertificateName(r *operatorv1alpha1.RootShard, certName operatorv1alpha1.Certificate) string
	GetRootShardCAName(r *operatorv1alpha1.RootShard, caName operatorv1alpha1.CA) string
	GetRootShardProxyDynamicKubeconfigName(r *operatorv1alpha1.RootShard) string
	GetRootShardProxyConfigName(r *operatorv1alpha1.RootShard) string
	GetRootShardProxyServiceName(r *operatorv1alpha1.RootShard) string
	GetRootShardKubeconfigSecret(r *operatorv1alpha1.RootShard, cert operatorv1alpha1.Certificate) string

	// Shard naming
	GetShardDeploymentName(s *operatorv1alpha1.Shard) string
	GetShardServiceName(s *operatorv1alpha1.Shard) string
	GetShardResourceLabels(s *operatorv1alpha1.Shard) map[string]string
	GetShardBaseHost(s *operatorv1alpha1.Shard) string
	GetShardBaseURL(s *operatorv1alpha1.Shard) string
	GetShardCertificateName(s *operatorv1alpha1.Shard, certName operatorv1alpha1.Certificate) string
	GetShardKubeconfigSecret(shard *operatorv1alpha1.Shard, cert operatorv1alpha1.Certificate) string

	// CacheServer naming
	GetCacheServerDeploymentName(s *operatorv1alpha1.CacheServer) string
	GetCacheServerServiceName(s *operatorv1alpha1.CacheServer) string
	GetCacheServerResourceLabels(s *operatorv1alpha1.CacheServer) map[string]string
	GetCacheServerBaseHost(s *operatorv1alpha1.CacheServer) string
	GetCacheServerBaseURL(s *operatorv1alpha1.CacheServer) string
	GetCacheServerCertificateName(s *operatorv1alpha1.CacheServer, certName operatorv1alpha1.Certificate) string
	GetCacheServerCAName(cacheServerName string, caName operatorv1alpha1.CA) string
	GetCacheServerClientCertificateName(s *operatorv1alpha1.CacheServer) string
	GetCacheServerKubeconfigName(cacheServerName string) string

	// VirtualWorkspace naming
	GetVirtualWorkspaceDeploymentName(vw *operatorv1alpha1.VirtualWorkspace) string
	GetVirtualWorkspaceResourceLabels(vw *operatorv1alpha1.VirtualWorkspace) map[string]string
	GetVirtualWorkspaceBaseHost(s *operatorv1alpha1.VirtualWorkspace) string
	GetVirtualWorkspaceBaseURL(s *operatorv1alpha1.VirtualWorkspace) string
	GetVirtualWorkspaceCertificateName(vw *operatorv1alpha1.VirtualWorkspace, certName operatorv1alpha1.Certificate) string

	// FrontProxy naming
	GetFrontProxyResourceLabels(f *operatorv1alpha1.FrontProxy) map[string]string
	GetFrontProxyDeploymentName(f *operatorv1alpha1.FrontProxy) string
	GetFrontProxyCertificateName(r *operatorv1alpha1.RootShard, f *operatorv1alpha1.FrontProxy, certName operatorv1alpha1.Certificate) string
	GetFrontProxyDynamicKubeconfigName(r *operatorv1alpha1.RootShard, f *operatorv1alpha1.FrontProxy) string
	GetFrontProxyConfigName(f *operatorv1alpha1.FrontProxy) string
	GetFrontProxyServiceName(f *operatorv1alpha1.FrontProxy) string

	// Bundle naming
	GetBundleName(ownerName string) string
	GetMergedClientCAName(ownerName string) string
}
