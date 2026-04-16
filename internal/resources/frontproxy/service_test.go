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

func TestGetExternalPort(t *testing.T) {
	tests := []struct {
		name         string
		frontProxy   *operatorv1alpha1.FrontProxy
		rootShard    *operatorv1alpha1.RootShard
		expectedPort int
	}{
		{
			name: "FrontProxy with explicit External.Port",
			frontProxy: &operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{Name: "test-front-proxy"},
				Spec: operatorv1alpha1.FrontProxySpec{
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{Name: "test-root-shard"},
					},
					External: operatorv1alpha1.ExternalConfig{
						Port: 8443,
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
			expectedPort: 8443,
		},
		{
			name: "FrontProxy with deprecated ExternalHostname including port",
			frontProxy: &operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{Name: "test-front-proxy"},
				Spec: operatorv1alpha1.FrontProxySpec{
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{Name: "test-root-shard"},
					},
					ExternalHostname: "kcp.example.com:9443",
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
			expectedPort: 9443,
		},
		{
			name: "FrontProxy External.Port takes precedence over deprecated ExternalHostname",
			frontProxy: &operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{Name: "test-front-proxy"},
				Spec: operatorv1alpha1.FrontProxySpec{
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{Name: "test-root-shard"},
					},
					External: operatorv1alpha1.ExternalConfig{
						Port: 8443,
					},
					ExternalHostname: "kcp.example.com:9443",
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
			expectedPort: 8443,
		},
		{
			name: "FrontProxy falls back to RootShard External.Port",
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
						Port:     7443,
					},
				},
			},
			expectedPort: 7443,
		},
		{
			name: "FrontProxy with ExternalHostname without a port defaults to 6443",
			frontProxy: &operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{Name: "test-front-proxy"},
				Spec: operatorv1alpha1.FrontProxySpec{
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{Name: "test-root-shard"},
					},
					ExternalHostname: "kcp.example.com",
				},
			},
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{Name: "test-root-shard"},
				Spec: operatorv1alpha1.RootShardSpec{
					External: operatorv1alpha1.ExternalConfig{
						Hostname: "kcp.example.com",
						Port:     7443,
					},
				},
			},
			expectedPort: 6443,
		},
		{
			name: "FrontProxy with non-numeric port in ExternalHostname defaults to 6443",
			frontProxy: &operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{Name: "test-front-proxy"},
				Spec: operatorv1alpha1.FrontProxySpec{
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{Name: "test-root-shard"},
					},
					ExternalHostname: "kcp.example.com:invalid",
				},
			},
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{Name: "test-root-shard"},
				Spec: operatorv1alpha1.RootShardSpec{
					External: operatorv1alpha1.ExternalConfig{
						Hostname: "kcp.example.com",
						Port:     7443,
					},
				},
			},
			expectedPort: 6443,
		},
		{
			name: "FrontProxy with no port configuration uses default 6443",
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
						Port:     0,
					},
				},
			},
			expectedPort: 6443,
		},
		{
			name:       "RootShard internal proxy always uses default 6443",
			frontProxy: nil,
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{Name: "test-root-shard"},
				Spec: operatorv1alpha1.RootShardSpec{
					External: operatorv1alpha1.ExternalConfig{
						Hostname: "kcp.example.com",
						Port:     8443,
					},
				},
			},
			expectedPort: 6443,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rec *reconciler
			if tt.frontProxy != nil {
				rec = NewFrontProxy(tt.frontProxy, tt.rootShard)
			} else {
				rec = NewRootShardProxy(tt.rootShard)
			}

			actualPort := rec.getExternalPort()
			require.Equal(t, tt.expectedPort, actualPort)
		})
	}
}
