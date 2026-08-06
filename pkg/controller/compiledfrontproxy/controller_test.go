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

package compiledfrontproxy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	mcreconcile "sigs.k8s.io/multicluster-runtime/pkg/reconcile"

	"github.com/kcp-dev/kcp-operator/pkg/controller/util"
	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func TestReconciling(t *testing.T) {
	const namespace = "frontproxy-tests"

	testcases := []struct {
		name       string
		frontProxy *deployv1alpha1.CompiledFrontProxy
	}{
		{
			name: "vanilla",
			frontProxy: &deployv1alpha1.CompiledFrontProxy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "frontal",
					Namespace: namespace,
				},
				Spec: deployv1alpha1.CompiledFrontProxySpec{
					FrontProxy: operatorv1alpha1.FrontProxySpec{
						External: operatorv1alpha1.ExternalConfig{
							Hostname: "example.kcp.io",
							Port:     6443,
						},
					},
					RootShard: deployv1alpha1.NamedRootShardSpec{
						Name: "rooty",
						Spec: operatorv1alpha1.RootShardSpec{
							External: operatorv1alpha1.ExternalConfig{
								Hostname: "example.kcp.io",
								Port:     6443,
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
			client := ctrlruntimefakeclient.
				NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(testcase.frontProxy).
				WithObjects(testcase.frontProxy).
				Build()

			ctx := context.Background()

			controllerReconciler := &CompiledFrontProxyReconciler{
				GetCluster: util.SingleCluster(client),
			}

			_, err := controllerReconciler.Reconcile(ctx, mcreconcile.Request{
				Request: reconcile.Request{
					NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(testcase.frontProxy),
				},
			})
			require.NoError(t, err)
		})
	}
}
