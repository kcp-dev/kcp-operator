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

func (v *version1) RootShardDeploymentName(r *operatorv1alpha1.RootShard) string {
	return fmt.Sprintf("%s-kcp", r.Name)
}

func (v *version1) RootShardProxyDeploymentName(r *operatorv1alpha1.RootShard) string {
	return fmt.Sprintf("%s-proxy", r.Name)
}

func (v *version1) RootShardServiceName(r *operatorv1alpha1.RootShard) string {
	return fmt.Sprintf("%s-kcp", r.Name)
}

func (v *version1) RootShardResourceLabels(r *operatorv1alpha1.RootShard) map[string]string {
	return v.getResourceLabels(r.Name, "rootshard")
}

func (v *version1) RootShardProxyResourceLabels(r *operatorv1alpha1.RootShard) map[string]string {
	return v.getResourceLabels(r.Name, "rootshard-proxy")
}

func (v *version1) RootShardBaseHost(r *operatorv1alpha1.RootShard) string {
	return fqService(v.RootShardServiceName(r), r.Namespace, r.Spec.ClusterDomain)
}

func (v *version1) RootShardBaseURL(r *operatorv1alpha1.RootShard) string {
	if r.Spec.ShardBaseURL != "" {
		return r.Spec.ShardBaseURL
	}
	return fmt.Sprintf("https://%s:6443", v.RootShardBaseHost(r))
}

func (v *version1) RootShardCertificateName(r *operatorv1alpha1.RootShard, cert operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s", r.Name, cert)
}

func (v *version1) RootShardProxyCertificateName(r *operatorv1alpha1.RootShard, cert operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-proxy-%s", r.Name, cert)
}

func (v *version1) RootShardCAName(r *operatorv1alpha1.RootShard, ca operatorv1alpha1.CA) string {
	if ca == operatorv1alpha1.RootCA {
		return fmt.Sprintf("%s-ca", r.Name)
	}
	return fmt.Sprintf("%s-%s-ca", r.Name, ca)
}

func (v *version1) RootShardProxyDynamicKubeconfigName(r *operatorv1alpha1.RootShard) string {
	return fmt.Sprintf("%s-proxy-dynamic-kubeconfig", r.Name)
}

func (v *version1) RootShardProxyConfigName(r *operatorv1alpha1.RootShard) string {
	return fmt.Sprintf("%s-proxy-config", r.Name)
}

func (v *version1) RootShardProxyServiceName(r *operatorv1alpha1.RootShard) string {
	return fmt.Sprintf("%s-proxy", r.Name)
}

func (v *version1) RootShardKubeconfigSecret(r *operatorv1alpha1.RootShard, cert operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s-kubeconfig", r.Name, cert)
}

// Shard naming

func (v *version1) ShardDeploymentName(s *operatorv1alpha1.Shard) string {
	return fmt.Sprintf("%s-shard-kcp", s.Name)
}

func (v *version1) ShardServiceName(s *operatorv1alpha1.Shard) string {
	return fmt.Sprintf("%s-shard-kcp", s.Name)
}

func (v *version1) ShardResourceLabels(s *operatorv1alpha1.Shard) map[string]string {
	return v.getResourceLabels(s.Name, "shard")
}

func (v *version1) ShardBaseHost(s *operatorv1alpha1.Shard) string {
	return fqService(v.ShardServiceName(s), s.Namespace, s.Spec.ClusterDomain)
}

func (v *version1) ShardBaseURL(s *operatorv1alpha1.Shard) string {
	if s.Spec.ShardBaseURL != "" {
		return s.Spec.ShardBaseURL
	}
	return fmt.Sprintf("https://%s:6443", v.ShardBaseHost(s))
}

func (v *version1) ShardCertificateName(s *operatorv1alpha1.Shard, cert operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s", s.Name, cert)
}

func (v *version1) ShardKubeconfigSecret(s *operatorv1alpha1.Shard, cert operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s-kubeconfig", s.Name, cert)
}

// CacheServer naming

func (v *version1) CacheServerDeploymentName(c *operatorv1alpha1.CacheServer) string {
	return fmt.Sprintf("%s-cache-server", c.Name)
}

func (v *version1) CacheServerServiceName(c *operatorv1alpha1.CacheServer) string {
	return fmt.Sprintf("%s-cache-server", c.Name)
}

