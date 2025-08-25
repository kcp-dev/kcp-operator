/*
Copyright 2024 The KCP Authors.

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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func TestReconciling(t *testing.T) {
	const namespace = "shard-tests"

	testcases := []struct {
		name      string
		rootShard *operatorv1alpha1.RootShard
		shard     *operatorv1alpha1.Shard
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
		},
	}

	scheme := runtime.NewScheme()
	require.Nil(t, operatorv1alpha1.AddToScheme(scheme))
	require.Nil(t, certmanagerv1.AddToScheme(scheme))

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			client := ctrlruntimefakeclient.
				NewClientBuilder().
				WithScheme(scheme).
				WithObjects(testcase.rootShard).
				Build()

			ctx := context.Background()

			controllerReconciler := &ShardReconciler{
				Client: client,
				Scheme: client.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(testcase.rootShard),
			})
			require.Nil(t, err)
		})
	}
}
