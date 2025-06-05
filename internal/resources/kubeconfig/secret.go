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

package kubeconfig

import (
	"fmt"
	"net/url"

	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func KubeconfigSecretReconciler(
	kubeconfig *operatorv1alpha1.Kubeconfig,
	rootShard *operatorv1alpha1.RootShard,
	shard *operatorv1alpha1.Shard,
	caSecret, certSecret *corev1.Secret,
) (reconciling.NamedSecretReconcilerFactory, error) {
	config := &clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{},
		Contexts: map[string]*clientcmdapi.Context{},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			kubeconfig.Spec.Username: {
				ClientCertificateData: certSecret.Data["tls.crt"],
				ClientKeyData:         certSecret.Data["tls.key"],
			},
		},
	}

	addCluster := func(name, url string) {
		config.Clusters[name] = &clientcmdapi.Cluster{
			Server:                   url,
			CertificateAuthorityData: caSecret.Data["tls.crt"],
		}
		config.Contexts[name] = &clientcmdapi.Context{
			Cluster:  name,
			AuthInfo: kubeconfig.Spec.Username,
		}
	}

	switch {
	case kubeconfig.Spec.Target.RootShardRef != nil:
		if rootShard == nil {
			panic("RootShard must be provided when kubeconfig targets one.")
		}

		serverURL := resources.GetRootShardBaseURL(rootShard)
		defaultURL, err := url.JoinPath(serverURL, "clusters", "root")
		if err != nil {
			return nil, err
		}

		addCluster("default", defaultURL)
		addCluster("shard-base", serverURL)
		config.CurrentContext = "default"

	case kubeconfig.Spec.Target.ShardRef != nil:
		if shard == nil {
			panic("Shard must be provided when kubeconfig targets one.")
		}

		serverURL := resources.GetShardBaseURL(shard)

		addCluster("default", serverURL)
		addCluster("shard-base", serverURL)
		config.CurrentContext = "default"

	case kubeconfig.Spec.Target.FrontProxyRef != nil:
		if rootShard == nil {
			panic("RootShard must be provided when kubeconfig targets a FrontProxy.")
		}

		serverURL := fmt.Sprintf("https://%s:6443", rootShard.Spec.External.Hostname)
		defaultURL, err := url.JoinPath(serverURL, "clusters", "root")
		if err != nil {
			return nil, err
		}

		addCluster("default", defaultURL)
		config.CurrentContext = "default"

	default:
		panic("Called reconciler for an invalid kubeconfig, this should not have happened.")
	}

	return func() (string, reconciling.SecretReconciler) {
		return kubeconfig.Spec.SecretRef.Name, func(secret *corev1.Secret) (*corev1.Secret, error) {
			if secret.Data == nil {
				secret.Data = make(map[string][]byte)
			}

			data, err := clientcmd.Write(*config)
			if err != nil {
				return nil, err
			}

			secret.Data["kubeconfig"] = data

			return secret, nil
		}
	}, nil
}
