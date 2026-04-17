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

package shard

import (
	"testing"

	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func TestBuildShardDNSNames(t *testing.T) {
	tests := []struct {
		name     string
		shard    *operatorv1alpha1.Shard
		expected []string
	}{
		{
			name: "without shardBaseURL",
			shard: &operatorv1alpha1.Shard{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-shard",
					Namespace: "kcp",
				},
				Spec: operatorv1alpha1.ShardSpec{
					CommonShardSpec: operatorv1alpha1.CommonShardSpec{
						ClusterDomain: "cluster.local",
					},
				},
			},
			expected: []string{
				"localhost",
				"test-shard-shard-kcp.kcp.svc.cluster.local",
			},
		},
		{
			name: "with shardBaseURL",
			shard: &operatorv1alpha1.Shard{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "worker-1",
					Namespace: "kcp",
				},
				Spec: operatorv1alpha1.ShardSpec{
					CommonShardSpec: operatorv1alpha1.CommonShardSpec{
						ClusterDomain: "cluster.local",
						ShardBaseURL:  "https://worker-1.shard.kcp.example.com:6443",
					},
				},
			},
			expected: []string{
				"localhost",
				"worker-1-shard-kcp.kcp.svc.cluster.local",
				"worker-1.shard.kcp.example.com",
			},
		},
		{
			name: "with shardBaseURL matching localhost",
			shard: &operatorv1alpha1.Shard{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "local-shard",
					Namespace: "kcp",
				},
				Spec: operatorv1alpha1.ShardSpec{
					CommonShardSpec: operatorv1alpha1.CommonShardSpec{
						ShardBaseURL: "https://localhost:6443",
					},
				},
			},
			expected: []string{
				"localhost",
				"local-shard-shard-kcp.kcp.svc.cluster.local",
			},
		},
		{
			name: "with invalid shardBaseURL",
			shard: &operatorv1alpha1.Shard{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-shard",
					Namespace: "kcp",
				},
				Spec: operatorv1alpha1.ShardSpec{
					CommonShardSpec: operatorv1alpha1.CommonShardSpec{
						ShardBaseURL: "not-a-valid-url",
					},
				},
			},
			expected: []string{
				"localhost",
				"test-shard-shard-kcp.kcp.svc.cluster.local",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildShardDNSNames(tt.shard)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}
