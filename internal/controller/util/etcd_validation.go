/*
Copyright 2025 The KCP Authors.

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

package util

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// ValidateEtcdTLSConfig validates the etcd TLS configuration and ensures the referenced secret exists.
func ValidateEtcdTLSConfig(ctx context.Context, client ctrlruntimeclient.Client, etcdConfig operatorv1alpha1.EtcdConfig, namespace string) error {
	if etcdConfig.TLSConfig == nil {
		return nil
	}

	tlsConfig := etcdConfig.TLSConfig

	if tlsConfig.SecretRef == nil {
		return fmt.Errorf("etcd TLS configuration must specify secretRef")
	}

	secretName := tlsConfig.SecretRef.Name
	secretNamespace := tlsConfig.SecretRef.Namespace

	if secretNamespace == "" {
		secretNamespace = namespace
	}

	secret := &corev1.Secret{}
	if err := client.Get(ctx, types.NamespacedName{Name: secretName, Namespace: secretNamespace}, secret); err != nil {
		return fmt.Errorf("failed to get etcd TLS secret %s/%s: %w", secretNamespace, secretName, err)
	}

	requiredKeys := []string{"tls.crt", "tls.key", "ca.crt"}
	for _, key := range requiredKeys {
		if _, exists := secret.Data[key]; !exists {
			return fmt.Errorf("etcd TLS secret %s/%s is missing required key: %s", secretNamespace, secretName, key)
		}
	}

	return nil
}
