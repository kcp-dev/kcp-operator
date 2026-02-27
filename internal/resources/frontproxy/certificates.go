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

package frontproxy

import (
	"fmt"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"

	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/utils"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func (r *reconciler) certSecretLabels() map[string]string {
	labels := map[string]string{
		resources.RootShardLabel: r.rootShard.Name,
	}

	if r.frontProxy != nil {
		labels[resources.FrontProxyLabel] = r.frontProxy.Name
	}

	return labels
}

func (r *reconciler) certName(certKind operatorv1alpha1.Certificate) string {
	if r.frontProxy != nil {
		return resources.GetFrontProxyCertificateName(r.rootShard, r.frontProxy, certKind)
	} else {
		return resources.GetRootShardProxyCertificateName(r.rootShard, certKind)
	}
}

func (r *reconciler) certCommonName() string {
	if r.frontProxy != nil {
		return "kcp-front-proxy"
	} else {
		return "kcp-root-shard-proxy"
	}
}

func (r *reconciler) certTemplateMap() operatorv1alpha1.CertificateTemplateMap {
	switch {
	case r.frontProxy != nil:
		return r.frontProxy.Spec.CertificateTemplates
	case r.rootShard.Spec.Proxy != nil:
		return r.rootShard.Spec.Proxy.CertificateTemplates
	default:
		return nil
	}
}

