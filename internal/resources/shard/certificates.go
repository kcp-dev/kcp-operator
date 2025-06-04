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

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"

	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/utils"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func ServerCertificateReconciler(shard *operatorv1alpha1.Shard, rootShard *operatorv1alpha1.RootShard) reconciling.NamedCertificateReconcilerFactory {
	const certKind = operatorv1alpha1.ServerCertificate

	name := resources.GetShardCertificateName(shard, certKind)
	template := shard.Spec.CertificateTemplates.CertificateTemplate(certKind)

	return func() (string, reconciling.CertificateReconciler) {
		return name, func(cert *certmanagerv1.Certificate) (*certmanagerv1.Certificate, error) {
			cert.SetLabels(resources.GetShardResourceLabels(shard))
			cert.Spec = certmanagerv1.CertificateSpec{
				SecretName:  name,
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
					"localhost",
					resources.GetShardBaseHost(shard),
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

func VirtualWorkspacesCertificateReconciler(shard *operatorv1alpha1.Shard, rootShard *operatorv1alpha1.RootShard) reconciling.NamedCertificateReconcilerFactory {
	const certKind = operatorv1alpha1.VirtualWorkspacesCertificate

	name := resources.GetShardCertificateName(shard, certKind)
	template := shard.Spec.CertificateTemplates.CertificateTemplate(certKind)

	return func() (string, reconciling.CertificateReconciler) {
		return name, func(cert *certmanagerv1.Certificate) (*certmanagerv1.Certificate, error) {
			cert.SetLabels(resources.GetShardResourceLabels(shard))
			cert.Spec = certmanagerv1.CertificateSpec{
				SecretName:  name,
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
					resources.GetShardBaseHost(shard),
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

func ServiceAccountCertificateReconciler(shard *operatorv1alpha1.Shard, rootShard *operatorv1alpha1.RootShard) reconciling.NamedCertificateReconcilerFactory {
	const certKind = operatorv1alpha1.ServiceAccountCertificate

	name := resources.GetShardCertificateName(shard, certKind)
	template := shard.Spec.CertificateTemplates.CertificateTemplate(certKind)

	return func() (string, reconciling.CertificateReconciler) {
		return name, func(cert *certmanagerv1.Certificate) (*certmanagerv1.Certificate, error) {
			cert.SetLabels(resources.GetShardResourceLabels(shard))
			cert.Spec = certmanagerv1.CertificateSpec{
				CommonName:  name,
				SecretName:  name,
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

func RootShardClientCertificateReconciler(shard *operatorv1alpha1.Shard, rootShard *operatorv1alpha1.RootShard) reconciling.NamedCertificateReconcilerFactory {
	const certKind = operatorv1alpha1.ClientCertificate

	name := resources.GetShardCertificateName(shard, certKind)
	template := shard.Spec.CertificateTemplates.CertificateTemplate(certKind)

	return func() (string, reconciling.CertificateReconciler) {
		return name, func(cert *certmanagerv1.Certificate) (*certmanagerv1.Certificate, error) {
			cert.SetLabels(resources.GetShardResourceLabels(shard))
			cert.Spec = certmanagerv1.CertificateSpec{
				CommonName:  fmt.Sprintf("shard-%s", shard.Name),
				SecretName:  name,
				Duration:    &operatorv1alpha1.DefaultCertificateDuration,
				RenewBefore: &operatorv1alpha1.DefaultCertificateRenewal,

				PrivateKey: &certmanagerv1.CertificatePrivateKey{
					Algorithm: certmanagerv1.RSAKeyAlgorithm,
					Size:      4096,
				},

				Subject: &certmanagerv1.X509Subject{
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
