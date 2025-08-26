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

package rootshard

import (
	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"

	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/utils"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func ServerCertificateReconciler(rootShard *operatorv1alpha1.RootShard) reconciling.NamedCertificateReconcilerFactory {
	const certKind = operatorv1alpha1.ServerCertificate

	name := resources.GetRootShardCertificateName(rootShard, certKind)
	template := rootShard.Spec.CertificateTemplates.CertificateTemplate(certKind)

	return func() (string, reconciling.CertificateReconciler) {
		return name, func(cert *certmanagerv1.Certificate) (*certmanagerv1.Certificate, error) {
			cert.SetLabels(resources.GetRootShardResourceLabels(rootShard))
			cert.Spec = certmanagerv1.CertificateSpec{
				SecretName: name,
				SecretTemplate: &certmanagerv1.CertificateSecretTemplate{
					Labels: map[string]string{
						resources.RootShardLabel: rootShard.Name,
					},
				},
				Duration:    &operatorv1alpha1.DefaultCertificateDuration,
				RenewBefore: &operatorv1alpha1.DefaultCertificateRenewal,

				PrivateKey: &certmanagerv1.CertificatePrivateKey{
					Algorithm: certmanagerv1.RSAKeyAlgorithm,
					Size:      4096,
				},

				Usages: []certmanagerv1.KeyUsage{
					certmanagerv1.UsageServerAuth,
				},

				DNSNames: []string{
					resources.GetRootShardBaseHost(rootShard),
					rootShard.Spec.External.Hostname,
				},

				IssuerRef: certmanagermetav1.ObjectReference{
					Name:  resources.GetRootShardCAName(rootShard, operatorv1alpha1.ServerCA),
					Kind:  "Issuer",
					Group: "cert-manager.io",
				},
			}

			return utils.ApplyCertificateTemplate(cert, &template), nil
		}
	}
}

func VirtualWorkspacesCertificateReconciler(rootShard *operatorv1alpha1.RootShard) reconciling.NamedCertificateReconcilerFactory {
	const certKind = operatorv1alpha1.VirtualWorkspacesCertificate

	name := resources.GetRootShardCertificateName(rootShard, certKind)
	template := rootShard.Spec.CertificateTemplates.CertificateTemplate(certKind)

	return func() (string, reconciling.CertificateReconciler) {
		return name, func(cert *certmanagerv1.Certificate) (*certmanagerv1.Certificate, error) {
			cert.SetLabels(resources.GetRootShardResourceLabels(rootShard))
			cert.Spec = certmanagerv1.CertificateSpec{
				SecretName: name,
				SecretTemplate: &certmanagerv1.CertificateSecretTemplate{
					Labels: map[string]string{
						resources.RootShardLabel: rootShard.Name,
					},
				},
				Duration:    &operatorv1alpha1.DefaultCertificateDuration,
				RenewBefore: &operatorv1alpha1.DefaultCertificateRenewal,

				PrivateKey: &certmanagerv1.CertificatePrivateKey{
					Algorithm: certmanagerv1.RSAKeyAlgorithm,
					Size:      4096,
				},

				Usages: []certmanagerv1.KeyUsage{
					certmanagerv1.UsageServerAuth,
				},

				DNSNames: []string{
					rootShard.Spec.External.Hostname,
				},

				IssuerRef: certmanagermetav1.ObjectReference{
					Name:  resources.GetRootShardCAName(rootShard, operatorv1alpha1.ServerCA),
					Kind:  "Issuer",
					Group: "cert-manager.io",
				},
			}

			return utils.ApplyCertificateTemplate(cert, &template), nil
		}
	}
}

func ServiceAccountCertificateReconciler(rootShard *operatorv1alpha1.RootShard) reconciling.NamedCertificateReconcilerFactory {
	const certKind = operatorv1alpha1.ServiceAccountCertificate

	name := resources.GetRootShardCertificateName(rootShard, certKind)
	template := rootShard.Spec.CertificateTemplates.CertificateTemplate(certKind)

	return func() (string, reconciling.CertificateReconciler) {
		return name, func(cert *certmanagerv1.Certificate) (*certmanagerv1.Certificate, error) {
			cert.SetLabels(resources.GetRootShardResourceLabels(rootShard))
			cert.Spec = certmanagerv1.CertificateSpec{
				CommonName: name,
				SecretName: name,
				SecretTemplate: &certmanagerv1.CertificateSecretTemplate{
					Labels: map[string]string{
						resources.RootShardLabel: rootShard.Name,
					},
				},
				Duration:    &operatorv1alpha1.DefaultCertificateDuration,
				RenewBefore: &operatorv1alpha1.DefaultCertificateRenewal,

				PrivateKey: &certmanagerv1.CertificatePrivateKey{
					Algorithm: certmanagerv1.RSAKeyAlgorithm,
					Size:      4096,
				},

				IssuerRef: certmanagermetav1.ObjectReference{
					Name:  resources.GetRootShardCAName(rootShard, operatorv1alpha1.ServiceAccountCA),
					Kind:  "Issuer",
					Group: "cert-manager.io",
				},
			}

			return utils.ApplyCertificateTemplate(cert, &template), nil
		}
	}
}

