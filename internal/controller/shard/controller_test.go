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

package shard

import (
	"context"
	"testing"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func TestReconciling(t *testing.T) {
	const namespace = "shard-tests"

	testcases := []struct {
		name         string
		rootShard    *operatorv1alpha1.RootShard
		shard        *operatorv1alpha1.Shard
		extraObjects []ctrlruntimeclient.Object
		checkFunc    func(t *testing.T, client ctrlruntimeclient.Client, shard *operatorv1alpha1.Shard)
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
			shard: &operatorv1alpha1.Shard{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "shardy",
					Namespace: namespace,
				},
				Spec: operatorv1alpha1.ShardSpec{
					CommonShardSpec: operatorv1alpha1.CommonShardSpec{
						Etcd: operatorv1alpha1.EtcdConfig{
							Endpoints: []string{"https://localhost:2379"},
						},
					},
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{
							Name: "rooty",
						},
					},
				},
			},
			extraObjects: nil,
			checkFunc: func(t *testing.T, client ctrlruntimeclient.Client, shard *operatorv1alpha1.Shard) {
				// Check that the external logical cluster admin kubeconfig uses the server CA
				_ = api.Config{}
				kubeconfigSecret := &corev1.Secret{}
				err := client.Get(context.Background(), ctrlruntimeclient.ObjectKey{
					Name:      shard.Name + "-external-logical-cluster-admin-kubeconfig",
					Namespace: shard.Namespace,
				}, kubeconfigSecret)
				require.NoError(t, err)

				kubeconfigData, exists := kubeconfigSecret.Data["kubeconfig"]
				require.True(t, exists, "kubeconfig data should exist")

				config, err := clientcmd.Load(kubeconfigData)
				require.NoError(t, err)

				cluster, exists := config.Clusters["external-logical-cluster:admin"]
				require.True(t, exists, "external-logical-cluster:admin cluster should exist")

				require.Equal(t, "/etc/kcp/tls/ca/server/tls.crt", cluster.CertificateAuthority, "should use server CA path")
			},
		},
		{
			name: "with-ca-bundle-secret-ref",
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rooty-ca",
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
			shard: &operatorv1alpha1.Shard{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "shardy-ca",
					Namespace: namespace,
				},
				Spec: operatorv1alpha1.ShardSpec{
					CommonShardSpec: operatorv1alpha1.CommonShardSpec{
						Etcd: operatorv1alpha1.EtcdConfig{
							Endpoints: []string{"https://localhost:2379"},
						},
						CABundleSecretRef: &corev1.LocalObjectReference{
							Name: "custom-ca",
						},
					},
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{
							Name: "rooty-ca",
						},
					},
				},
			},
			extraObjects: []ctrlruntimeclient.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "custom-ca",
						Namespace: namespace,
					},
					Data: map[string][]byte{
						"tls.crt": []byte("custom-ca-cert"),
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "shardy-ca-server",
						Namespace: namespace,
					},
					Data: map[string][]byte{
						"tls.crt": []byte("server-ca-cert"),
					},
				},
			},
			checkFunc: func(t *testing.T, client ctrlruntimeclient.Client, shard *operatorv1alpha1.Shard) {
				// Check that the external logical cluster admin kubeconfig uses the merged CA bundle
				_ = api.Config{}
				kubeconfigSecret := &corev1.Secret{}
				err := client.Get(context.Background(), ctrlruntimeclient.ObjectKey{
					Name:      shard.Name + "-external-logical-cluster-admin-kubeconfig",
					Namespace: shard.Namespace,
				}, kubeconfigSecret)
				require.NoError(t, err)

				kubeconfigData, exists := kubeconfigSecret.Data["kubeconfig"]
				require.True(t, exists, "kubeconfig data should exist")

				config, err := clientcmd.Load(kubeconfigData)
				require.NoError(t, err)

				cluster, exists := config.Clusters["external-logical-cluster:admin"]
				require.True(t, exists, "external-logical-cluster:admin cluster should exist")

				require.Equal(t, "/etc/kcp/tls/ca/ca-bundle/tls.crt", cluster.CertificateAuthority, "should use merged CA bundle path")
			},
		},
	}

	scheme := runtime.NewScheme()
	require.Nil(t, corev1.AddToScheme(scheme))
	require.Nil(t, appsv1.AddToScheme(scheme))
	require.Nil(t, operatorv1alpha1.AddToScheme(scheme))
	require.Nil(t, certmanagerv1.AddToScheme(scheme))

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			objects := []ctrlruntimeclient.Object{testcase.rootShard, testcase.shard}
			if testcase.extraObjects != nil {
				objects = append(objects, testcase.extraObjects...)
			}

			client := ctrlruntimefakeclient.
				NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(testcase.rootShard, testcase.shard).
				WithObjects(objects...).
				Build()

			ctx := context.Background()

			controllerReconciler := &ShardReconciler{
				Client: client,
				Scheme: client.Scheme(),
			}

			// First reconcile adds finalizer and returns early
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(testcase.shard),
			})
			require.NoError(t, err)

			// Second reconcile performs actual reconciliation
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(testcase.shard),
			})
			require.NoError(t, err)

			if testcase.checkFunc != nil {
				testcase.checkFunc(t, client, testcase.shard)
			}
		})
	}
}
