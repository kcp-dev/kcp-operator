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

package kubeconfig

import (
	"fmt"

	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/kcp-dev/kcp-operator/api/v1alpha1"
)

func KubeconfigSecretReconciler(kubeconfig *v1alpha1.Kubeconfig, certSecret *corev1.Secret, serverName, serverUrl string) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return kubeconfig.Spec.SecretRef.Name, func(secret *corev1.Secret) (*corev1.Secret, error) {
			var config *clientcmdapi.Config

			if secret.Data == nil {
				secret.Data = make(map[string][]byte)
			}

			config = &clientcmdapi.Config{}

			config.Clusters = map[string]*clientcmdapi.Cluster{
				serverName: {
					Server:                   serverUrl,
					CertificateAuthorityData: certSecret.Data["ca.crt"],
				},
			}

			contextName := fmt.Sprintf("%s:%s", serverName, kubeconfig.Spec.Username)

			config.Contexts = map[string]*clientcmdapi.Context{
				contextName: {
					Cluster:  serverName,
					AuthInfo: kubeconfig.Spec.Username,
				},
			}
			config.AuthInfos = map[string]*clientcmdapi.AuthInfo{
				kubeconfig.Spec.Username: {
					ClientCertificateData: certSecret.Data["tls.crt"],
					ClientKeyData:         certSecret.Data["tls.key"],
				},
			}
			config.CurrentContext = contextName

			data, err := clientcmd.Write(*config)
			if err != nil {
				return nil, err
			}

			secret.Data["kubeconfig"] = data

			return secret, nil
		}
	}
}
