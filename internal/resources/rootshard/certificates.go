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

package rootshard

import (
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kcp-dev/kcp-operator/api/v1alpha1"
	"github.com/kcp-dev/kcp-operator/internal/reconciling"
)

func ServerCertificateReconciler(rootShard *v1alpha1.RootShard) reconciling.NamedCertificateReconcilerFactory {
	name := rootShard.GetCertificateName(v1alpha1.ServerCertificate)

	return func() (string, reconciling.CertificateReconciler) {
		return name, func(cert *certmanagerv1.Certificate) (*certmanagerv1.Certificate, error) {
			cert.SetLabels(rootShard.GetResourceLabels())
			cert.Spec = certmanagerv1.CertificateSpec{
				SecretName:  name,
				Duration:    &metav1.Duration{Duration: time.Hour * 24 * 365},
				RenewBefore: &metav1.Duration{Duration: time.Hour * 24 * 7},

				PrivateKey: &certmanagerv1.CertificatePrivateKey{
					Algorithm: certmanagerv1.RSAKeyAlgorithm,
					Size:      4096,
				},

				Usages: []certmanagerv1.KeyUsage{
					certmanagerv1.UsageServerAuth,
				},

				DNSNames: []string{
					rootShard.Spec.Hostname,
				},

				IssuerRef: certmanagermetav1.ObjectReference{
					Name:  rootShard.GetCAName(v1alpha1.RootCA),
					Kind:  "Issuer",
					Group: "cert-manager.io",
				},
			}

			return cert, nil
		}
	}
}

func ServiceAccountCertificateReconciler(rootShard *v1alpha1.RootShard) reconciling.NamedCertificateReconcilerFactory {
	name := rootShard.GetCertificateName(v1alpha1.ServiceAccountCertificate)

	return func() (string, reconciling.CertificateReconciler) {
		return name, func(cert *certmanagerv1.Certificate) (*certmanagerv1.Certificate, error) {
			cert.SetLabels(rootShard.GetResourceLabels())
			cert.Spec = certmanagerv1.CertificateSpec{
				CommonName:  name,
				SecretName:  name,
				Duration:    &metav1.Duration{Duration: time.Hour * 24 * 365},
				RenewBefore: &metav1.Duration{Duration: time.Hour * 24 * 7},

				PrivateKey: &certmanagerv1.CertificatePrivateKey{
					Algorithm: certmanagerv1.RSAKeyAlgorithm,
					Size:      4096,
				},

				IssuerRef: certmanagermetav1.ObjectReference{
					Name:  rootShard.GetCAName(v1alpha1.ServiceAccountCA),
					Kind:  "Issuer",
					Group: "cert-manager.io",
				},
			}

			return cert, nil
		}
	}
}
