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

package compiledvirtualworkspace

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kcp-dev/kcp-operator/internal/controller/util"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/virtualworkspace"
	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func TestReconciling(t *testing.T) {
	const namespace = "compiled-virtualworkspace-tests"

	compiled := &deployv1alpha1.CompiledVirtualWorkspace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "confy",
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
					CommonShardSpec: operatorv1alpha1.CommonShardSpec{
						Etcd: operatorv1alpha1.EtcdConfig{
							Endpoints: []string{"https://localhost:2379"},
						},
					},
				},
			},
		},
	}

	vw := &operatorv1alpha1.VirtualWorkspace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      compiled.Name,
			Namespace: compiled.Namespace,
		},
	}

	rootShard := &operatorv1alpha1.RootShard{
		ObjectMeta: metav1.ObjectMeta{
			Name:      compiled.Spec.RootShard.Name,
			Namespace: compiled.Namespace,
		},
	}

	// Secrets mounted into the deployment must exist for the revision-labels modifier.
	var mountedSecretNames []string
	for _, ca := range []operatorv1alpha1.CA{
		operatorv1alpha1.RootCA,
		operatorv1alpha1.ServerCA,
		operatorv1alpha1.RequestHeaderClientCA,
		operatorv1alpha1.ClientCA,
	} {
		mountedSecretNames = append(mountedSecretNames, resources.GetRootShardCAName(rootShard, ca))
	}
	for _, cert := range []operatorv1alpha1.Certificate{
		operatorv1alpha1.ServerCertificate,
		operatorv1alpha1.ClientCertificate,
	} {
		mountedSecretNames = append(mountedSecretNames, resources.GetVirtualWorkspaceCertificateName(vw, cert))
	}
	mountedSecretNames = append(mountedSecretNames, resources.GetRootShardKubeconfigSecret(rootShard, operatorv1alpha1.LogicalClusterAdminCertificate))

	objects := []ctrlruntimeclient.Object{compiled}
	for _, name := range mountedSecretNames {
		objects = append(objects, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		})
	}

	scheme := util.GetTestScheme()

	client := ctrlruntimefakeclient.
		NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		WithStatusSubresource(compiled).
		Build()

	ctx := context.Background()

	controllerReconciler := &Reconciler{
		Client: client,
		Scheme: client.Scheme(),
	}

	_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
		NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(compiled),
	})
	require.NoError(t, err)

	deployment := &appsv1.Deployment{}
	err = client.Get(ctx, ctrlruntimeclient.ObjectKey{
		Name:      resources.GetVirtualWorkspaceDeploymentName(vw),
		Namespace: namespace,
	}, deployment)
	require.NoError(t, err)
	require.Len(t, deployment.OwnerReferences, 1)
	require.Equal(t, "CompiledVirtualWorkspace", deployment.OwnerReferences[0].Kind)
	require.Equal(t, compiled.Name, deployment.OwnerReferences[0].Name)

	service := &corev1.Service{}
	err = client.Get(ctx, ctrlruntimeclient.ObjectKey{
		Name:      virtualworkspace.GetVirtualWorkspaceServiceName(vw),
		Namespace: namespace,
	}, service)
	require.NoError(t, err)
	require.Len(t, service.OwnerReferences, 1)
	require.Equal(t, "CompiledVirtualWorkspace", service.OwnerReferences[0].Kind)
	require.Equal(t, compiled.Name, service.OwnerReferences[0].Name)

	updated := &deployv1alpha1.CompiledVirtualWorkspace{}
	err = client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(compiled), updated)
	require.NoError(t, err)

	availableCond := apimeta.FindStatusCondition(updated.Status.Conditions, string(operatorv1alpha1.ConditionTypeAvailable))
	require.NotNil(t, availableCond)
	require.Equal(t, metav1.ConditionFalse, availableCond.Status)
}
