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

package frontproxy

import (
	"fmt"

	k8creconciling "k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/utils"
)

func (r *reconciler) clientCABundleSecretName() string {
	if r.frontProxy != nil {
		return fmt.Sprintf("%s-merged-client-ca", r.frontProxy.Name)
	}
	return fmt.Sprintf("%s-proxy-merged-client-ca", r.rootShard.Name)
}

// clientCABundleSecretReconciler creates a single secret with the
// shard ClientCA and an optional additional client CA bundle concatenated
// so that the front proxy accepts clients signed by either CA.
func (r *reconciler) clientCABundleSecretReconciler(clientCAs ...[]byte) k8creconciling.NamedSecretReconcilerFactory {
	return func() (string, k8creconciling.SecretReconciler) {
		return r.clientCABundleSecretName(), func(secret *corev1.Secret) (*corev1.Secret, error) {
			if secret.Data == nil {
				secret.Data = make(map[string][]byte)
			}

			secret.Data["tls.crt"] = utils.MergeCertificates(clientCAs...)

			if secret.Labels == nil {
				secret.Labels = make(map[string]string)
			}
			if r.frontProxy != nil {
				secret.Labels[resources.FrontProxyLabel] = r.frontProxy.Name
			} else {
				secret.Labels[resources.RootShardLabel] = r.rootShard.Name
			}

			return secret, nil
		}
	}
}

func (r *reconciler) backendCABundleSecretName() string {
	// Validate whether called for frontProxy or rootShardFrontProxy
	if r.frontProxy != nil {
		return fmt.Sprintf("%s-merged-ca-bundle", r.frontProxy.Name)
	}
	return fmt.Sprintf("%s-proxy-merged-ca-bundle", r.rootShard.Name)
}

func (r *reconciler) backendCABundleSecretReconciler(serverCACert, userCABundle []byte) k8creconciling.NamedSecretReconcilerFactory {
	return func() (string, k8creconciling.SecretReconciler) {
		return r.backendCABundleSecretName(), func(secret *corev1.Secret) (*corev1.Secret, error) {
			if secret.Data == nil {
				secret.Data = make(map[string][]byte)
			}

			secret.Data["tls.crt"] = utils.MergeCertificates(serverCACert, userCABundle)

			// Set labels to identify this as a merged CA bundle
			if secret.Labels == nil {
				secret.Labels = make(map[string]string)
			}
			if r.frontProxy != nil {
				secret.Labels[resources.FrontProxyLabel] = r.frontProxy.Name
			} else {
				secret.Labels[resources.RootShardLabel] = r.rootShard.Name
			}

			return secret, nil
		}
	}
}
