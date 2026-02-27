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

package cacheserver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kcp-dev/kcp-operator/internal/controller/util"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func TestReconciling(t *testing.T) {
	const namespace = "cacheserver-tests"

	testcases := []struct {
		name        string
		cacheServer *operatorv1alpha1.CacheServer
	}{
		{
			name: "vanilla",
			cacheServer: &operatorv1alpha1.CacheServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource",
					Namespace: namespace,
				},
				Spec: operatorv1alpha1.CacheServerSpec{
					Certificates: operatorv1alpha1.Certificates{
						IssuerRef: &operatorv1alpha1.ObjectReference{
							Group: "cert-manager.io",
							Kind:  "Issuer",
							Name:  "test",
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
				WithStatusSubresource(testcase.cacheServer).
				WithObjects(testcase.cacheServer).
				Build()

			ctx := context.Background()

			controllerReconciler := &CacheServerReconciler{
				Client: client,
				Scheme: client.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(testcase.cacheServer),
			})
			require.NoError(t, err)
		})
	}
}
