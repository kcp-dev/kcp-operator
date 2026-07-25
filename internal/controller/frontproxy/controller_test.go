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

package frontproxy

import (
	"context"
	"fmt"
	"testing"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
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
	const namespace = "frontproxy-tests"

	testcases := []struct {
		name       string
		rootShard  *operatorv1alpha1.RootShard
		frontProxy *operatorv1alpha1.FrontProxy
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
			frontProxy: &operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fronty",
					Namespace: namespace,
				},
				Spec: operatorv1alpha1.FrontProxySpec{
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{
							Name: "rooty",
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
					},
				},
			},
			frontProxy: &operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fronty",
					Namespace: namespace,
				},
				Spec: operatorv1alpha1.FrontProxySpec{
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{
							Name: "rooty",
						},
					},
					Auth: &operatorv1alpha1.AuthSpec{
						Webhook: &operatorv1alpha1.AuthenticationWebhookSpec{
							ConfigSecretName: "test-webhook-config",
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
				WithStatusSubresource(testcase.rootShard, testcase.frontProxy).
				WithObjects(testcase.rootShard, testcase.frontProxy, clientCASecret).
				Build()

			ctx := context.Background()

			controllerReconciler := &FrontProxyReconciler{
				Client: client,
				Scheme: client.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(testcase.frontProxy),
			})
			require.NoError(t, err)

			compiled := &deployv1alpha1.CompiledFrontProxy{}
			err = client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(testcase.frontProxy), compiled)
			require.True(t, apierrors.IsNotFound(err))

			require.NoError(t, util.MarkCertificatesReady(ctx, client, namespace))

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(testcase.frontProxy),
			})
			require.NoError(t, err)

			err = client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(testcase.frontProxy), compiled)
			require.NoError(t, err)
			require.Equal(t, testcase.frontProxy.Spec, compiled.Spec.FrontProxy)
			require.Equal(t, testcase.rootShard.Name, compiled.Spec.RootShard.Name)
			require.Equal(t, testcase.rootShard.Spec, compiled.Spec.RootShard.Spec)
			require.Equal(t, testcase.rootShard.Name, compiled.Labels[resources.RootShardLabel])
			require.Equal(t, testcase.frontProxy.Name, compiled.Labels[resources.FrontProxyLabel])
			require.Len(t, compiled.OwnerReferences, 1)
			require.Equal(t, "FrontProxy", compiled.OwnerReferences[0].Kind)
			require.Equal(t, testcase.frontProxy.Name, compiled.OwnerReferences[0].Name)

			certs := &certmanagerv1.CertificateList{}
			require.NoError(t, client.List(ctx, certs, ctrlruntimeclient.InNamespace(namespace)))
			require.NotEmpty(t, certs.Items)
			for _, cert := range certs.Items {
				require.Equal(t, "1", compiled.Annotations[fmt.Sprintf("operator.kcp.io/cert-%s-revision", cert.Name)])
			}
		})
	}
}

func TestClientCABundleMerging(t *testing.T) {
	const namespace = "frontproxy-ca-tests"

	testcases := []struct {
		name                   string
		rootShard              *operatorv1alpha1.RootShard
		frontProxy             *operatorv1alpha1.FrontProxy
		extraSecrets           []*corev1.Secret
		expectedMergedContents []string
	}{
		{
			name: "without any clientCABundleRef merged secret contains only ClientCA",
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
			frontProxy: &operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fronty-no-bundle",
					Namespace: namespace,
				},
				Spec: operatorv1alpha1.FrontProxySpec{
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{
							Name: "rooty-no-bundle",
						},
					},
				},
			},
			expectedMergedContents: []string{"RootClientCA"},
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
			frontProxy: &operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fronty-inherits",
					Namespace: namespace,
				},
				Spec: operatorv1alpha1.FrontProxySpec{
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{
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
			expectedMergedContents: []string{"RootClientCA", "RootShardExtraCA"},
		},
		{
			name: "with frontProxy clientCABundleRef only, merged secret contains ClientCA and FrontProxy bundle",
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
			frontProxy: &operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fronty-own-bundle",
					Namespace: namespace,
				},
				Spec: operatorv1alpha1.FrontProxySpec{
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{
							Name: "rooty-plain",
						},
					},
					ClientCABundleRef: &corev1.LocalObjectReference{
						Name: "frontproxy-extra-ca",
					},
				},
			},
			extraSecrets: []*corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "frontproxy-extra-ca",
						Namespace: namespace,
					},
					Data: map[string][]byte{
						"tls.crt": []byte("-----BEGIN CERTIFICATE-----\nFrontProxyExtraCA\n-----END CERTIFICATE-----"),
					},
				},
			},
			expectedMergedContents: []string{"RootClientCA", "FrontProxyExtraCA"},
		},
		{
			name: "with both rootShard and frontProxy clientCABundleRef, merged secret contains all three CAs",
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
			frontProxy: &operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fronty-both",
					Namespace: namespace,
				},
				Spec: operatorv1alpha1.FrontProxySpec{
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{
							Name: "rooty-both",
						},
					},
					ClientCABundleRef: &corev1.LocalObjectReference{
						Name: "frontproxy-ca-both",
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
						Name:      "frontproxy-ca-both",
						Namespace: namespace,
					},
					Data: map[string][]byte{
						"tls.crt": []byte("-----BEGIN CERTIFICATE-----\nFrontProxyExtraCA\n-----END CERTIFICATE-----"),
					},
				},
			},
			expectedMergedContents: []string{"RootClientCA", "RootShardExtraCA", "FrontProxyExtraCA"},
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

			objects := []ctrlruntimeclient.Object{tc.rootShard, tc.frontProxy, clientCASecret}
			for _, s := range tc.extraSecrets {
				objects = append(objects, s)
			}

			client := ctrlruntimefakeclient.
				NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(tc.rootShard, tc.frontProxy).
				WithObjects(objects...).
				Build()

			ctx := context.Background()

			controllerReconciler := &FrontProxyReconciler{
				Client: client,
				Scheme: client.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(tc.frontProxy),
			})
			require.NoError(t, err)

			// The merged client CA secret is always created for FrontProxy
			mergedSecret := &corev1.Secret{}
			mergedSecretName := tc.frontProxy.Name + "-merged-client-ca"
			err = client.Get(ctx, ctrlruntimeclient.ObjectKey{
				Name:      mergedSecretName,
				Namespace: namespace,
			}, mergedSecret)
			require.NoError(t, err, "merged client CA secret should exist")
			require.NotNil(t, mergedSecret.Data["tls.crt"], "merged secret should contain tls.crt")

			mergedData := string(mergedSecret.Data["tls.crt"])
			for _, expected := range tc.expectedMergedContents {
				require.Contains(t, mergedData, expected,
					"merged CA should contain %s", expected)
			}
		})
	}
}
