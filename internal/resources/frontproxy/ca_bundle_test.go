/*
Copyright 2026 The KCP Authors.

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
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func TestMergedClientCASecretReconciler(t *testing.T) {
	tests := []struct {
		name                      string
		frontProxy                *operatorv1alpha1.FrontProxy
		rootShard                 *operatorv1alpha1.RootShard
		clientCASecret            *corev1.Secret
		additionalClientCASecret  *corev1.Secret
		expectAdditionalCAInMerge bool
	}{
		{
			name: "without clientCABundleRef merges only ClientCA",
			frontProxy: &operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-front-proxy",
					Namespace: "test-namespace",
				},
				Spec: operatorv1alpha1.FrontProxySpec{
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{Name: "test-root-shard"},
					},
				},
			},
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-root-shard",
					Namespace: "test-namespace",
				},
				Spec: operatorv1alpha1.RootShardSpec{
					External: operatorv1alpha1.ExternalConfig{
						Hostname: "kcp.example.com",
						Port:     6443,
					},
				},
			},
			clientCASecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-root-shard-client-ca",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"tls.crt": []byte("-----BEGIN CERTIFICATE-----\nClientCA\n-----END CERTIFICATE-----"),
				},
			},
			expectAdditionalCAInMerge: false,
		},
		{
			name: "with clientCABundleRef merges ClientCA and additional CA",
			frontProxy: &operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-front-proxy",
					Namespace: "test-namespace",
				},
				Spec: operatorv1alpha1.FrontProxySpec{
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{Name: "test-root-shard"},
					},
					ClientCABundleRef: &corev1.LocalObjectReference{
						Name: "additional-client-ca",
					},
				},
			},
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-root-shard",
					Namespace: "test-namespace",
				},
				Spec: operatorv1alpha1.RootShardSpec{
					External: operatorv1alpha1.ExternalConfig{
						Hostname: "kcp.example.com",
						Port:     6443,
					},
				},
			},
			clientCASecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-root-shard-client-ca",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"tls.crt": []byte("-----BEGIN CERTIFICATE-----\nClientCA\n-----END CERTIFICATE-----"),
				},
			},
			additionalClientCASecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "additional-client-ca",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"tls.crt": []byte("-----BEGIN CERTIFICATE-----\nAdditionalClientCA\n-----END CERTIFICATE-----"),
				},
			},
			expectAdditionalCAInMerge: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, corev1.AddToScheme(scheme))
			require.NoError(t, operatorv1alpha1.AddToScheme(scheme))

			objects := []ctrlruntimeclient.Object{
				tt.frontProxy,
				tt.rootShard,
				tt.clientCASecret,
			}
			if tt.additionalClientCASecret != nil {
				objects = append(objects, tt.additionalClientCASecret)
			}

			_ = ctrlruntimefakeclient.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objects...).
				Build()

			rec := NewFrontProxy(tt.frontProxy, tt.rootShard)

			// Fetch the ClientCA data
			clientCACert := tt.clientCASecret.Data["tls.crt"]

			// Fetch additional CA bundle data if configured
			var additionalClientCABundle []byte
			if tt.additionalClientCASecret != nil {
				additionalClientCABundle = tt.additionalClientCASecret.Data["tls.crt"]
			}

			reconcilerFactory := rec.clientCABundleSecretReconciler(clientCACert, additionalClientCABundle)

			secretName, reconciler := reconcilerFactory()
			require.Equal(t, "test-front-proxy-merged-client-ca", secretName)

			mergedSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "test-namespace",
				},
			}

			result, err := reconciler(mergedSecret)
			require.NoError(t, err)
			require.NotNil(t, result)

			mergedCA := result.Data["tls.crt"]
			require.NotEmpty(t, mergedCA)

			// Verify ClientCA is always present
			require.Contains(t, string(mergedCA), "ClientCA", "Merged CA should contain ClientCA")

			// Verify additional CA is present only when configured
			if tt.expectAdditionalCAInMerge {
				require.Contains(t, string(mergedCA), "AdditionalClientCA",
					"Merged CA should contain AdditionalClientCA when clientCABundleRef is set")

				// Verify both CAs are separated by newline
				require.True(t, bytes.Contains(mergedCA, []byte("ClientCA\n-----END CERTIFICATE-----\n-----BEGIN CERTIFICATE-----\nAdditionalClientCA")),
					"CAs should be separated by newline")
			} else {
				require.NotContains(t, string(mergedCA), "AdditionalClientCA",
					"Merged CA should not contain AdditionalClientCA when clientCABundleRef is not set")
			}

			// Verify labels are set correctly
			require.Equal(t, tt.frontProxy.Name, result.Labels["operator.kcp.io/front-proxy"])
		})
	}
}

func TestMergedClientCASecretReconciler_RootShardProxy(t *testing.T) {
	// Test the root shard internal proxy (without a FrontProxy object)
	rootShard := &operatorv1alpha1.RootShard{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-root-shard",
			Namespace: "test-namespace",
		},
		Spec: operatorv1alpha1.RootShardSpec{
			External: operatorv1alpha1.ExternalConfig{
				Hostname: "kcp.example.com",
				Port:     6443,
			},
		},
	}

	clientCASecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-root-shard-client-ca",
			Namespace: "test-namespace",
		},
		Data: map[string][]byte{
			"tls.crt": []byte("-----BEGIN CERTIFICATE-----\nClientCA\n-----END CERTIFICATE-----"),
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, operatorv1alpha1.AddToScheme(scheme))

	rec := NewRootShardProxy(rootShard)

	// Fetch the ClientCA data
	clientCACert := clientCASecret.Data["tls.crt"]

	// Root shard proxy doesn't support additional CA bundle
	reconcilerFactory := rec.clientCABundleSecretReconciler(clientCACert, nil)

	secretName, reconciler := reconcilerFactory()
	require.Equal(t, "test-root-shard-proxy-merged-client-ca", secretName)

	mergedSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: "test-namespace",
		},
	}

	result, err := reconciler(mergedSecret)
	require.NoError(t, err)
	require.NotNil(t, result)

	mergedCA := result.Data["tls.crt"]
	require.NotEmpty(t, mergedCA)
	require.Contains(t, string(mergedCA), "ClientCA")

	// For root shard proxy, no additional CA should be added (clientCABundleRef is only for FrontProxy)
	require.Equal(t, clientCASecret.Data["tls.crt"], mergedCA,
		"Root shard proxy should only contain ClientCA without any additional bundle")

	// Verify labels are set correctly
	require.Equal(t, rootShard.Name, result.Labels["operator.kcp.io/rootshard"])
}
