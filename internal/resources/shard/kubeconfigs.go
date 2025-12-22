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

package shard

import (
	"fmt"

	k8creconciling "k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func kubeconfigSecret(shard *operatorv1alpha1.Shard, cert operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s-kubeconfig", shard.Name, cert)
}

func RootShardClientKubeconfigReconciler(shard *operatorv1alpha1.Shard, rootShard *operatorv1alpha1.RootShard) k8creconciling.NamedSecretReconcilerFactory {
	const (
		serverName   = "root-shard"
		contextName  = "shard-base" // hardcoded in kcp
		authInfoName = "shard"
	)

	return func() (string, k8creconciling.SecretReconciler) {
		return kubeconfigSecret(shard, operatorv1alpha1.ClientCertificate), func(secret *corev1.Secret) (*corev1.Secret, error) {
			var config *clientcmdapi.Config

			if secret.Data == nil {
				secret.Data = make(map[string][]byte)
			}

			config = &clientcmdapi.Config{
				Clusters: map[string]*clientcmdapi.Cluster{
					serverName: {
						Server:               resources.GetRootShardBaseURL(rootShard),
						CertificateAuthority: getCAMountPath(operatorv1alpha1.ServerCA) + "/tls.crt",
					},
				},
				Contexts: map[string]*clientcmdapi.Context{
					contextName: {
						Cluster:  serverName,
						AuthInfo: authInfoName,
					},
				},
				AuthInfos: map[string]*clientcmdapi.AuthInfo{
					authInfoName: {
						ClientCertificate: getCertificateMountPath(operatorv1alpha1.ClientCertificate) + "/tls.crt",
						ClientKey:         getCertificateMountPath(operatorv1alpha1.ClientCertificate) + "/tls.key",
					},
				},
				CurrentContext: contextName,
			}

			data, err := clientcmd.Write(*config)
			if err != nil {
				return nil, err
			}

			secret.Data["kubeconfig"] = data

			return secret, nil
		}
	}
}

func LogicalClusterAdminKubeconfigReconciler(shard *operatorv1alpha1.Shard, rootShard *operatorv1alpha1.RootShard) k8creconciling.NamedSecretReconcilerFactory {
	const (
		serverName   = "logical-cluster:admin"
		contextName  = "logical-cluster"
		authInfoName = "logical-cluster-admin"
	)

	return func() (string, k8creconciling.SecretReconciler) {
		return kubeconfigSecret(shard, operatorv1alpha1.LogicalClusterAdminCertificate), func(secret *corev1.Secret) (*corev1.Secret, error) {
			var config *clientcmdapi.Config

			if secret.Data == nil {
				secret.Data = make(map[string][]byte)
			}

			config = &clientcmdapi.Config{
				Clusters: map[string]*clientcmdapi.Cluster{
					serverName: {
						Server:               resources.GetRootShardBaseURL(rootShard),
						CertificateAuthority: getCAMountPath(operatorv1alpha1.ServerCA) + "/tls.crt",
					},
				},
				Contexts: map[string]*clientcmdapi.Context{
					contextName: {
						Cluster:  serverName,
						AuthInfo: authInfoName,
					},
				},
				AuthInfos: map[string]*clientcmdapi.AuthInfo{
					authInfoName: {
						ClientCertificate: getCertificateMountPath(operatorv1alpha1.LogicalClusterAdminCertificate) + "/tls.crt",
						ClientKey:         getCertificateMountPath(operatorv1alpha1.LogicalClusterAdminCertificate) + "/tls.key",
					},
				},
				CurrentContext: contextName,
			}

			data, err := clientcmd.Write(*config)
			if err != nil {
				return nil, err
			}

			secret.Data["kubeconfig"] = data

			return secret, nil
		}
	}
}

func ExternalLogicalClusterAdminKubeconfigReconciler(shard *operatorv1alpha1.Shard, rootShard *operatorv1alpha1.RootShard) k8creconciling.NamedSecretReconcilerFactory {
	const (
		serverName   = "external-logical-cluster:admin"
		contextName  = "external-logical-cluster"
		authInfoName = "external-logical-cluster-admin"
	)

	return func() (string, k8creconciling.SecretReconciler) {
		return kubeconfigSecret(shard, operatorv1alpha1.ExternalLogicalClusterAdminCertificate), func(secret *corev1.Secret) (*corev1.Secret, error) {
			var config *clientcmdapi.Config

			if secret.Data == nil {
				secret.Data = make(map[string][]byte)
			}

			config = &clientcmdapi.Config{
				Clusters: map[string]*clientcmdapi.Cluster{
					serverName: {
						Server: fmt.Sprintf("https://%s:%d", rootShard.Spec.External.Hostname, rootShard.Spec.External.Port),
						// CertificateAuthority will be configured below to respect CABundleSecretRef property in the shard spec
					},
				},
				Contexts: map[string]*clientcmdapi.Context{
					contextName: {
						Cluster:  serverName,
						AuthInfo: authInfoName,
					},
				},
				AuthInfos: map[string]*clientcmdapi.AuthInfo{
					authInfoName: {
						ClientCertificate: getCertificateMountPath(operatorv1alpha1.ExternalLogicalClusterAdminCertificate) + "/tls.crt",
						ClientKey:         getCertificateMountPath(operatorv1alpha1.ExternalLogicalClusterAdminCertificate) + "/tls.key",
					},
				},
				CurrentContext: contextName,
			}

			if shard.Spec.CABundleSecretRef == nil {
				config.Clusters[serverName].CertificateAuthority = getCAMountPath(operatorv1alpha1.ServerCA) + "/tls.crt"
			} else {
				config.Clusters[serverName].CertificateAuthority = getCAMountPath(operatorv1alpha1.CABundleCA) + "/tls.crt"
			}

			data, err := clientcmd.Write(*config)
			if err != nil {
				return nil, err
			}

			secret.Data["kubeconfig"] = data

			return secret, nil
		}
	}
}
