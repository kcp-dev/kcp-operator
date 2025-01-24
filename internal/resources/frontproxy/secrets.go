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

package frontproxy

import (
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"sigs.k8s.io/yaml"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

const (
	ClientCertPath        = FrontProxyBasepath + "/kubeconfig-client-cert"
	ClientCertificatePath = ClientCertPath + "/tls.crt"
	ClientKeyPath         = ClientCertPath + "/tls.key"
	KubeconfigCAPath      = "/etc/kcp/tls/ca/tls.crt"
)

func DynamicKubeconfigSecretReconciler(frontproxy *operatorv1alpha1.FrontProxy, rootshard *operatorv1alpha1.RootShard) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return resources.GetFrontProxyDynamicKubeconfigName(rootshard, frontproxy), func(obj *corev1.Secret) (*corev1.Secret, error) {
			obj.SetLabels(resources.GetFrontProxyResourceLabels(frontproxy))

			kubeconfig := clientcmdv1.Config{
				Clusters: []clientcmdv1.NamedCluster{
					{
						Name: "system:admin",
						Cluster: clientcmdv1.Cluster{
							CertificateAuthority: KubeconfigCAPath,
							Server:               resources.GetRootShardBaseURL(rootshard),
						},
					},
				},
				Contexts: []clientcmdv1.NamedContext{
					{
						Name: "system:admin",
						Context: clientcmdv1.Context{
							Cluster:  "system:admin",
							AuthInfo: "admin",
						},
					},
				},
				CurrentContext: "system:admin",
				AuthInfos: []clientcmdv1.NamedAuthInfo{
					{
						Name: "admin",
						AuthInfo: clientcmdv1.AuthInfo{
							ClientCertificate: ClientCertificatePath,
							ClientKey:         ClientKeyPath,
						},
					},
				},
			}

			var b []byte
			var err error
			if b, err = yaml.Marshal(kubeconfig); err != nil {
				return nil, err
			}

			obj.Data = map[string][]byte{
				"kubeconfig": b,
			}

			return obj, nil
		}
	}
}
