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

package client

import (
	"testing"

	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func TestInCluster(t *testing.T) {
	meta := metav1.ObjectMeta{Name: "root", Namespace: "kcp"}

	testcases := []struct {
		name          string
		clusterDomain string
		rootShard     string
		proxy         string
		shard         string
	}{
		{
			name:      "default cluster domain",
			rootShard: "https://root-kcp.kcp.svc.cluster.local:6443",
			proxy:     "https://root-proxy.kcp.svc.cluster.local:6443",
			shard:     "https://root-shard-kcp.kcp.svc.cluster.local:6443",
		},
		{
			// The whole cluster is addressed under the configured domain, and
			// the serving certificates are issued for it, so these have to
			// follow or the operator dials names nothing answers to.
			name:          "configured cluster domain",
			clusterDomain: "example.internal",
			rootShard:     "https://root-kcp.kcp.svc.example.internal:6443",
			proxy:         "https://root-proxy.kcp.svc.example.internal:6443",
			shard:         "https://root-shard-kcp.kcp.svc.example.internal:6443",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			rootShard := &operatorv1alpha1.RootShard{
				ObjectMeta: meta,
				Spec: operatorv1alpha1.RootShardSpec{
					CommonShardSpec: operatorv1alpha1.CommonShardSpec{ClusterDomain: tc.clusterDomain},
				},
			}
			shard := &operatorv1alpha1.Shard{
				ObjectMeta: meta,
				Spec: operatorv1alpha1.ShardSpec{
					CommonShardSpec: operatorv1alpha1.CommonShardSpec{ClusterDomain: tc.clusterDomain},
				},
			}

			assert.Equal(t, tc.rootShard, InCluster{}.RootShard(rootShard).URL)
			assert.Equal(t, tc.proxy, InCluster{}.RootShardProxy(rootShard).URL)
			assert.Equal(t, tc.shard, InCluster{}.Shard(shard).URL)
		})
	}
}
