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

	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func TestDefaultPathMappings(t *testing.T) {
	tests := []struct {
		name           string
		frontProxy     *operatorv1alpha1.FrontProxy
		rootShard      *operatorv1alpha1.RootShard
		expectedCAPath string
	}{
		{
			name: "without CABundleSecretRef uses default CA path",
			frontProxy: &operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{Name: "test-front-proxy"},
				Spec: operatorv1alpha1.FrontProxySpec{
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{Name: "test-root-shard"},
					},
				},
			},
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{Name: "test-root-shard"},
				Spec: operatorv1alpha1.RootShardSpec{
					External: operatorv1alpha1.ExternalConfig{
						Hostname: "kcp.example.com",
						Port:     6443,
					},
				},
			},
			expectedCAPath: "/etc/kcp/tls/ca/tls.crt",
		},
		{
			name: "with CABundleSecretRef uses ca-bundle path",
			frontProxy: &operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{Name: "test-front-proxy"},
				Spec: operatorv1alpha1.FrontProxySpec{
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{Name: "test-root-shard"},
					},
					CABundleSecretRef: &corev1.LocalObjectReference{
						Name: "custom-ca-bundle",
					},
				},
			},
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{Name: "test-root-shard"},
				Spec: operatorv1alpha1.RootShardSpec{
					External: operatorv1alpha1.ExternalConfig{
						Hostname: "kcp.example.com",
						Port:     6443,
					},
				},
			},
			expectedCAPath: "/etc/kcp/tls/ca/ca-bundle/tls.crt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := NewFrontProxy(tt.frontProxy, tt.rootShard)
			mappings := rec.defaultPathMappings()

			require.Len(t, mappings, 2)
			for _, mapping := range mappings {
				require.Equal(t, tt.expectedCAPath, mapping.BackendServerCA,
					"Expected BackendServerCA to be %s for path %s", tt.expectedCAPath, mapping.Path)
			}
		})
	}
}
