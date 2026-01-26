/*
Copyright 2026 The KCP Authors.

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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

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
func getBundleObjectsForShard(shard *operatorv1alpha1.Shard, rootShardName string) []operatorv1alpha1.BundleObject {
	shardName := shard.Name
	namespace := shard.Namespace

	objects := []operatorv1alpha1.BundleObject{
		// CA certificates from RootShard (shared)
		{GVR: secretGVR, Name: fmt.Sprintf("%s-front-proxy-client-ca", rootShardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-requestheader-client-ca", rootShardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-server-ca", rootShardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-ca", rootShardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-client-ca", rootShardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-service-account-ca", rootShardName), Namespace: namespace},

		// Shard-specific certificates and secrets
		{GVR: secretGVR, Name: fmt.Sprintf("%s-logical-cluster-admin", shardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-logical-cluster-admin-kubeconfig", shardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-client-kubeconfig", shardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-external-logical-cluster-admin", shardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-external-logical-cluster-admin-kubeconfig", shardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-virtual-workspaces", shardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-client", shardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-server", shardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-service-account", shardName), Namespace: namespace},
	}

	// Add merged CA bundle only if CABundleSecretRef is configured
	if shard.Spec.CABundleSecretRef != nil {
		objects = append(objects, operatorv1alpha1.BundleObject{
			GVR:       secretGVR,
			Name:      fmt.Sprintf("%s-merged-ca-bundle", shardName),
			Namespace: namespace,
		})
	}

	// Deployment
	objects = append(objects, operatorv1alpha1.BundleObject{
		GVR:       deploymentGVR,
		Name:      fmt.Sprintf("%s-shard-kcp", shardName),
		Namespace: namespace,
	})

	// Service
	objects = append(objects, operatorv1alpha1.BundleObject{
		GVR:       serviceGVR,
		Name:      fmt.Sprintf("%s-shard-kcp", shardName),
		Namespace: namespace,
	})

	return objects
}

// getBundleObjectsForRootShard returns the list of objects required for a RootShard bundle
// TODO(mjudeikis): These are not yet tested. Need to double check if its full list.
func getBundleObjectsForRootShard(rootShard *operatorv1alpha1.RootShard) []operatorv1alpha1.BundleObject {
	rootShardName := rootShard.Name
	namespace := rootShard.Namespace

	objects := []operatorv1alpha1.BundleObject{
		// Root CA and intermediate CAs
		{GVR: secretGVR, Name: fmt.Sprintf("%s-ca", rootShardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-server-ca", rootShardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-requestheader-client-ca", rootShardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-client-ca", rootShardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-service-account-ca", rootShardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-front-proxy-client-ca", rootShardName), Namespace: namespace},

		// RootShard certificates
		{GVR: secretGVR, Name: fmt.Sprintf("%s-server", rootShardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-service-account", rootShardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-virtual-workspaces", rootShardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-logical-cluster-admin", rootShardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-external-logical-cluster-admin", rootShardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-operator-client", rootShardName), Namespace: namespace},

		// Kubeconfig secrets
		{GVR: secretGVR, Name: fmt.Sprintf("%s-logical-cluster-admin-kubeconfig", rootShardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-external-logical-cluster-admin-kubeconfig", rootShardName), Namespace: namespace},

		// Service
		{GVR: serviceGVR, Name: fmt.Sprintf("%s-kcp", rootShardName), Namespace: namespace},
	}

	// Add merged CA bundle if configured
	if rootShard.Spec.CABundleSecretRef != nil {
		objects = append(objects, operatorv1alpha1.BundleObject{
			GVR:       secretGVR,
			Name:      fmt.Sprintf("%s-merged-ca-bundle", rootShardName),
			Namespace: namespace,
		})
	}

	// Add proxy-related objects
	objects = append(objects, []operatorv1alpha1.BundleObject{
		{GVR: secretGVR, Name: fmt.Sprintf("%s-proxy-dynamic-kubeconfig", rootShardName), Namespace: namespace},
		{GVR: configMapGVR, Name: fmt.Sprintf("%s-proxy-config", rootShardName), Namespace: namespace},
		{GVR: serviceGVR, Name: fmt.Sprintf("%s-proxy", rootShardName), Namespace: namespace},
		{GVR: deploymentGVR, Name: fmt.Sprintf("%s-proxy", rootShardName), Namespace: namespace},
	}...)

	// Add rootshard deployment
	objects = append(objects, operatorv1alpha1.BundleObject{
		GVR:       deploymentGVR,
		Name:      fmt.Sprintf("%s-kcp", rootShardName),
		Namespace: namespace,
	})

	return objects
}

// getBundleObjectsForFrontProxy returns the list of objects required for a FrontProxy bundle
// TODO(mjudeikis): These are not yet tested. Need to double check if its full list.
func getBundleObjectsForFrontProxy(frontProxy *operatorv1alpha1.FrontProxy, rootShardName string) []operatorv1alpha1.BundleObject {
	frontProxyName := frontProxy.Name
	namespace := frontProxy.Namespace

	return []operatorv1alpha1.BundleObject{
		// CA certificates from RootShard (shared)
		{GVR: secretGVR, Name: fmt.Sprintf("%s-front-proxy-client-ca", rootShardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-requestheader-client-ca", rootShardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-server-ca", rootShardName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-ca", rootShardName), Namespace: namespace},

		// FrontProxy-specific certificates
		{GVR: secretGVR, Name: fmt.Sprintf("%s-%s-server", rootShardName, frontProxyName), Namespace: namespace},
		{GVR: secretGVR, Name: fmt.Sprintf("%s-%s-client", rootShardName, frontProxyName), Namespace: namespace},

		// FrontProxy configuration and service
		{GVR: secretGVR, Name: fmt.Sprintf("%s-%s-dynamic-kubeconfig", rootShardName, frontProxyName), Namespace: namespace},
		{GVR: configMapGVR, Name: fmt.Sprintf("%s-config", frontProxyName), Namespace: namespace},
		{GVR: serviceGVR, Name: fmt.Sprintf("%s-front-proxy", frontProxyName), Namespace: namespace},

		// FrontProxy deployment
		{GVR: deploymentGVR, Name: fmt.Sprintf("%s-front-proxy", frontProxyName), Namespace: namespace},
	}
}

// GetBundleObjectsForTarget returns the list of objects required for a Bundle based on the target
func GetBundleObjectsForTarget(target operatorv1alpha1.BundleTarget, namespace string, shard *operatorv1alpha1.Shard, rootShard *operatorv1alpha1.RootShard, frontProxy *operatorv1alpha1.FrontProxy) []operatorv1alpha1.BundleObject {
	switch {
	case target.RootShardRef != nil && rootShard != nil:
		return getBundleObjectsForRootShard(rootShard)

	case target.ShardRef != nil && shard != nil && rootShard != nil:
		return getBundleObjectsForShard(shard, rootShard.Name)

	case target.FrontProxyRef != nil && frontProxy != nil && rootShard != nil:
		return getBundleObjectsForFrontProxy(frontProxy, rootShard.Name)

	default:
		return []operatorv1alpha1.BundleObject{}
	}
}
