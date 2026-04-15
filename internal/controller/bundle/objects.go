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

package bundle

import (
	"github.com/kcp-dev/kcp-operator/internal/resources/bundling"
	"github.com/kcp-dev/kcp-operator/internal/resources/frontproxy"
	"github.com/kcp-dev/kcp-operator/internal/resources/naming"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// BundleObjectSpec defines the required objects for a specific target type
type BundleObjectSpec struct {
	// TargetKind is the kind of the target object (RootShard, Shard, FrontProxy)
	TargetKind string
	// Objects is the list of objects that should be included in the bundle
	Objects []operatorv1alpha1.BundleObject
}

// getBundleObjectsForShard returns the list of objects required for a Shard bundle
func getBundleObjectsForShard(shard *operatorv1alpha1.Shard, rootShardName string, names naming.Scheme) []operatorv1alpha1.BundleObject {
	namespace := shard.Namespace

	rootShard := &operatorv1alpha1.RootShard{}
	rootShard.Name = rootShardName
	rootShard.Namespace = namespace

	objects := []operatorv1alpha1.BundleObject{
		// CA certificates from RootShard (shared)
		bundling.NewSecret(names.RootShardCAName(rootShard, operatorv1alpha1.RootCA), namespace),
		bundling.NewSecret(names.RootShardCAName(rootShard, operatorv1alpha1.ServerCA), namespace),
		bundling.NewSecret(names.RootShardCAName(rootShard, operatorv1alpha1.RequestHeaderClientCA), namespace),
		bundling.NewSecret(names.RootShardCAName(rootShard, operatorv1alpha1.ClientCA), namespace),
		bundling.NewSecret(names.RootShardCAName(rootShard, operatorv1alpha1.ServiceAccountCA), namespace),
		bundling.NewSecret(names.RootShardCAName(rootShard, operatorv1alpha1.FrontProxyClientCA), namespace),

		// Shard-specific certificates and secrets
		bundling.NewSecret(names.ShardCertificateName(shard, operatorv1alpha1.LogicalClusterAdminCertificate), namespace),
		bundling.NewSecret(names.ShardKubeconfigSecret(shard, operatorv1alpha1.LogicalClusterAdminCertificate), namespace),
		bundling.NewSecret(names.ShardCertificateName(shard, operatorv1alpha1.ClientCertificate), namespace),
		bundling.NewSecret(names.ShardKubeconfigSecret(shard, operatorv1alpha1.ClientCertificate), namespace),
		bundling.NewSecret(names.ShardCertificateName(shard, operatorv1alpha1.ExternalLogicalClusterAdminCertificate), namespace),
		bundling.NewSecret(names.ShardKubeconfigSecret(shard, operatorv1alpha1.ExternalLogicalClusterAdminCertificate), namespace),
		bundling.NewSecret(names.ShardCertificateName(shard, operatorv1alpha1.ServerCertificate), namespace),
		bundling.NewSecret(names.ShardCertificateName(shard, operatorv1alpha1.VirtualWorkspacesCertificate), namespace),
		bundling.NewSecret(names.ShardCertificateName(shard, operatorv1alpha1.ServiceAccountCertificate), namespace),
	}

	// Add merged CA bundle only if CABundleSecretRef is configured
	if shard.Spec.CABundleSecretRef != nil {
		objects = append(objects, bundling.NewSecret(names.MergedCABundleName(shard.Name), namespace))
	}

	// Deployment
	objects = append(objects, bundling.NewDeployment(names.ShardDeploymentName(shard), namespace))

	// Service
	objects = append(objects, bundling.NewService(names.ShardServiceName(shard), namespace))

	return objects
}

// getBundleObjectsForRootShard returns the list of objects required for a RootShard bundle
// TODO(mjudeikis): These are not yet tested. Need to double check if its full list.
func getBundleObjectsForRootShard(rootShard *operatorv1alpha1.RootShard, names naming.Scheme) []operatorv1alpha1.BundleObject {
	namespace := rootShard.Namespace

	objects := []operatorv1alpha1.BundleObject{
		// Root CA and intermediate CAs
		bundling.NewSecret(names.RootShardCAName(rootShard, operatorv1alpha1.RootCA), namespace),
		bundling.NewSecret(names.RootShardCAName(rootShard, operatorv1alpha1.ServerCA), namespace),
		bundling.NewSecret(names.RootShardCAName(rootShard, operatorv1alpha1.RequestHeaderClientCA), namespace),
		bundling.NewSecret(names.RootShardCAName(rootShard, operatorv1alpha1.ClientCA), namespace),
		bundling.NewSecret(names.RootShardCAName(rootShard, operatorv1alpha1.ServiceAccountCA), namespace),
		bundling.NewSecret(names.RootShardCAName(rootShard, operatorv1alpha1.FrontProxyClientCA), namespace),

		// RootShard certificates
		bundling.NewSecret(names.RootShardCertificateName(rootShard, operatorv1alpha1.ServerCertificate), namespace),
		bundling.NewSecret(names.RootShardCertificateName(rootShard, operatorv1alpha1.ServiceAccountCertificate), namespace),
		bundling.NewSecret(names.RootShardCertificateName(rootShard, operatorv1alpha1.VirtualWorkspacesCertificate), namespace),
		bundling.NewSecret(names.RootShardCertificateName(rootShard, operatorv1alpha1.LogicalClusterAdminCertificate), namespace),
		bundling.NewSecret(names.RootShardCertificateName(rootShard, operatorv1alpha1.ExternalLogicalClusterAdminCertificate), namespace),
		bundling.NewSecret(names.RootShardCertificateName(rootShard, operatorv1alpha1.OperatorCertificate), namespace),

		// Kubeconfig secrets
		bundling.NewSecret(names.RootShardKubeconfigSecret(rootShard, operatorv1alpha1.LogicalClusterAdminCertificate), namespace),
		bundling.NewSecret(names.RootShardKubeconfigSecret(rootShard, operatorv1alpha1.ExternalLogicalClusterAdminCertificate), namespace),

		// Service
		bundling.NewService(names.RootShardServiceName(rootShard), namespace),
	}

	// Add merged CA bundle if configured
	if rootShard.Spec.CABundleSecretRef != nil {
		objects = append(objects, bundling.NewSecret(names.MergedCABundleName(rootShard.Name), namespace))
	}

	// Add proxy-related objects
	objects = append(objects, []operatorv1alpha1.BundleObject{
		bundling.NewSecret(names.RootShardProxyDynamicKubeconfigName(rootShard), namespace),
		bundling.NewConfigMap(names.RootShardProxyConfigName(rootShard), namespace),
		bundling.NewService(names.RootShardProxyServiceName(rootShard), namespace),
		bundling.NewDeployment(names.RootShardProxyDeploymentName(rootShard), namespace),
	}...)

	// Add rootshard deployment
	objects = append(objects, bundling.NewDeployment(names.RootShardDeploymentName(rootShard), namespace))

	return objects
}

// getBundleObjectsForFrontProxy returns the list of objects required for a FrontProxy bundle
// TODO(mjudeikis): These are not yet tested. Need to double check if its full list.
func getBundleObjectsForFrontProxy(frontProxy *operatorv1alpha1.FrontProxy, rootShard *operatorv1alpha1.RootShard, names naming.Scheme) []operatorv1alpha1.BundleObject {
	return frontproxy.NewFrontProxy(frontProxy, rootShard, names).Bundle()
}
