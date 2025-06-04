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

package utils

import (
	"maps"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"

	"k8s.io/apimachinery/pkg/util/sets"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func addNewKeys(existing, toAdd map[string]string) map[string]string {
	if len(toAdd) == 0 {
		return existing
	}

	// do not modify the given map in-place, but clone it; start from the source
	// so we do not need to check for key existence individually
	result := maps.Clone(toAdd)

	// copy and overwrite all keys from the destination
	maps.Copy(result, existing)

	return result
}

func ApplyCertificateTemplate(cert *certmanagerv1.Certificate, tpl *operatorv1alpha1.CertificateTemplate) *certmanagerv1.Certificate {
	if tpl == nil {
		return cert
	}

	if metadata := tpl.Metadata; metadata != nil {
		cert.Annotations = addNewKeys(cert.Annotations, metadata.Annotations)
		cert.Labels = addNewKeys(cert.Labels, metadata.Labels)
	}

	if spec := tpl.Spec; spec != nil {
		cert.Spec.DNSNames = sets.List(sets.New(cert.Spec.DNSNames...).Insert(spec.DNSNames...))
		cert.Spec.IPAddresses = sets.List(sets.New(cert.Spec.IPAddresses...).Insert(spec.IPAddresses...))

		if spec.Duration != nil {
			cert.Spec.Duration = spec.Duration.DeepCopy()
		}

		if spec.RenewBefore != nil {
			cert.Spec.RenewBefore = spec.RenewBefore.DeepCopy()
		}

		if secretTpl := spec.SecretTemplate; secretTpl != nil {
			if cert.Spec.SecretTemplate == nil {
				cert.Spec.SecretTemplate = &certmanagerv1.CertificateSecretTemplate{}
			}

			cert.Spec.SecretTemplate.Annotations = addNewKeys(cert.Spec.SecretTemplate.Annotations, secretTpl.Annotations)
			cert.Spec.SecretTemplate.Labels = addNewKeys(cert.Spec.SecretTemplate.Labels, secretTpl.Labels)
		}

		if pk := spec.PrivateKey; pk != nil {
			// This should never happen.
			if cert.Spec.PrivateKey == nil {
				cert.Spec.PrivateKey = &certmanagerv1.CertificatePrivateKey{}
			}

			if pk.Algorithm != "" {
				cert.Spec.PrivateKey.Algorithm = certmanagerv1.PrivateKeyAlgorithm(pk.Algorithm)
			}

			if pk.Encoding != "" {
				cert.Spec.PrivateKey.Encoding = certmanagerv1.PrivateKeyEncoding(pk.Encoding)
			}

			if pk.RotationPolicy != "" {
				cert.Spec.PrivateKey.RotationPolicy = certmanagerv1.PrivateKeyRotationPolicy(pk.RotationPolicy)
			}

			if pk.Size > 0 {
				cert.Spec.PrivateKey.Size = pk.Size
			}
		}
	}

	return cert
}
