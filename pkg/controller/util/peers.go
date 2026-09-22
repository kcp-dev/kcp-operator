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

package util

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// GetShardPeers returns the root shard and all given shards as peers serving the
// Admin workspace (/services/admin). Like kcp itself, a shard with an external
// VirtualWorkspace is reached through that VirtualWorkspace, since the shard does
// not serve /services/ in that case.
func GetShardPeers(ctx context.Context, client ctrlruntimeclient.Client, rootShard *operatorv1alpha1.RootShard, shards []operatorv1alpha1.Shard) ([]deployv1alpha1.ShardPeer, error) {
	rootURL, err := adminEndpoint(ctx, client, rootShard.Namespace, rootShard.Spec.KCPVirtualWorkspace, resources.GetRootShardBaseURL(rootShard))
	if err != nil {
		return nil, err
	}

	peers := []deployv1alpha1.ShardPeer{{Name: rootShard.Name, URL: rootURL}}

	for i := range shards {
		shard := &shards[i]

		url, err := adminEndpoint(ctx, client, shard.Namespace, shard.Spec.KCPVirtualWorkspace, resources.GetShardBaseURL(shard))
		if err != nil {
			return nil, err
		}

		peers = append(peers, deployv1alpha1.ShardPeer{Name: shard.Name, URL: url})
	}

	return peers, nil
}

func adminEndpoint(ctx context.Context, client ctrlruntimeclient.Client, namespace string, vwRef *corev1.LocalObjectReference, baseURL string) (string, error) {
	if vwRef == nil {
		return baseURL, nil
	}

	vw := &operatorv1alpha1.VirtualWorkspace{}
	if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: vwRef.Name}, vw); err != nil {
		return "", fmt.Errorf("failed to get VirtualWorkspace %s: %w", vwRef.Name, err)
	}

	return fmt.Sprintf("https://%s:%d", vw.Spec.External.Hostname, vw.Spec.External.Port), nil
}
