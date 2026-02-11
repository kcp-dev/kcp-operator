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

// This file has been copied from the multicluster-provider to avoid a costly
// dependency for a little bit of convenience.

package utils

import (
	"fmt"
	"sync"

	"github.com/kcp-dev/logicalcluster/v3"

	"k8s.io/client-go/rest"
	"k8s.io/utils/lru"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterClient is a cluster-aware client.
type ClusterClient interface {
	// Cluster returns the client for the given cluster.
	Cluster(cluster logicalcluster.Path) ctrlruntimeclient.Client
}

// clusterClient is a multi-cluster-aware client.
type clusterClient struct {
	baseConfig *rest.Config
	opts       ctrlruntimeclient.Options

	lock  sync.RWMutex
	cache *lru.Cache
}

// newClusterClient creates a new cluster-aware client.
func newClusterClient(cfg *rest.Config, options ctrlruntimeclient.Options) ClusterClient {
	ca := lru.New(100)
	return &clusterClient{
		opts:       options,
		baseConfig: cfg,
		cache:      ca,
	}
}

func (c *clusterClient) Cluster(cluster logicalcluster.Path) ctrlruntimeclient.Client {
	// quick path
	c.lock.RLock()
	cli, ok := c.cache.Get(cluster)
	c.lock.RUnlock()
	if ok {
		return cli.(ctrlruntimeclient.Client)
	}

	// slow path
	c.lock.Lock()
	defer c.lock.Unlock()
	if cli, ok := c.cache.Get(cluster); ok {
		return cli.(ctrlruntimeclient.Client)
	}

	// cache miss
	cfg := rest.CopyConfig(c.baseConfig)
	cfg.Host += cluster.RequestPath()
	cli, err := ctrlruntimeclient.New(cfg, c.opts)
	if err != nil {
		panic(fmt.Errorf("failed to create client for cluster %s: %w", cluster, err))
	}
	c.cache.Add(cluster, cli)
	return cli.(ctrlruntimeclient.Client)
}