func LogicalClusterAdminCertificateReconciler(rootShard *operatorv1alpha1.RootShard) reconciling.NamedCertificateReconcilerFactory {
	const certKind = operatorv1alpha1.LogicalClusterAdminCertificate

	name := resources.GetRootShardCertificateName(rootShard, certKind)
	template := rootShard.Spec.CertificateTemplates.CertificateTemplate(certKind)

	return func() (string, reconciling.CertificateReconciler) {
		return name, func(cert *certmanagerv1.Certificate) (*certmanagerv1.Certificate, error) {
			cert.SetLabels(resources.GetRootShardResourceLabels(rootShard))
			cert.Spec = certmanagerv1.CertificateSpec{
				CommonName:  "logical-cluster-admin",
				SecretName:  name,
				Duration:    &operatorv1alpha1.DefaultCertificateDuration,
				RenewBefore: &operatorv1alpha1.DefaultCertificateRenewal,

				PrivateKey: &certmanagerv1.CertificatePrivateKey{
					Algorithm: certmanagerv1.RSAKeyAlgorithm,
					Size:      4096,
				},

				Subject: &certmanagerv1.X509Subject{
					Organizations: []string{"system:kcp:logical-cluster-admin"},
				},

				Usages: []certmanagerv1.KeyUsage{
					certmanagerv1.UsageClientAuth,
				},

				IssuerRef: certmanagermetav1.ObjectReference{
					Name:  resources.GetRootShardCAName(rootShard, operatorv1alpha1.ClientCA),
					Kind:  "Issuer",
					Group: "cert-manager.io",
				},
			}

			return utils.ApplyCertificateTemplate(cert, &template), nil
		}
	}
}

func ExternalLogicalClusterAdminCertificateReconciler(rootShard *operatorv1alpha1.RootShard) reconciling.NamedCertificateReconcilerFactory {
	const certKind = operatorv1alpha1.ExternalLogicalClusterAdminCertificate

	name := resources.GetRootShardCertificateName(rootShard, certKind)
	template := rootShard.Spec.CertificateTemplates.CertificateTemplate(certKind)

	return func() (string, reconciling.CertificateReconciler) {
		return name, func(cert *certmanagerv1.Certificate) (*certmanagerv1.Certificate, error) {
			cert.SetLabels(resources.GetRootShardResourceLabels(rootShard))
			cert.Spec = certmanagerv1.CertificateSpec{
				CommonName:  "external-logical-cluster-admin",
				SecretName:  name,
				Duration:    &operatorv1alpha1.DefaultCertificateDuration,
				RenewBefore: &operatorv1alpha1.DefaultCertificateRenewal,

				PrivateKey: &certmanagerv1.CertificatePrivateKey{
					Algorithm: certmanagerv1.RSAKeyAlgorithm,
					Size:      4096,
				},

				Subject: &certmanagerv1.X509Subject{
					Organizations: []string{"system:kcp:external-logical-cluster-admin"},
				},

				Usages: []certmanagerv1.KeyUsage{
					certmanagerv1.UsageClientAuth,
				},

				IssuerRef: certmanagermetav1.ObjectReference{
					Name:  resources.GetRootShardCAName(rootShard, operatorv1alpha1.FrontProxyClientCA),
					Kind:  "Issuer",
					Group: "cert-manager.io",
				},
			}

			return utils.ApplyCertificateTemplate(cert, &template), nil
		}
	}
}

// OperatorClientCertificateReconciler reconciles a client certificate that is used both
// for bootstrapping (root)shards (in this case the system:masters organization in it gives
// it permissions) and subsequently by other controllers to provision resources inside
// workspaces (like RBAC for kubeconfigs), in which case it is assumed that a front-proxy
// will strip the system:masters group from the authInfo, but pass on the commonName
// untouched. And the common name is bootstrapped to have all necessary permissions.
func OperatorClientCertificateReconciler(rootShard *operatorv1alpha1.RootShard) reconciling.NamedCertificateReconcilerFactory {
	const certKind = operatorv1alpha1.OperatorCertificate

	name := resources.GetRootShardCertificateName(rootShard, certKind)
	template := rootShard.Spec.CertificateTemplates.CertificateTemplate(certKind)

	return func() (string, reconciling.CertificateReconciler) {
		return name, func(cert *certmanagerv1.Certificate) (*certmanagerv1.Certificate, error) {
			cert.SetLabels(resources.GetRootShardResourceLabels(rootShard))
			cert.Spec = certmanagerv1.CertificateSpec{
				CommonName:  resources.OperatorUsername,
				SecretName:  name,
				Duration:    &operatorv1alpha1.DefaultCertificateDuration,
				RenewBefore: &operatorv1alpha1.DefaultCertificateRenewal,

				PrivateKey: &certmanagerv1.CertificatePrivateKey{
					Algorithm: certmanagerv1.RSAKeyAlgorithm,
					Size:      4096,
				},

				Subject: &certmanagerv1.X509Subject{
					// The operator needs access to system clusters like system:admin.
					Organizations: []string{"system:masters"},
				},

				Usages: []certmanagerv1.KeyUsage{
					certmanagerv1.UsageClientAuth,
				},

				IssuerRef: certmanagermetav1.ObjectReference{
					Name:  resources.GetRootShardCAName(rootShard, operatorv1alpha1.ClientCA),
					Kind:  "Issuer",
					Group: "cert-manager.io",
				},
			}

			return utils.ApplyCertificateTemplate(cert, &template), nil
		}
	}
}
