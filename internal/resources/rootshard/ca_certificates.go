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
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// RootCACertificateReconciler creates the central CA used for the kcp setup around a specific RootShard. This shouldn't be called if the RootShard is configured to use a BYO CA certificate.
func RootCACertificateReconciler(rootShard *operatorv1alpha1.RootShard) reconciling.NamedCertificateReconcilerFactory {
	name := resources.GetRootShardCAName(rootShard, operatorv1alpha1.RootCA)

	if rootShard.Spec.Certificates.IssuerRef == nil {
		panic("RootCACertificateReconciler must not be called if not issuerRef is specified.")
	}

	return func() (string, reconciling.CertificateReconciler) {
		return name, func(cert *certmanagerv1.Certificate) (*certmanagerv1.Certificate, error) {
			cert.SetLabels(resources.GetRootShardResourceLabels(rootShard))

			cert.Spec = certmanagerv1.CertificateSpec{
				IsCA:       true,
				CommonName: name,
				SecretName: name,
				SecretTemplate: &certmanagerv1.CertificateSecretTemplate{
					Labels: map[string]string{
						resources.RootShardLabel: rootShard.Name,
					},
				},
				Duration:    &operatorv1alpha1.DefaultCADuration,
				RenewBefore: &operatorv1alpha1.DefaultCARenewal,

				PrivateKey: &certmanagerv1.CertificatePrivateKey{
					Algorithm: certmanagerv1.RSAKeyAlgorithm,
					Size:      4096,
				},

				IssuerRef: certmanagermetav1.ObjectReference{
					Name:  rootShard.Spec.Certificates.IssuerRef.Name,
					Kind:  rootShard.Spec.Certificates.IssuerRef.Kind,
					Group: rootShard.Spec.Certificates.IssuerRef.Group,
				},
			}

			return cert, nil
		}
	}
}

func CACertificateReconciler(rootShard *operatorv1alpha1.RootShard, ca operatorv1alpha1.CA) reconciling.NamedCertificateReconcilerFactory {
	name := resources.GetRootShardCAName(rootShard, ca)

	return func() (string, reconciling.CertificateReconciler) {
		return name, func(cert *certmanagerv1.Certificate) (*certmanagerv1.Certificate, error) {
			cert.SetLabels(resources.GetRootShardResourceLabels(rootShard))
			cert.Spec = certmanagerv1.CertificateSpec{
				IsCA:       true,
				CommonName: name,
				SecretName: name,
				SecretTemplate: &certmanagerv1.CertificateSecretTemplate{
					Labels: map[string]string{
						resources.RootShardLabel: rootShard.Name,
					},
				},
				Duration:    &operatorv1alpha1.DefaultCADuration,
				RenewBefore: &operatorv1alpha1.DefaultCARenewal,

				PrivateKey: &certmanagerv1.CertificatePrivateKey{
					Algorithm: certmanagerv1.RSAKeyAlgorithm,
					Size:      4096,
				},

				IssuerRef: certmanagermetav1.ObjectReference{
					Name:  resources.GetRootShardCAName(rootShard, operatorv1alpha1.RootCA),
					Kind:  "Issuer",
					Group: "cert-manager.io",
				},
			}

			return cert, nil
		}
	}
}
