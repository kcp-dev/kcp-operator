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

type Scheme interface {
	// RootShard naming
	RootShardDeploymentName(r *operatorv1alpha1.RootShard) string
	RootShardProxyDeploymentName(r *operatorv1alpha1.RootShard) string
	RootShardServiceName(r *operatorv1alpha1.RootShard) string
	RootShardResourceLabels(r *operatorv1alpha1.RootShard) map[string]string
	RootShardProxyResourceLabels(r *operatorv1alpha1.RootShard) map[string]string
	RootShardBaseHost(r *operatorv1alpha1.RootShard) string
	RootShardBaseURL(r *operatorv1alpha1.RootShard) string
	RootShardCertificateName(r *operatorv1alpha1.RootShard, cert operatorv1alpha1.Certificate) string
	RootShardProxyCertificateName(r *operatorv1alpha1.RootShard, cert operatorv1alpha1.Certificate) string
	RootShardCAName(r *operatorv1alpha1.RootShard, ca operatorv1alpha1.CA) string
	RootShardProxyDynamicKubeconfigName(r *operatorv1alpha1.RootShard) string
	RootShardProxyConfigName(r *operatorv1alpha1.RootShard) string
	RootShardProxyServiceName(r *operatorv1alpha1.RootShard) string
	RootShardKubeconfigSecret(r *operatorv1alpha1.RootShard, cert operatorv1alpha1.Certificate) string

	// Shard naming
	ShardDeploymentName(s *operatorv1alpha1.Shard) string
	ShardServiceName(s *operatorv1alpha1.Shard) string
	ShardResourceLabels(s *operatorv1alpha1.Shard) map[string]string
	ShardBaseHost(s *operatorv1alpha1.Shard) string
	ShardBaseURL(s *operatorv1alpha1.Shard) string
	ShardCertificateName(s *operatorv1alpha1.Shard, cert operatorv1alpha1.Certificate) string
	ShardKubeconfigSecret(s *operatorv1alpha1.Shard, cert operatorv1alpha1.Certificate) string

	// CacheServer naming
	CacheServerDeploymentName(c *operatorv1alpha1.CacheServer) string
	CacheServerServiceName(c *operatorv1alpha1.CacheServer) string
	CacheServerResourceLabels(c *operatorv1alpha1.CacheServer) map[string]string
	CacheServerBaseHost(c *operatorv1alpha1.CacheServer) string
	CacheServerBaseURL(c *operatorv1alpha1.CacheServer) string
	CacheServerCertificateName(c *operatorv1alpha1.CacheServer, cert operatorv1alpha1.Certificate) string
	CacheServerCAName(cacheServerName string, ca operatorv1alpha1.CA) string
	// deprecated, use CacheServerCertificateName instead
	CacheServerClientCertificateName(cacheServerName string) string
	CacheServerKubeconfigName(cacheServerName string) string

	// VirtualWorkspace naming
	VirtualWorkspaceDeploymentName(vw *operatorv1alpha1.VirtualWorkspace) string
	VirtualWorkspaceServiceName(vw *operatorv1alpha1.VirtualWorkspace) string
	VirtualWorkspaceResourceLabels(vw *operatorv1alpha1.VirtualWorkspace) map[string]string
	VirtualWorkspaceBaseHost(vw *operatorv1alpha1.VirtualWorkspace) string
	VirtualWorkspaceBaseURL(vw *operatorv1alpha1.VirtualWorkspace) string
	VirtualWorkspaceCertificateName(vw *operatorv1alpha1.VirtualWorkspace, cert operatorv1alpha1.Certificate) string

	// FrontProxy naming
	FrontProxyResourceLabels(fp *operatorv1alpha1.FrontProxy) map[string]string
	FrontProxyDeploymentName(fp *operatorv1alpha1.FrontProxy) string
	FrontProxyCertificateName(r *operatorv1alpha1.RootShard, fp *operatorv1alpha1.FrontProxy, cert operatorv1alpha1.Certificate) string
	FrontProxyDynamicKubeconfigName(r *operatorv1alpha1.RootShard, fp *operatorv1alpha1.FrontProxy) string
	FrontProxyConfigName(fp *operatorv1alpha1.FrontProxy) string
	FrontProxyServiceName(fp *operatorv1alpha1.FrontProxy) string
	FrontProxyBaseHost(fp *operatorv1alpha1.FrontProxy, r *operatorv1alpha1.RootShard) string

	// Bundle naming
	BundleName(ownerName string) string
	MergedCABundleName(ownerName string) string
	MergedClientCAName(ownerName string) string
}

func fqService(svcName, namespace, clusterDomain string) string {
	if clusterDomain == "" {
		clusterDomain = defaultClusterDomain
	}

	return fmt.Sprintf("%s.%s.svc.%s", svcName, namespace, clusterDomain)
}