func (r *reconciler) serverCertificateReconciler() reconciling.NamedCertificateReconcilerFactory {
	const certKind = operatorv1alpha1.ServerCertificate

	name := r.certName(certKind)
	template := r.certTemplateMap().CertificateTemplate(certKind)
	fpService := r.serviceName()

	dnsNames := []string{
		fpService,
		fmt.Sprintf("%s.%s", fpService, r.rootShard.Namespace),
		fmt.Sprintf("%s.%s.svc.cluster.local", fpService, r.rootShard.Namespace),
	}

	if r.frontProxy != nil {
		// only add the external hostname if this is not reconciling the rootshard-internal-only proxy
		dnsNames = append(dnsNames, r.rootShard.Spec.External.Hostname)
		if r.rootShard.Spec.External.PrivateHostname != "" {
			dnsNames = append(dnsNames, r.rootShard.Spec.External.PrivateHostname)
		}

		// DEPRECATED: keep support for the deprecated ExternalHostname field for now
		// to not break existing front-proxy installations.
		if r.frontProxy.Spec.ExternalHostname != "" {
			dnsNames = append(dnsNames, r.frontProxy.Spec.ExternalHostname)
		} else if r.frontProxy.Spec.External.Hostname != "" {
			dnsNames = append(dnsNames, r.frontProxy.Spec.External.Hostname)
		}
	}

	return func() (string, reconciling.CertificateReconciler) {
		return name, func(cert *certmanagerv1.Certificate) (*certmanagerv1.Certificate, error) {
			cert.SetLabels(r.resourceLabels)
			cert.Spec = certmanagerv1.CertificateSpec{
				SecretName: name,
				SecretTemplate: &certmanagerv1.CertificateSecretTemplate{
					Labels: r.certSecretLabels(),
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

				DNSNames: dnsNames,

				IssuerRef: certmanagermetav1.ObjectReference{
					Name:  resources.GetRootShardCAName(r.rootShard, operatorv1alpha1.ServerCA),
					Kind:  "Issuer",
					Group: "cert-manager.io",
				},
			}

			return utils.ApplyCertificateTemplate(cert, &template), nil
		}
	}
}

// adminKubeconfigCertificateReconciler is only reconciled for true front-proxies.
func (r *reconciler) adminKubeconfigCertificateReconciler() reconciling.NamedCertificateReconcilerFactory {
	const certKind = operatorv1alpha1.AdminKubeconfigClientCertificate

	name := r.certName(certKind)
	template := r.certTemplateMap().CertificateTemplate(certKind)

	return func() (string, reconciling.CertificateReconciler) {
		return name, func(cert *certmanagerv1.Certificate) (*certmanagerv1.Certificate, error) {
			cert.SetLabels(r.resourceLabels)
			cert.Spec = certmanagerv1.CertificateSpec{
				SecretName: name,
				SecretTemplate: &certmanagerv1.CertificateSecretTemplate{
					Labels: r.certSecretLabels(),
				},
				Duration:    &operatorv1alpha1.DefaultCertificateDuration,
				RenewBefore: &operatorv1alpha1.DefaultCertificateRenewal,

				PrivateKey: &certmanagerv1.CertificatePrivateKey{
					Algorithm: certmanagerv1.RSAKeyAlgorithm,
					Size:      4096,
				},

				CommonName: "external-logical-cluster-admin",

				Usages: []certmanagerv1.KeyUsage{
					certmanagerv1.UsageClientAuth,
				},

				Subject: &certmanagerv1.X509Subject{
					Organizations: []string{"system:kcp:external-logical-cluster-admin"},
				},

				IssuerRef: certmanagermetav1.ObjectReference{
					Name:  resources.GetRootShardCAName(r.rootShard, operatorv1alpha1.FrontProxyClientCA),
					Kind:  "Issuer",
					Group: "cert-manager.io",
				},
			}

			return utils.ApplyCertificateTemplate(cert, &template), nil
		}
	}
}

func (r *reconciler) kubeconfigCertificateReconciler() reconciling.NamedCertificateReconcilerFactory {
	const certKind = operatorv1alpha1.KubeconfigCertificate

	name := r.certName(certKind)
	template := r.certTemplateMap().CertificateTemplate(certKind)

	return func() (string, reconciling.CertificateReconciler) {
		return name, func(cert *certmanagerv1.Certificate) (*certmanagerv1.Certificate, error) {
			cert.SetLabels(r.resourceLabels)
			cert.Spec = certmanagerv1.CertificateSpec{
				SecretName: name,
				SecretTemplate: &certmanagerv1.CertificateSecretTemplate{
					Labels: r.certSecretLabels(),
				},
				Duration:    &operatorv1alpha1.DefaultCertificateDuration,
				RenewBefore: &operatorv1alpha1.DefaultCertificateRenewal,

				PrivateKey: &certmanagerv1.CertificatePrivateKey{
					Algorithm: certmanagerv1.RSAKeyAlgorithm,
					Size:      4096,
				},

				CommonName: r.certCommonName(),

				Subject: &certmanagerv1.X509Subject{
					Organizations: []string{"system:masters"},
				},

				Usages: []certmanagerv1.KeyUsage{
					certmanagerv1.UsageClientAuth,
				},

				IssuerRef: certmanagermetav1.ObjectReference{
					Name:  resources.GetRootShardCAName(r.rootShard, operatorv1alpha1.ClientCA),
					Kind:  "Issuer",
					Group: "cert-manager.io",
				},
			}

			return utils.ApplyCertificateTemplate(cert, &template), nil
		}
	}
}

func (r *reconciler) requestHeaderCertificateReconciler() reconciling.NamedCertificateReconcilerFactory {
	const certKind = operatorv1alpha1.RequestHeaderClientCertificate

	name := r.certName(certKind)
	template := r.certTemplateMap().CertificateTemplate(certKind)

	return func() (string, reconciling.CertificateReconciler) {
		return name, func(cert *certmanagerv1.Certificate) (*certmanagerv1.Certificate, error) {
			cert.SetLabels(r.resourceLabels)
			cert.Spec = certmanagerv1.CertificateSpec{
				SecretName: name,
				SecretTemplate: &certmanagerv1.CertificateSecretTemplate{
					Labels: r.certSecretLabels(),
				},
				Duration:    &operatorv1alpha1.DefaultCertificateDuration,
				RenewBefore: &operatorv1alpha1.DefaultCertificateRenewal,

				PrivateKey: &certmanagerv1.CertificatePrivateKey{
					Algorithm: certmanagerv1.RSAKeyAlgorithm,
					Size:      4096,
				},

				CommonName: r.certCommonName(),

				Usages: []certmanagerv1.KeyUsage{
					certmanagerv1.UsageClientAuth,
				},

				IssuerRef: certmanagermetav1.ObjectReference{
					Name:  resources.GetRootShardCAName(r.rootShard, operatorv1alpha1.RequestHeaderClientCA),
					Kind:  "Issuer",
					Group: "cert-manager.io",
				},
			}

			return utils.ApplyCertificateTemplate(cert, &template), nil
		}
	}
}
