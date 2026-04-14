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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

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

var (
	// secretGVR is the GVR for Secret resources
	secretGVR = schema.GroupVersionResource{
		Group:    corev1.GroupName,
		Version:  "v1",
		Resource: "secrets",
	}

	// serviceGVR is the GVR for Service resources
	serviceGVR = schema.GroupVersionResource{
		Group:    corev1.GroupName,
		Version:  "v1",
		Resource: "services",
	}

	// configMapGVR is the GVR for ConfigMap resources
	configMapGVR = schema.GroupVersionResource{
		Group:    corev1.GroupName,
		Version:  "v1",
		Resource: "configmaps",
	}

	// deploymentGVR is the GVR for Deployment resources
	deploymentGVR = schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}
)

// getBundleObjectsForShard returns the list of objects required for a Shard bundle
func getBundleObjectsForShard(shard *operatorv1alpha1.Shard, rootShardName string, names naming.Scheme) []operatorv1alpha1.BundleObject {
	namespace := shard.Namespace

	rootShard := &operatorv1alpha1.RootShard{}
	rootShard.Name = rootShardName
	rootShard.Namespace = namespace

	objects := []operatorv1alpha1.BundleObject{
		// CA certificates from RootShard (shared)
		{GVR: secretGVR, Name: names.RootShardCAName(rootShard, operatorv1alpha1.RootCA), Namespace: namespace},
		{GVR: secretGVR, Name: names.RootShardCAName(rootShard, operatorv1alpha1.ServerCA), Namespace: namespace},
		{GVR: secretGVR, Name: names.RootShardCAName(rootShard, operatorv1alpha1.RequestHeaderClientCA), Namespace: namespace},
		{GVR: secretGVR, Name: names.RootShardCAName(rootShard, operatorv1alpha1.ClientCA), Namespace: namespace},
		{GVR: secretGVR, Name: names.RootShardCAName(rootShard, operatorv1alpha1.ServiceAccountCA), Namespace: namespace},
		{GVR: secretGVR, Name: names.RootShardCAName(rootShard, operatorv1alpha1.FrontProxyClientCA), Namespace: namespace},

		// Shard-specific certificates and secrets
		{GVR: secretGVR, Name: names.ShardCertificateName(shard, operatorv1alpha1.LogicalClusterAdminCertificate), Namespace: namespace},
		{GVR: secretGVR, Name: names.ShardKubeconfigSecret(shard, operatorv1alpha1.LogicalClusterAdminCertificate), Namespace: namespace},
		{GVR: secretGVR, Name: names.ShardCertificateName(shard, operatorv1alpha1.ClientCertificate), Namespace: namespace},
		{GVR: secretGVR, Name: names.ShardKubeconfigSecret(shard, operatorv1alpha1.ClientCertificate), Namespace: namespace},
		{GVR: secretGVR, Name: names.ShardCertificateName(shard, operatorv1alpha1.ExternalLogicalClusterAdminCertificate), Namespace: namespace},
		{GVR: secretGVR, Name: names.ShardKubeconfigSecret(shard, operatorv1alpha1.ExternalLogicalClusterAdminCertificate), Namespace: namespace},
		{GVR: secretGVR, Name: names.ShardCertificateName(shard, operatorv1alpha1.ServerCertificate), Namespace: namespace},
		{GVR: secretGVR, Name: names.ShardCertificateName(shard, operatorv1alpha1.VirtualWorkspacesCertificate), Namespace: namespace},
		{GVR: secretGVR, Name: names.ShardCertificateName(shard, operatorv1alpha1.ServiceAccountCertificate), Namespace: namespace},
	}

	// Add merged CA bundle only if CABundleSecretRef is configured
	if shard.Spec.CABundleSecretRef != nil {
		objects = append(objects, operatorv1alpha1.BundleObject{
			GVR:       secretGVR,
			Name:      names.MergedCABundleName(shard.Name),
			Namespace: namespace,
		})
	}

	// Deployment
	objects = append(objects, operatorv1alpha1.BundleObject{
		GVR:       deploymentGVR,
		Name:      names.ShardDeploymentName(shard),
		Namespace: namespace,
	})

	// Service
	objects = append(objects, operatorv1alpha1.BundleObject{
		GVR:       serviceGVR,
		Name:      names.ShardServiceName(shard),
		Namespace: namespace,
	})

	return objects
}

