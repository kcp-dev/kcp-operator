/*
Copyright 2024 The kcp Authors.

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
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kcp-dev/kcp-operator/internal/controller/util"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func TestReconciling(t *testing.T) {
	const namespace = "rootshard-tests"

	testcases := []struct {
		name      string
		rootShard *operatorv1alpha1.RootShard
	}{
		{
			name: "vanilla",
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rooty",
					Namespace: namespace,
				},
				Spec: operatorv1alpha1.RootShardSpec{
					External: operatorv1alpha1.ExternalConfig{
						Hostname: "example.kcp.io",
						Port:     6443,
					},
					CommonShardSpec: operatorv1alpha1.CommonShardSpec{
						Etcd: operatorv1alpha1.EtcdConfig{
							Endpoints: []string{"https://localhost:2379"},
						},
					},
				},
			},
		},
		{
			name: "vanilla with authentication webhook",
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rooty",
					Namespace: namespace,
				},
				Spec: operatorv1alpha1.RootShardSpec{
					External: operatorv1alpha1.ExternalConfig{
						Hostname: "example.kcp.io",
						Port:     6443,
					},
					CommonShardSpec: operatorv1alpha1.CommonShardSpec{
						Etcd: operatorv1alpha1.EtcdConfig{
							Endpoints: []string{"https://localhost:2379"},
						},
						Auth: &operatorv1alpha1.AuthSpec{
							Webhook: &operatorv1alpha1.AuthenticationWebhookSpec{
								ConfigSecretName: "test-webhook-config",
							},
						},
					},
				},
			},
		},
	}

	scheme := util.GetTestScheme()

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			// The merged client CA reconciler fetches ClientCA.
			clientCASecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testcase.rootShard.Name + "-client-ca",
					Namespace: namespace,
				},
				Data: map[string][]byte{
					"tls.crt": []byte("client-ca-cert"),
				},
			}

			client := ctrlruntimefakeclient.
				NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(testcase.rootShard).
				WithObjects(testcase.rootShard, clientCASecret).
				Build()

			ctx := context.Background()

			controllerReconciler := &RootShardReconciler{
				Client: client,
				Scheme: client.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(testcase.rootShard),
			})
			require.NoError(t, err)
		})
	}
}

func TestClientCABundleMerging(t *testing.T) {
	const namespace = "rootshard-ca-tests"

	testcases := []struct {
		name                   string
		rootShard              *operatorv1alpha1.RootShard
		extraSecrets           []*corev1.Secret
		expectMergedSecret     bool
		expectedMergedContents []string
	}{
		{
			name: "without clientCABundleRef no merged secret is created",
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rooty-no-bundle",
					Namespace: namespace,
				},
				Spec: operatorv1alpha1.RootShardSpec{
					External: operatorv1alpha1.ExternalConfig{
						Hostname: "example.kcp.io",
						Port:     6443,
					},
					CommonShardSpec: operatorv1alpha1.CommonShardSpec{
						Etcd: operatorv1alpha1.EtcdConfig{
							Endpoints: []string{"https://localhost:2379"},
						},
					},
				},
			},
			expectMergedSecret: false,
		},
		{
			name: "with clientCABundleRef merged secret contains both CAs",
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rooty-with-bundle",
					Namespace: namespace,
				},
				Spec: operatorv1alpha1.RootShardSpec{
					External: operatorv1alpha1.ExternalConfig{
						Hostname: "example.kcp.io",
						Port:     6443,
					},
					CommonShardSpec: operatorv1alpha1.CommonShardSpec{
						Etcd: operatorv1alpha1.EtcdConfig{
							Endpoints: []string{"https://localhost:2379"},
						},
						ClientCABundleRef: &corev1.LocalObjectReference{
							Name: "extra-client-ca",
						},
					},
				},
			},
			extraSecrets: []*corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "extra-client-ca",
						Namespace: namespace,
					},
					Data: map[string][]byte{
						"tls.crt": []byte("-----BEGIN CERTIFICATE-----\nExtraClientCA\n-----END CERTIFICATE-----"),
					},
				},
			},
			expectMergedSecret:     true,
			expectedMergedContents: []string{"RootClientCA", "ExtraClientCA"},
		},
	}

	scheme := util.GetTestScheme()

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			clientCASecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tc.rootShard.Name + "-client-ca",
					Namespace: namespace,
				},
				Data: map[string][]byte{
					"tls.crt": []byte("-----BEGIN CERTIFICATE-----\nRootClientCA\n-----END CERTIFICATE-----"),
				},
			}

			objects := []ctrlruntimeclient.Object{tc.rootShard, clientCASecret}
			for _, s := range tc.extraSecrets {
				objects = append(objects, s)
			}

			client := ctrlruntimefakeclient.
				NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(tc.rootShard).
				WithObjects(objects...).
				Build()

			ctx := context.Background()

			controllerReconciler := &RootShardReconciler{
				Client: client,
				Scheme: client.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(tc.rootShard),
			})
			require.NoError(t, err)

			// Check if merged secret exists
			mergedSecret := &corev1.Secret{}
			mergedSecretName := tc.rootShard.Name + "-merged-client-ca"
			err = client.Get(ctx, ctrlruntimeclient.ObjectKey{
				Name:      mergedSecretName,
				Namespace: namespace,
			}, mergedSecret)

			if tc.expectMergedSecret {
				require.NoError(t, err, "merged client CA secret should exist")
				require.NotNil(t, mergedSecret.Data["tls.crt"], "merged secret should contain tls.crt")

				mergedData := string(mergedSecret.Data["tls.crt"])
				for _, expected := range tc.expectedMergedContents {
					require.Contains(t, mergedData, expected,
						"merged CA should contain %s", expected)
				}
			} else {
				require.Error(t, err, "merged client CA secret should not exist when clientCABundleRef is not set")
			}
		})
	}
}
