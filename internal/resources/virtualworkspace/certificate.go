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

package virtualworkspace

import (
	"fmt"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"

	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/utils"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func commonName(vw *operatorv1alpha1.VirtualWorkspace) string {
	return fmt.Sprintf("%s-virtual-workspace", vw.Name)
}

func ClientCertificateReconciler(vw *operatorv1alpha1.VirtualWorkspace, issuerName string) reconciling.NamedCertificateReconcilerFactory {
	const certKind = operatorv1alpha1.ClientCertificate

	template := vw.Spec.CertificateTemplates.CertificateTemplate(certKind)
	name := resources.GetVirtualWorkspaceCertificateName(vw, certKind)

	return func() (string, reconciling.CertificateReconciler) {
		return name, func(cert *certmanagerv1.Certificate) (*certmanagerv1.Certificate, error) {
			cert.SetLabels(vw.Labels)
			cert.Spec = certmanagerv1.CertificateSpec{
				SecretName: name,
				SecretTemplate: &certmanagerv1.CertificateSecretTemplate{
					Labels: map[string]string{
						resources.VirtualWorkspaceLabel: vw.Name,
					},
				},

				Duration:    &operatorv1alpha1.DefaultCertificateDuration,
				RenewBefore: &operatorv1alpha1.DefaultCertificateRenewal,

				PrivateKey: &certmanagerv1.CertificatePrivateKey{
					Algorithm: certmanagerv1.RSAKeyAlgorithm,
					Size:      4096,
				},

				Usages: []certmanagerv1.KeyUsage{
					certmanagerv1.UsageClientAuth,
				},

				CommonName: commonName(vw),

				// The VW needs system:masters to access the kcp APIs.
				Subject: &certmanagerv1.X509Subject{
					Organizations: []string{"system:masters"},
				},

				IssuerRef: certmanagermetav1.ObjectReference{
					Name:  issuerName,
					Kind:  "Issuer",
					Group: "cert-manager.io",
				},
			}

			return utils.ApplyCertificateTemplate(cert, &template), nil
		}
	}
}

func ServerCertificateReconciler(vw *operatorv1alpha1.VirtualWorkspace, rootShard *operatorv1alpha1.RootShard) reconciling.NamedCertificateReconcilerFactory {
	const certKind = operatorv1alpha1.ServerCertificate

	name := resources.GetVirtualWorkspaceCertificateName(vw, certKind)
	template := vw.Spec.CertificateTemplates.CertificateTemplate(certKind)

	return func() (string, reconciling.CertificateReconciler) {
		return name, func(cert *certmanagerv1.Certificate) (*certmanagerv1.Certificate, error) {
			cert.SetLabels(resources.GetVirtualWorkspaceResourceLabels(vw))
			cert.Spec = certmanagerv1.CertificateSpec{
				SecretName: name,
				SecretTemplate: &certmanagerv1.CertificateSecretTemplate{
					Labels: map[string]string{
						resources.VirtualWorkspaceLabel: vw.Name,
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
					certmanagerv1.UsageKeyEncipherment,
					certmanagerv1.UsageDigitalSignature,
				},

				DNSNames: []string{
					"localhost",
					resources.GetVirtualWorkspaceBaseHost(vw),
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
