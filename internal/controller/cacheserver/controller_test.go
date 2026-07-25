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
	"fmt"
	"testing"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/require"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kcp-dev/kcp-operator/internal/controller/util"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
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

			compiled := &deployv1alpha1.CompiledCacheServer{}
			err = client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(testcase.cacheServer), compiled)
			require.True(t, apierrors.IsNotFound(err))

			require.NoError(t, util.MarkCertificatesReady(ctx, client, namespace))

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(testcase.cacheServer),
			})
			require.NoError(t, err)

			err = client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(testcase.cacheServer), compiled)
			require.NoError(t, err)
			require.Equal(t, testcase.cacheServer.Spec, compiled.Spec.CacheServer)
			require.Equal(t, testcase.cacheServer.Name, compiled.Labels[resources.CacheServerLabel])
			require.Len(t, compiled.OwnerReferences, 1)
			require.Equal(t, "CacheServer", compiled.OwnerReferences[0].Kind)
			require.Equal(t, testcase.cacheServer.Name, compiled.OwnerReferences[0].Name)

			certs := &certmanagerv1.CertificateList{}
			require.NoError(t, client.List(ctx, certs, ctrlruntimeclient.InNamespace(namespace)))
			require.NotEmpty(t, certs.Items)
			for _, cert := range certs.Items {
				require.Equal(t, "1", compiled.Annotations[fmt.Sprintf("operator.kcp.io/cert-%s-revision", cert.Name)])
			}
		})
	}
}
