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

package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/kcp-dev/logicalcluster/v3"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func NewInternalKubeconfigClient(ctx context.Context, c ctrlruntimeclient.Client, kubeconfig *operatorv1alpha1.Kubeconfig, cluster logicalcluster.Name, scheme *runtime.Scheme) (ctrlruntimeclient.Client, error) {
	target := kubeconfig.Spec.Target

	switch {
	case target.RootShardRef != nil:
		rootShard := &operatorv1alpha1.RootShard{}
		if err := c.Get(ctx, types.NamespacedName{Name: target.RootShardRef.Name, Namespace: kubeconfig.Namespace}, rootShard); err != nil {
			return nil, fmt.Errorf("failed to get RootShard: %w", err)
		}

		return NewRootShardClient(ctx, c, rootShard, cluster, scheme)

	case target.ShardRef != nil:
		shard := &operatorv1alpha1.Shard{}
		if err := c.Get(ctx, types.NamespacedName{Name: target.ShardRef.Name, Namespace: kubeconfig.Namespace}, shard); err != nil {
			return nil, fmt.Errorf("failed to get Shard: %w", err)
		}

		return NewShardClient(ctx, c, shard, cluster, scheme)

	case target.FrontProxyRef != nil:
		frontProxy := &operatorv1alpha1.FrontProxy{}
		if err := c.Get(ctx, types.NamespacedName{Name: target.FrontProxyRef.Name, Namespace: kubeconfig.Namespace}, frontProxy); err != nil {
			return nil, fmt.Errorf("failed to get FrontProxy: %w", err)
		}

		rootShard := &operatorv1alpha1.RootShard{}
		if err := c.Get(ctx, types.NamespacedName{Name: frontProxy.Spec.RootShard.Reference.Name, Namespace: kubeconfig.Namespace}, rootShard); err != nil {
			return nil, fmt.Errorf("failed to get RootShard: %w", err)
		}

		return NewRootShardProxyClient(ctx, c, rootShard, cluster, scheme)

	default:
		return nil, errors.New("no valid target configured in Kubeconfig: neither rootShard, shard nor frontProxy ref set")
	}
}
