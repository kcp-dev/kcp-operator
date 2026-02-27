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

package kubeconfig

import (
	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/utils"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func ClientCertificateReconciler(kubeConfig *operatorv1alpha1.Kubeconfig, issuerName string) reconciling.NamedCertificateReconcilerFactory {
	orgs := sets.New(kubeConfig.Spec.Groups...)
	orgs.Insert(KubeconfigGroup(kubeConfig))

	return func() (string, reconciling.CertificateReconciler) {
		return kubeConfig.GetCertificateName(), func(cert *certmanagerv1.Certificate) (*certmanagerv1.Certificate, error) {
			cert.SetLabels(kubeConfig.Labels)
			cert.Spec = certmanagerv1.CertificateSpec{
				SecretName: kubeConfig.GetCertificateName(),
				SecretTemplate: &certmanagerv1.CertificateSecretTemplate{
					Labels: map[string]string{
						resources.KubeconfigLabel: kubeConfig.Name,
					},
				},
				Duration: &kubeConfig.Spec.Validity,

				PrivateKey: &certmanagerv1.CertificatePrivateKey{
					Algorithm: certmanagerv1.RSAKeyAlgorithm,
					Size:      4096,
				},

				Usages: []certmanagerv1.KeyUsage{
					certmanagerv1.UsageClientAuth,
				},

				CommonName: kubeConfig.Spec.Username,
				Subject: &certmanagerv1.X509Subject{
					Organizations: sets.List(orgs),
				},

				IssuerRef: certmanagermetav1.ObjectReference{
					Name:  issuerName,
					Kind:  "Issuer",
					Group: "cert-manager.io",
				},
			}

			return utils.ApplyCertificateTemplate(cert, kubeConfig.Spec.CertificateTemplate), nil
		}
	}
}
