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

package virtualworkspace

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
	const namespace = "virtual-workspace-tests"

	testcases := []struct {
		name             string
		rootShard        *operatorv1alpha1.RootShard
		virtualWorkspace *operatorv1alpha1.VirtualWorkspace
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
			virtualWorkspace: &operatorv1alpha1.VirtualWorkspace{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "confy",
					Namespace: namespace,
				},
				Spec: operatorv1alpha1.VirtualWorkspaceSpec{
					External: operatorv1alpha1.ExternalConfig{
						Hostname: "example.com",
						Port:     6443,
					},
					Target: operatorv1alpha1.VirtualWorkspaceTarget{
						RootShardRef: &corev1.LocalObjectReference{
							Name: "rooty",
						},
					},
				},
			},
		},
	}

	scheme := util.GetTestScheme()

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			client := ctrlruntimefakeclient.
				NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(testcase.rootShard).
				WithStatusSubresource(testcase.virtualWorkspace).
				WithObjects(testcase.rootShard, testcase.virtualWorkspace).
				Build()

			ctx := context.Background()

			controllerReconciler := &Reconciler{
				Client: client,
				Scheme: client.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(testcase.virtualWorkspace),
			})
			require.NoError(t, err)
		})
	}
}

func TestClientCABundleMerging(t *testing.T) {
	const namespace = "vw-ca-tests"

	testcases := []struct {
		name                   string
		rootShard              *operatorv1alpha1.RootShard
		virtualWorkspace       *operatorv1alpha1.VirtualWorkspace
		extraSecrets           []*corev1.Secret
		expectMergedSecret     bool
		expectedMergedContents []string
	}{
		{
			name: "without any clientCABundleRef no merged secret is created",
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
			virtualWorkspace: &operatorv1alpha1.VirtualWorkspace{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vw-no-bundle",
					Namespace: namespace,
				},
				Spec: operatorv1alpha1.VirtualWorkspaceSpec{
					External: operatorv1alpha1.ExternalConfig{
						Hostname: "vw.example.com",
						Port:     6443,
					},
					Target: operatorv1alpha1.VirtualWorkspaceTarget{
						RootShardRef: &corev1.LocalObjectReference{
							Name: "rooty-no-bundle",
						},
					},
				},
			},
			expectMergedSecret: false,
		},
		{
			name: "with rootShard clientCABundleRef only, merged secret contains ClientCA and RootShard bundle",
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
							Name: "rootshard-extra-ca",
						},
					},
				},
			},
			virtualWorkspace: &operatorv1alpha1.VirtualWorkspace{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vw-inherits",
					Namespace: namespace,
				},
				Spec: operatorv1alpha1.VirtualWorkspaceSpec{
					External: operatorv1alpha1.ExternalConfig{
						Hostname: "vw.example.com",
						Port:     6443,
					},
					Target: operatorv1alpha1.VirtualWorkspaceTarget{
						RootShardRef: &corev1.LocalObjectReference{
							Name: "rooty-with-bundle",
						},
					},
				},
			},
			extraSecrets: []*corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rootshard-extra-ca",
						Namespace: namespace,
					},
					Data: map[string][]byte{
						"tls.crt": []byte("-----BEGIN CERTIFICATE-----\nRootShardExtraCA\n-----END CERTIFICATE-----"),
					},
				},
			},
			expectMergedSecret:     true,
			expectedMergedContents: []string{"RootClientCA", "RootShardExtraCA"},
		},
		{
			name: "with virtualWorkspace clientCABundleRef only, merged secret contains ClientCA and VW bundle",
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rooty-plain",
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
			virtualWorkspace: &operatorv1alpha1.VirtualWorkspace{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vw-own-bundle",
					Namespace: namespace,
				},
				Spec: operatorv1alpha1.VirtualWorkspaceSpec{
					External: operatorv1alpha1.ExternalConfig{
						Hostname: "vw.example.com",
						Port:     6443,
					},
					Target: operatorv1alpha1.VirtualWorkspaceTarget{
						RootShardRef: &corev1.LocalObjectReference{
							Name: "rooty-plain",
						},
					},
					ClientCABundleRef: &corev1.LocalObjectReference{
						Name: "vw-extra-ca",
					},
				},
			},
			extraSecrets: []*corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "vw-extra-ca",
						Namespace: namespace,
					},
					Data: map[string][]byte{
						"tls.crt": []byte("-----BEGIN CERTIFICATE-----\nVWExtraCA\n-----END CERTIFICATE-----"),
					},
				},
			},
			expectMergedSecret:     true,
			expectedMergedContents: []string{"RootClientCA", "VWExtraCA"},
		},
		{
			name: "with both rootShard and virtualWorkspace clientCABundleRef, merged secret contains all three CAs",
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rooty-both",
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
							Name: "rootshard-ca-both",
						},
					},
				},
			},
			virtualWorkspace: &operatorv1alpha1.VirtualWorkspace{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vw-both",
					Namespace: namespace,
				},
				Spec: operatorv1alpha1.VirtualWorkspaceSpec{
					External: operatorv1alpha1.ExternalConfig{
						Hostname: "vw.example.com",
						Port:     6443,
					},
					Target: operatorv1alpha1.VirtualWorkspaceTarget{
						RootShardRef: &corev1.LocalObjectReference{
							Name: "rooty-both",
						},
					},
					ClientCABundleRef: &corev1.LocalObjectReference{
						Name: "vw-ca-both",
					},
				},
			},
			extraSecrets: []*corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rootshard-ca-both",
						Namespace: namespace,
					},
					Data: map[string][]byte{
						"tls.crt": []byte("-----BEGIN CERTIFICATE-----\nRootShardExtraCA\n-----END CERTIFICATE-----"),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "vw-ca-both",
						Namespace: namespace,
					},
					Data: map[string][]byte{
						"tls.crt": []byte("-----BEGIN CERTIFICATE-----\nVWExtraCA\n-----END CERTIFICATE-----"),
					},
				},
			},
			expectMergedSecret:     true,
			expectedMergedContents: []string{"RootClientCA", "RootShardExtraCA", "VWExtraCA"},
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

			objects := []ctrlruntimeclient.Object{tc.rootShard, tc.virtualWorkspace, clientCASecret}
			for _, s := range tc.extraSecrets {
				objects = append(objects, s)
			}

			client := ctrlruntimefakeclient.
				NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(tc.rootShard, tc.virtualWorkspace).
				WithObjects(objects...).
				Build()

			ctx := context.Background()

			controllerReconciler := &Reconciler{
				Client: client,
				Scheme: client.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(tc.virtualWorkspace),
			})
			require.NoError(t, err)

			// Check if merged secret exists
			mergedSecret := &corev1.Secret{}
			mergedSecretName := tc.virtualWorkspace.Name + "-merged-client-ca"
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
				require.Error(t, err, "merged client CA secret should not exist when no clientCABundleRef is set")
			}
		})
	}
}
