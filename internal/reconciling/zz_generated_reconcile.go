/*
Copyright 2024 The KCP Authors.

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
package reconciling

import (
	"context"
	"fmt"

	"k8c.io/reconciler/pkg/reconciling"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	apiv1alpha1 "github.com/kcp-dev/kcp-operator/api/v1alpha1"
)

// RootShardReconciler defines an interface to create/update RootShards.
type RootShardReconciler = func(existing *apiv1alpha1.RootShard) (*apiv1alpha1.RootShard, error)

// NamedRootShardReconcilerFactory returns the name of the resource and the corresponding Reconciler function.
type NamedRootShardReconcilerFactory = func() (name string, reconciler RootShardReconciler)

// RootShardObjectWrapper adds a wrapper so the RootShardReconciler matches ObjectReconciler.
// This is needed as Go does not support function interface matching.
func RootShardObjectWrapper(reconciler RootShardReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return reconciler(existing.(*apiv1alpha1.RootShard))
		}
		return reconciler(&apiv1alpha1.RootShard{})
	}
}

// ReconcileRootShards will create and update the RootShards coming from the passed RootShardReconciler slice.
func ReconcileRootShards(ctx context.Context, namedFactories []NamedRootShardReconcilerFactory, namespace string, client ctrlruntimeclient.Client, objectModifiers ...reconciling.ObjectModifier) error {
	for _, factory := range namedFactories {
		name, reconciler := factory()
		reconcileObject := RootShardObjectWrapper(reconciler)
		reconcileObject = reconciling.CreateWithNamespace(reconcileObject, namespace)
		reconcileObject = reconciling.CreateWithName(reconcileObject, name)

		for _, objectModifier := range objectModifiers {
			reconcileObject = objectModifier(reconcileObject)
		}

		if err := reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, reconcileObject, client, &apiv1alpha1.RootShard{}, false); err != nil {
			return fmt.Errorf("failed to ensure RootShard %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// ShardReconciler defines an interface to create/update Shards.
type ShardReconciler = func(existing *apiv1alpha1.Shard) (*apiv1alpha1.Shard, error)

// NamedShardReconcilerFactory returns the name of the resource and the corresponding Reconciler function.
type NamedShardReconcilerFactory = func() (name string, reconciler ShardReconciler)

// ShardObjectWrapper adds a wrapper so the ShardReconciler matches ObjectReconciler.
// This is needed as Go does not support function interface matching.
func ShardObjectWrapper(reconciler ShardReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return reconciler(existing.(*apiv1alpha1.Shard))
		}
		return reconciler(&apiv1alpha1.Shard{})
	}
}

// ReconcileShards will create and update the Shards coming from the passed ShardReconciler slice.
func ReconcileShards(ctx context.Context, namedFactories []NamedShardReconcilerFactory, namespace string, client ctrlruntimeclient.Client, objectModifiers ...reconciling.ObjectModifier) error {
	for _, factory := range namedFactories {
		name, reconciler := factory()
		reconcileObject := ShardObjectWrapper(reconciler)
		reconcileObject = reconciling.CreateWithNamespace(reconcileObject, namespace)
		reconcileObject = reconciling.CreateWithName(reconcileObject, name)

		for _, objectModifier := range objectModifiers {
			reconcileObject = objectModifier(reconcileObject)
		}

		if err := reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, reconcileObject, client, &apiv1alpha1.Shard{}, false); err != nil {
			return fmt.Errorf("failed to ensure Shard %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// CacheServerReconciler defines an interface to create/update CacheServers.
type CacheServerReconciler = func(existing *apiv1alpha1.CacheServer) (*apiv1alpha1.CacheServer, error)

// NamedCacheServerReconcilerFactory returns the name of the resource and the corresponding Reconciler function.
type NamedCacheServerReconcilerFactory = func() (name string, reconciler CacheServerReconciler)

// CacheServerObjectWrapper adds a wrapper so the CacheServerReconciler matches ObjectReconciler.
// This is needed as Go does not support function interface matching.
func CacheServerObjectWrapper(reconciler CacheServerReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return reconciler(existing.(*apiv1alpha1.CacheServer))
		}
		return reconciler(&apiv1alpha1.CacheServer{})
	}
}

// ReconcileCacheServers will create and update the CacheServers coming from the passed CacheServerReconciler slice.
func ReconcileCacheServers(ctx context.Context, namedFactories []NamedCacheServerReconcilerFactory, namespace string, client ctrlruntimeclient.Client, objectModifiers ...reconciling.ObjectModifier) error {
	for _, factory := range namedFactories {
		name, reconciler := factory()
		reconcileObject := CacheServerObjectWrapper(reconciler)
		reconcileObject = reconciling.CreateWithNamespace(reconcileObject, namespace)
		reconcileObject = reconciling.CreateWithName(reconcileObject, name)

		for _, objectModifier := range objectModifiers {
			reconcileObject = objectModifier(reconcileObject)
		}

		if err := reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, reconcileObject, client, &apiv1alpha1.CacheServer{}, false); err != nil {
			return fmt.Errorf("failed to ensure CacheServer %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// FrontProxyReconciler defines an interface to create/update FrontProxys.
type FrontProxyReconciler = func(existing *apiv1alpha1.FrontProxy) (*apiv1alpha1.FrontProxy, error)

// NamedFrontProxyReconcilerFactory returns the name of the resource and the corresponding Reconciler function.
type NamedFrontProxyReconcilerFactory = func() (name string, reconciler FrontProxyReconciler)

// FrontProxyObjectWrapper adds a wrapper so the FrontProxyReconciler matches ObjectReconciler.
// This is needed as Go does not support function interface matching.
func FrontProxyObjectWrapper(reconciler FrontProxyReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return reconciler(existing.(*apiv1alpha1.FrontProxy))
		}
		return reconciler(&apiv1alpha1.FrontProxy{})
	}
}

// ReconcileFrontProxys will create and update the FrontProxys coming from the passed FrontProxyReconciler slice.
func ReconcileFrontProxys(ctx context.Context, namedFactories []NamedFrontProxyReconcilerFactory, namespace string, client ctrlruntimeclient.Client, objectModifiers ...reconciling.ObjectModifier) error {
	for _, factory := range namedFactories {
		name, reconciler := factory()
		reconcileObject := FrontProxyObjectWrapper(reconciler)
		reconcileObject = reconciling.CreateWithNamespace(reconcileObject, namespace)
		reconcileObject = reconciling.CreateWithName(reconcileObject, name)

		for _, objectModifier := range objectModifiers {
			reconcileObject = objectModifier(reconcileObject)
		}

		if err := reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, reconcileObject, client, &apiv1alpha1.FrontProxy{}, false); err != nil {
			return fmt.Errorf("failed to ensure FrontProxy %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// KubeconfigReconciler defines an interface to create/update Kubeconfigs.
type KubeconfigReconciler = func(existing *apiv1alpha1.Kubeconfig) (*apiv1alpha1.Kubeconfig, error)

// NamedKubeconfigReconcilerFactory returns the name of the resource and the corresponding Reconciler function.
type NamedKubeconfigReconcilerFactory = func() (name string, reconciler KubeconfigReconciler)

// KubeconfigObjectWrapper adds a wrapper so the KubeconfigReconciler matches ObjectReconciler.
// This is needed as Go does not support function interface matching.
func KubeconfigObjectWrapper(reconciler KubeconfigReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return reconciler(existing.(*apiv1alpha1.Kubeconfig))
		}
		return reconciler(&apiv1alpha1.Kubeconfig{})
	}
}

// ReconcileKubeconfigs will create and update the Kubeconfigs coming from the passed KubeconfigReconciler slice.
func ReconcileKubeconfigs(ctx context.Context, namedFactories []NamedKubeconfigReconcilerFactory, namespace string, client ctrlruntimeclient.Client, objectModifiers ...reconciling.ObjectModifier) error {
	for _, factory := range namedFactories {
		name, reconciler := factory()
		reconcileObject := KubeconfigObjectWrapper(reconciler)
		reconcileObject = reconciling.CreateWithNamespace(reconcileObject, namespace)
		reconcileObject = reconciling.CreateWithName(reconcileObject, name)

		for _, objectModifier := range objectModifiers {
			reconcileObject = objectModifier(reconcileObject)
		}

		if err := reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, reconcileObject, client, &apiv1alpha1.Kubeconfig{}, false); err != nil {
			return fmt.Errorf("failed to ensure Kubeconfig %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}