func (v *version1) CacheServerResourceLabels(c *operatorv1alpha1.CacheServer) map[string]string {
	return v.getResourceLabels(c.Name, "cache-server")
}

func (v *version1) CacheServerBaseHost(c *operatorv1alpha1.CacheServer) string {
	return fqService(v.CacheServerServiceName(c), c.Namespace, c.Spec.ClusterDomain)
}

func (v *version1) CacheServerBaseURL(c *operatorv1alpha1.CacheServer) string {
	return fmt.Sprintf("https://%s:6443", v.CacheServerBaseHost(c))
}

func (v *version1) CacheServerCertificateName(c *operatorv1alpha1.CacheServer, cert operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s", c.Name, cert)
}

func (v *version1) CacheServerCAName(cacheServerName string, ca operatorv1alpha1.CA) string {
	if ca == operatorv1alpha1.RootCA {
		return fmt.Sprintf("%s-ca", cacheServerName)
	}
	return fmt.Sprintf("%s-%s-ca", cacheServerName, ca)
}

func (v *version1) CacheServerClientCertificateName(c *operatorv1alpha1.CacheServer) string {
	return fmt.Sprintf("%s-client-certificate", c.Name)
}

func (v *version1) CacheServerKubeconfigName(cacheServerName string) string {
	return fmt.Sprintf("%s-kubeconfig", cacheServerName)
}

// VirtualWorkspace naming

func (v *version1) VirtualWorkspaceDeploymentName(vw *operatorv1alpha1.VirtualWorkspace) string {
	return fmt.Sprintf("%s-virtual-workspace", vw.Name)
}

func (v *version1) VirtualWorkspaceServiceName(vw *operatorv1alpha1.VirtualWorkspace) string {
	return fmt.Sprintf("%s-virtual-workspace", vw.Name)
}

func (v *version1) VirtualWorkspaceResourceLabels(vw *operatorv1alpha1.VirtualWorkspace) map[string]string {
	return v.getResourceLabels(vw.Name, "virtual-workspace")
}

func (v *version1) VirtualWorkspaceBaseHost(vw *operatorv1alpha1.VirtualWorkspace) string {
	return fqService(v.VirtualWorkspaceServiceName(vw), vw.Namespace, vw.Spec.ClusterDomain)
}

func (v *version1) VirtualWorkspaceBaseURL(vw *operatorv1alpha1.VirtualWorkspace) string {
	return fmt.Sprintf("https://%s:6443", v.VirtualWorkspaceBaseHost(vw))
}

func (v *version1) VirtualWorkspaceCertificateName(vw *operatorv1alpha1.VirtualWorkspace, cert operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s", vw.Name, cert)
}

// FrontProxy naming

func (v *version1) FrontProxyResourceLabels(fp *operatorv1alpha1.FrontProxy) map[string]string {
	return v.getResourceLabels(fp.Name, "front-proxy")
}

func (v *version1) FrontProxyDeploymentName(fp *operatorv1alpha1.FrontProxy) string {
	return fmt.Sprintf("%s-front-proxy", fp.Name)
}

func (v *version1) FrontProxyCertificateName(r *operatorv1alpha1.RootShard, fp *operatorv1alpha1.FrontProxy, cert operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s-%s", r.Name, fp.Name, cert)
}

func (v *version1) FrontProxyDynamicKubeconfigName(r *operatorv1alpha1.RootShard, fp *operatorv1alpha1.FrontProxy) string {
	return fmt.Sprintf("%s-%s-dynamic-kubeconfig", r.Name, fp.Name)
}

func (v *version1) FrontProxyConfigName(fp *operatorv1alpha1.FrontProxy) string {
	return fmt.Sprintf("%s-config", fp.Name)
}

func (v *version1) FrontProxyServiceName(fp *operatorv1alpha1.FrontProxy) string {
	return fmt.Sprintf("%s-front-proxy", fp.Name)
}

// Bundle naming

func (v *version1) BundleName(ownerName string) string {
	return fmt.Sprintf("%s-bundle", ownerName)
}

func (v *version1) MergedClientCAName(ownerName string) string {
	return fmt.Sprintf("%s-merged-client-ca", ownerName)
}
