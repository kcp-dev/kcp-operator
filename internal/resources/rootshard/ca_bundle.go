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
	"github.com/kcp-dev/kcp-operator/internal/resources/utils"
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
			serverCACert, err := fetchTLSCert(ctx, kubeClient, rootShard.Namespace, resources.GetRootShardCAName(rootShard, operatorv1alpha1.ServerCA))
			if err != nil {
				return nil, fmt.Errorf("failed to get ServerCA: %w", err)
			}

			// Get user-provided CA bundle if specified
			var userCABundle []byte
			if rootShard.Spec.CABundleSecretRef != nil {
				userCABundle, err = fetchTLSCert(ctx, kubeClient, rootShard.Namespace, rootShard.Spec.CABundleSecretRef.Name)
				if err != nil {
					return nil, fmt.Errorf("failed to get user CA bundle: %w", err)
				}
			}

			secret.Data["tls.crt"] = utils.MergeCertificates(serverCACert, userCABundle)

			// Set labels to identify this as a merged CA bundle
			if secret.Labels == nil {
				secret.Labels = make(map[string]string)
			}
			secret.Labels[resources.RootShardLabel] = rootShard.Name

			return secret, nil
		}
	}
}

func MergedClientCABundleSecretReconciler(ctx context.Context, rootShard *operatorv1alpha1.RootShard, kubeClient ctrlruntimeclient.Client) k8creconciling.NamedSecretReconcilerFactory {
	return func() (string, k8creconciling.SecretReconciler) {
		secretName := fmt.Sprintf("%s-merged-client-ca", rootShard.Name)
		return secretName, func(secret *corev1.Secret) (*corev1.Secret, error) {
			if secret.Data == nil {
				secret.Data = make(map[string][]byte)
			}

			// Get ClientCA certificate
			clientCACert, err := fetchTLSCert(ctx, kubeClient, rootShard.Namespace, resources.GetRootShardCAName(rootShard, operatorv1alpha1.ClientCA))
			if err != nil {
				return nil, fmt.Errorf("failed to get ClientCA: %w", err)
			}

			// Get user-provided client CA bundle if specified
			var userClientCABundle []byte
			if rootShard.Spec.ClientCABundleRef != nil {
				userClientCABundle, err = fetchTLSCert(ctx, kubeClient, rootShard.Namespace, rootShard.Spec.ClientCABundleRef.Name)
				if err != nil {
					return nil, fmt.Errorf("failed to get user client CA bundle: %w", err)
				}
			}

			secret.Data["tls.crt"] = utils.MergeCertificates(clientCACert, userClientCABundle)

			// Set labels to identify this as a merged client CA bundle
			if secret.Labels == nil {
				secret.Labels = make(map[string]string)
			}
			secret.Labels[resources.RootShardLabel] = rootShard.Name

			return secret, nil
		}
	}
}

func fetchTLSCert(ctx context.Context, client ctrlruntimeclient.Client, namespace, secretName string) ([]byte, error) {
	secret := &corev1.Secret{}
	if err := client.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, secret); err != nil {
		return nil, fmt.Errorf("failed to get secret %s: %w", secretName, err)
	}

	data, exists := secret.Data["tls.crt"]
	if !exists {
		return nil, fmt.Errorf("secret %s missing tls.crt", secretName)
	}

	return data, nil
}
