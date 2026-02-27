/*
Copyright 2026 The kcp Authors.

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

package cacheserver

import (
	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"

	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/utils"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// RootCACertificateReconciler creates a standalone CA just for a single cache-server.
func RootCACertificateReconciler(server *operatorv1alpha1.CacheServer) reconciling.NamedCertificateReconcilerFactory {
	name := resources.GetCacheServerCAName(server.Name, operatorv1alpha1.RootCA)
	template := server.Spec.CertificateTemplates.CATemplate(operatorv1alpha1.RootCA)

	if server.Spec.Certificates.IssuerRef == nil {
		panic("RootCACertificateReconciler must not be called if no issuerRef is specified.")
	}

	return func() (string, reconciling.CertificateReconciler) {
		return name, func(cert *certmanagerv1.Certificate) (*certmanagerv1.Certificate, error) {
			cert.SetLabels(resources.GetCacheServerResourceLabels(server))

			cert.Spec = certmanagerv1.CertificateSpec{
				IsCA:       true,
				CommonName: name,
				SecretName: name,
				SecretTemplate: &certmanagerv1.CertificateSecretTemplate{
					Labels: map[string]string{
						resources.CacheServerLabel: server.Name,
					},
				},
				Duration:    &operatorv1alpha1.DefaultCADuration,
				RenewBefore: &operatorv1alpha1.DefaultCARenewal,

				PrivateKey: &certmanagerv1.CertificatePrivateKey{
					Algorithm: certmanagerv1.RSAKeyAlgorithm,
					Size:      4096,
				},

				IssuerRef: certmanagermetav1.ObjectReference{
					Name:  server.Spec.Certificates.IssuerRef.Name,
					Kind:  server.Spec.Certificates.IssuerRef.Kind,
					Group: server.Spec.Certificates.IssuerRef.Group,
				},
			}

			return utils.ApplyCertificateTemplate(cert, &template), nil
		}
	}
}

func ServerCertificateReconciler(server *operatorv1alpha1.CacheServer) reconciling.NamedCertificateReconcilerFactory {
	const certKind = operatorv1alpha1.ServerCertificate

	name := resources.GetCacheServerCertificateName(server, certKind)
	template := server.Spec.CertificateTemplates.CertificateTemplate(certKind)

	return func() (string, reconciling.CertificateReconciler) {
		return name, func(cert *certmanagerv1.Certificate) (*certmanagerv1.Certificate, error) {
			cert.SetLabels(resources.GetCacheServerResourceLabels(server))
			cert.Spec = certmanagerv1.CertificateSpec{
				SecretName: name,
				SecretTemplate: &certmanagerv1.CertificateSecretTemplate{
					Labels: map[string]string{
						resources.CacheServerLabel: server.Name,
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
					resources.GetCacheServerBaseHost(server),
				},

				IssuerRef: certmanagermetav1.ObjectReference{
					Name:  resources.GetCacheServerCAName(server.Name, operatorv1alpha1.RootCA),
					Kind:  "Issuer",
					Group: "cert-manager.io",
				},
			}

			return utils.ApplyCertificateTemplate(cert, &template), nil
		}
	}
}
