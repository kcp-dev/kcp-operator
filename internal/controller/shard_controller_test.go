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

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

var _ = Describe("Shard Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		shard := &operatorv1alpha1.Shard{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind RootShard")
			err := k8sClient.Get(ctx, typeNamespacedName, &operatorv1alpha1.RootShard{})
			if err != nil && apierrors.IsNotFound(err) {
				resource := &operatorv1alpha1.RootShard{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-root-shard",
						Namespace: "default",
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
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}

			By("creating the custom resource for the Kind Shard")
			err = k8sClient.Get(ctx, typeNamespacedName, shard)
			if err != nil && apierrors.IsNotFound(err) {
				resource := &operatorv1alpha1.Shard{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: operatorv1alpha1.ShardSpec{
						CommonShardSpec: operatorv1alpha1.CommonShardSpec{
							Etcd: operatorv1alpha1.EtcdConfig{
								Endpoints: []string{"https://localhost:2379"},
							},
						},
						RootShard: operatorv1alpha1.RootShardConfig{
							Reference: &corev1.LocalObjectReference{
								Name: "my-root-shard",
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &operatorv1alpha1.Shard{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Shard")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ShardReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})
