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

package compiledvirtualworkspace

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kcp-dev/kcp-operator/internal/controller/util"
	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func TestReconciling(t *testing.T) {
	const namespace = "virtualworkspace-tests"

	testcases := []struct {
		name string
		vw   *deployv1alpha1.CompiledVirtualWorkspace
	}{
		{
			name: "vanilla",
			vw: &deployv1alpha1.CompiledVirtualWorkspace{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "virtual",
					Namespace: namespace,
				},
				Spec: deployv1alpha1.CompiledVirtualWorkspaceSpec{
					VirtualWorkspace: operatorv1alpha1.VirtualWorkspaceSpec{
						External: operatorv1alpha1.ExternalConfig{
							Hostname: "example.com",
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
				WithStatusSubresource(testcase.vw).
				WithObjects(testcase.vw).
				Build()

			ctx := context.Background()

			controllerReconciler := &CompiledVirtualWorkspaceReconciler{
				Client: client,
				Scheme: client.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(testcase.vw),
			})
			require.NoError(t, err)
		})
	}
}
