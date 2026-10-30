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

package frontproxy

import (
	"testing"

	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func TestServerDNSNames(t *testing.T) {
	tests := []struct {
		name       string
		reconciler func() *reconciler
		expected   []string
	}{
		{
			name: "root shard proxy with default cluster domain",
			reconciler: func() *reconciler {
				return NewRootShardProxy(&operatorv1alpha1.RootShard{
					ObjectMeta: metav1.ObjectMeta{Name: "root", Namespace: "kcp"},
				})
			},
			expected: []string{
				"root-proxy",
				"root-proxy.kcp",
				"root-proxy.kcp.svc.cluster.local",
			},
		},
		{
			name: "root shard proxy with custom cluster domain",
			reconciler: func() *reconciler {
				return NewRootShardProxy(&operatorv1alpha1.RootShard{
					ObjectMeta: metav1.ObjectMeta{Name: "root", Namespace: "kcp"},
					Spec: operatorv1alpha1.RootShardSpec{
						CommonShardSpec: operatorv1alpha1.CommonShardSpec{
							ClusterDomain: "cozy.local",
						},
					},
				})
			},
			expected: []string{
				"root-proxy",
				"root-proxy.kcp",
				"root-proxy.kcp.svc.cozy.local",
			},
		},
		{
			name: "front proxy with custom cluster domain and external hostname",
			reconciler: func() *reconciler {
				return NewFrontProxy(
					&operatorv1alpha1.FrontProxy{
						ObjectMeta: metav1.ObjectMeta{Name: "kcp", Namespace: "kcp"},
					},
					&operatorv1alpha1.RootShard{
						ObjectMeta: metav1.ObjectMeta{Name: "root", Namespace: "kcp"},
						Spec: operatorv1alpha1.RootShardSpec{
							CommonShardSpec: operatorv1alpha1.CommonShardSpec{
								ClusterDomain: "cozy.local",
							},
							External: operatorv1alpha1.ExternalConfig{
								Hostname: "kcp.example.com",
							},
						},
					},
					nil,
				)
			},
			expected: []string{
				"kcp-front-proxy",
				"kcp-front-proxy.kcp",
				"kcp-front-proxy.kcp.svc.cozy.local",
				"kcp.example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ElementsMatch(t, tt.expected, tt.reconciler().serverDNSNames())
		})
	}
}
