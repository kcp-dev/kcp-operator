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

package rootshard

import (
	"testing"

	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kcp-dev/kcp-operator/internal/resources/naming"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func TestBuildRootShardDNSNames(t *testing.T) {
	tests := []struct {
		name      string
		rootShard *operatorv1alpha1.RootShard
		expected  []string
	}{
		{
			name: "without shardBaseURL",
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-shard",
					Namespace: "kcp",
				},
				Spec: operatorv1alpha1.RootShardSpec{
					CommonShardSpec: operatorv1alpha1.CommonShardSpec{
						ClusterDomain: "cluster.local",
					},
					External: operatorv1alpha1.ExternalConfig{
						Hostname: "api.example.com",
					},
				},
			},
			expected: []string{
				"test-shard-kcp.kcp.svc.cluster.local",
				"api.example.com",
			},
		},
		{
			name: "with shardBaseURL matching external hostname",
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-shard",
					Namespace: "kcp",
				},
				Spec: operatorv1alpha1.RootShardSpec{
					CommonShardSpec: operatorv1alpha1.CommonShardSpec{
						ClusterDomain: "cluster.local",
						ShardBaseURL:  "https://api.example.com:6443",
					},
					External: operatorv1alpha1.ExternalConfig{
						Hostname: "api.example.com",
					},
				},
			},
			expected: []string{
				"test-shard-kcp.kcp.svc.cluster.local",
				"api.example.com",
			},
		},
		{
			name: "with shardBaseURL different from external hostname",
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "root",
					Namespace: "kcp",
				},
				Spec: operatorv1alpha1.RootShardSpec{
					CommonShardSpec: operatorv1alpha1.CommonShardSpec{
						ClusterDomain: "cluster.local",
						ShardBaseURL:  "https://root.shard.kcp.example.com:6443",
					},
					External: operatorv1alpha1.ExternalConfig{
						Hostname: "api.kcp.example.com",
					},
				},
			},
			expected: []string{
				"root-kcp.kcp.svc.cluster.local",
				"api.kcp.example.com",
				"root.shard.kcp.example.com",
			},
		},
		{
			name: "with invalid shardBaseURL",
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-shard",
					Namespace: "kcp",
				},
				Spec: operatorv1alpha1.RootShardSpec{
					CommonShardSpec: operatorv1alpha1.CommonShardSpec{
						ClusterDomain: "cluster.local",
						ShardBaseURL:  "not-a-valid-url",
					},
					External: operatorv1alpha1.ExternalConfig{
						Hostname: "api.example.com",
					},
				},
			},
			expected: []string{
				"test-shard-kcp.kcp.svc.cluster.local",
				"api.example.com",
			},
		},
		{
			name: "with shardBaseURL containing subdomain",
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prod",
					Namespace: "kcp-system",
				},
				Spec: operatorv1alpha1.RootShardSpec{
					CommonShardSpec: operatorv1alpha1.CommonShardSpec{
						ShardBaseURL: "https://prod.shard.kcp.devsecops.dev.codefabric.cloud:6443",
					},
					External: operatorv1alpha1.ExternalConfig{
						Hostname: "api.kcp.devsecops.dev.codefabric.cloud",
					},
				},
			},
			expected: []string{
				"prod-kcp.kcp-system.svc.cluster.local",
				"api.kcp.devsecops.dev.codefabric.cloud",
				"prod.shard.kcp.devsecops.dev.codefabric.cloud",
			},
		},
	}

	namingScheme := naming.NewVersion1()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildRootShardDNSNames(tt.rootShard, namingScheme)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}
