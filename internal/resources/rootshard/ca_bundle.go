/*
Copyright 2025 The kcp Authors.

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

package rootshard

import (
	"context"
	"fmt"

	k8creconciling "k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func MergedCABundleSecretReconciler(ctx context.Context, rootShard *operatorv1alpha1.RootShard, kubeClient ctrlruntimeclient.Client) k8creconciling.NamedSecretReconcilerFactory {
	return func() (string, k8creconciling.SecretReconciler) {
		secretName := fmt.Sprintf("%s-merged-ca-bundle", rootShard.Name)
		return secretName, func(secret *corev1.Secret) (*corev1.Secret, error) {
			if secret.Data == nil {
				secret.Data = make(map[string][]byte)
			}

			// Get ServerCA certificate
			serverCASecret := &corev1.Secret{}
			serverCASecretName := resources.GetRootShardCAName(rootShard, operatorv1alpha1.ServerCA)
			err := kubeClient.Get(ctx, types.NamespacedName{
				Name:      serverCASecretName,
				Namespace: rootShard.Namespace,
			}, serverCASecret)
			if err != nil {
				return nil, fmt.Errorf("failed to get ServerCA secret %s: %w", serverCASecretName, err)
			}

			serverCACert, exists := serverCASecret.Data["tls.crt"]
			if !exists {
				return nil, fmt.Errorf("ServerCA secret %s missing tls.crt", serverCASecretName)
			}

			// Get user-provided CA bundle if specified
			var userCABundle []byte
			if rootShard.Spec.CABundleSecretRef != nil {
				userCABundleSecret := &corev1.Secret{}
				err := kubeClient.Get(ctx, types.NamespacedName{
					Name:      rootShard.Spec.CABundleSecretRef.Name,
					Namespace: rootShard.Namespace,
				}, userCABundleSecret)
				if err != nil {
					return nil, fmt.Errorf("failed to get user CA bundle secret %s: %w", rootShard.Spec.CABundleSecretRef.Name, err)
				}

				var exists bool
				userCABundle, exists = userCABundleSecret.Data["tls.crt"]
				if !exists {
					return nil, fmt.Errorf("user CA bundle secret %s missing tls.crt", rootShard.Spec.CABundleSecretRef.Name)
				}
			}

			// Merge certificates: ServerCA + user CA bundle
			var mergedCA []byte
			if len(userCABundle) > 0 {
				mergedCA = append(serverCACert, '\n')
				mergedCA = append(mergedCA, userCABundle...)
			} else {
				mergedCA = serverCACert
			}

			secret.Data["tls.crt"] = mergedCA

			// Set labels to identify this as a merged CA bundle
			if secret.Labels == nil {
				secret.Labels = make(map[string]string)
			}
			secret.Labels[resources.RootShardLabel] = rootShard.Name

			return secret, nil
		}
	}
}