// getBundleObjectsForRootShard returns the list of objects required for a RootShard bundle
// TODO(mjudeikis): These are not yet tested. Need to double check if its full list.
func getBundleObjectsForRootShard(rootShard *operatorv1alpha1.RootShard, names naming.Scheme) []operatorv1alpha1.BundleObject {
	namespace := rootShard.Namespace

	objects := []operatorv1alpha1.BundleObject{
		// Root CA and intermediate CAs
		{GVR: secretGVR, Name: names.RootShardCAName(rootShard, operatorv1alpha1.RootCA), Namespace: namespace},
		{GVR: secretGVR, Name: names.RootShardCAName(rootShard, operatorv1alpha1.ServerCA), Namespace: namespace},
		{GVR: secretGVR, Name: names.RootShardCAName(rootShard, operatorv1alpha1.RequestHeaderClientCA), Namespace: namespace},
		{GVR: secretGVR, Name: names.RootShardCAName(rootShard, operatorv1alpha1.ClientCA), Namespace: namespace},
		{GVR: secretGVR, Name: names.RootShardCAName(rootShard, operatorv1alpha1.ServiceAccountCA), Namespace: namespace},
		{GVR: secretGVR, Name: names.RootShardCAName(rootShard, operatorv1alpha1.FrontProxyClientCA), Namespace: namespace},

		// RootShard certificates
		{GVR: secretGVR, Name: names.RootShardCertificateName(rootShard, operatorv1alpha1.ServerCertificate), Namespace: namespace},
		{GVR: secretGVR, Name: names.RootShardCertificateName(rootShard, operatorv1alpha1.ServiceAccountCertificate), Namespace: namespace},
		{GVR: secretGVR, Name: names.RootShardCertificateName(rootShard, operatorv1alpha1.VirtualWorkspacesCertificate), Namespace: namespace},
		{GVR: secretGVR, Name: names.RootShardCertificateName(rootShard, operatorv1alpha1.LogicalClusterAdminCertificate), Namespace: namespace},
		{GVR: secretGVR, Name: names.RootShardCertificateName(rootShard, operatorv1alpha1.ExternalLogicalClusterAdminCertificate), Namespace: namespace},
		{GVR: secretGVR, Name: names.RootShardCertificateName(rootShard, operatorv1alpha1.OperatorCertificate), Namespace: namespace},

		// Kubeconfig secrets
		{GVR: secretGVR, Name: names.RootShardKubeconfigSecret(rootShard, operatorv1alpha1.LogicalClusterAdminCertificate), Namespace: namespace},
		{GVR: secretGVR, Name: names.RootShardKubeconfigSecret(rootShard, operatorv1alpha1.ExternalLogicalClusterAdminCertificate), Namespace: namespace},

		// Service
		{GVR: serviceGVR, Name: names.RootShardServiceName(rootShard), Namespace: namespace},
	}

	// Add merged CA bundle if configured
	if rootShard.Spec.CABundleSecretRef != nil {
		objects = append(objects, operatorv1alpha1.BundleObject{
			GVR:       secretGVR,
			Name:      names.MergedCABundleName(rootShard.Name),
			Namespace: namespace,
		})
	}

	// Add proxy-related objects
	objects = append(objects, []operatorv1alpha1.BundleObject{
		{GVR: secretGVR, Name: names.RootShardProxyDynamicKubeconfigName(rootShard), Namespace: namespace},
		{GVR: configMapGVR, Name: names.RootShardProxyConfigName(rootShard), Namespace: namespace},
		{GVR: serviceGVR, Name: names.RootShardProxyServiceName(rootShard), Namespace: namespace},
		{GVR: deploymentGVR, Name: names.RootShardProxyDeploymentName(rootShard), Namespace: namespace},
	}...)

	// Add rootshard deployment
	objects = append(objects, operatorv1alpha1.BundleObject{
		GVR:       deploymentGVR,
		Name:      names.RootShardDeploymentName(rootShard),
		Namespace: namespace,
	})

	return objects
}

// getBundleObjectsForFrontProxy returns the list of objects required for a FrontProxy bundle
// TODO(mjudeikis): These are not yet tested. Need to double check if its full list.
func getBundleObjectsForFrontProxy(frontProxy *operatorv1alpha1.FrontProxy, rootShardName string, names naming.Scheme) []operatorv1alpha1.BundleObject {
	namespace := frontProxy.Namespace

	rootShard := &operatorv1alpha1.RootShard{}
	rootShard.Name = rootShardName
	rootShard.Namespace = namespace

	return []operatorv1alpha1.BundleObject{
		// CA certificates from RootShard (shared) (Secret names are identical to Cert names)
		{GVR: secretGVR, Name: names.RootShardCAName(rootShard, operatorv1alpha1.FrontProxyClientCA), Namespace: namespace},
		{GVR: secretGVR, Name: names.RootShardCAName(rootShard, operatorv1alpha1.RequestHeaderClientCA), Namespace: namespace},
		{GVR: secretGVR, Name: names.RootShardCAName(rootShard, operatorv1alpha1.ServerCA), Namespace: namespace},
		{GVR: secretGVR, Name: names.RootShardCAName(rootShard, operatorv1alpha1.RootCA), Namespace: namespace},

		// FrontProxy-specific certificates
		{GVR: secretGVR, Name: names.FrontProxyCertificateName(rootShard, frontProxy, operatorv1alpha1.ServerCertificate), Namespace: namespace},
		{GVR: secretGVR, Name: names.FrontProxyCertificateName(rootShard, frontProxy, operatorv1alpha1.ClientCertificate), Namespace: namespace},

		// FrontProxy configuration and service
		{GVR: secretGVR, Name: names.FrontProxyDynamicKubeconfigName(rootShard, frontProxy), Namespace: namespace},
		{GVR: configMapGVR, Name: names.FrontProxyConfigName(frontProxy), Namespace: namespace},
		{GVR: serviceGVR, Name: names.FrontProxyServiceName(frontProxy), Namespace: namespace},

		// FrontProxy deployment
		{GVR: deploymentGVR, Name: names.FrontProxyDeploymentName(frontProxy), Namespace: namespace},
	}
}
