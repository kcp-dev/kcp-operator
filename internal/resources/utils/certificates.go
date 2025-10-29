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
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"maps"
	"os"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
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

func mergeSlices(a, b []string) []string {
	return sets.List(sets.New(a...).Insert(b...))
}

func ApplyCertificateTemplate(cert *certmanagerv1.Certificate, tpl *operatorv1alpha1.CertificateTemplate) *certmanagerv1.Certificate {
	if tpl == nil {
		return cert
	}

	if metadata := tpl.Metadata; metadata != nil {
		cert.Annotations = addNewKeys(cert.Annotations, metadata.Annotations)
		cert.Labels = addNewKeys(cert.Labels, metadata.Labels)
	}

	applyCertificateSpecTemplate(cert, tpl.Spec)

	return cert
}

func applyCertificateSpecTemplate(cert *certmanagerv1.Certificate, tpl *operatorv1alpha1.CertificateSpecTemplate) *certmanagerv1.Certificate {
	if tpl == nil {
		return cert
	}

	// If DNSNames is provided in the template and issuer is overridden,
	// it will replace any existing DNSNames.
	// We don't merge as we don't know if issuer supports our default names.
	// It's user responsibility to add them back if needed.
	if len(tpl.DNSNames) > 0 && tpl.IssuerRef != nil {
		cert.Spec.DNSNames = tpl.DNSNames
	} else if len(tpl.DNSNames) > 0 {
		cert.Spec.DNSNames = mergeSlices(cert.Spec.DNSNames, tpl.DNSNames)
	}

	cert.Spec.IPAddresses = mergeSlices(cert.Spec.IPAddresses, tpl.IPAddresses)

	if tpl.Duration != nil {
		cert.Spec.Duration = tpl.Duration.DeepCopy()
	}

	if tpl.RenewBefore != nil {
		cert.Spec.RenewBefore = tpl.RenewBefore.DeepCopy()
	}

	if secretTpl := tpl.SecretTemplate; secretTpl != nil {
		if cert.Spec.SecretTemplate == nil {
			cert.Spec.SecretTemplate = &certmanagerv1.CertificateSecretTemplate{}
		}

		cert.Spec.SecretTemplate.Annotations = addNewKeys(cert.Spec.SecretTemplate.Annotations, secretTpl.Annotations)
		cert.Spec.SecretTemplate.Labels = addNewKeys(cert.Spec.SecretTemplate.Labels, secretTpl.Labels)
	}
	if tpl.IssuerRef != nil {
		cert.Spec.IssuerRef = cmmeta.ObjectReference{
			Name:  tpl.IssuerRef.Name,
			Kind:  tpl.IssuerRef.Kind,
			Group: tpl.IssuerRef.Group,
		}
	}

	cert.Spec.PrivateKey = applyCertificatePrivateKeyTemplate(cert.Spec.PrivateKey, tpl.PrivateKey)
	cert.Spec.Subject = applyCertificateSubjectTemplate(cert.Spec.Subject, tpl.Subject)

	return cert
}

func applyCertificatePrivateKeyTemplate(pk *certmanagerv1.CertificatePrivateKey, tpl *operatorv1alpha1.CertificatePrivateKeyTemplate) *certmanagerv1.CertificatePrivateKey {
	if tpl == nil {
		return pk
	}

	// This should never happen.
	if pk == nil {
		pk = &certmanagerv1.CertificatePrivateKey{}
	}

	if tpl.Algorithm != "" {
		pk.Algorithm = certmanagerv1.PrivateKeyAlgorithm(tpl.Algorithm)
	}

	if tpl.Encoding != "" {
		pk.Encoding = certmanagerv1.PrivateKeyEncoding(tpl.Encoding)
	}

	if tpl.RotationPolicy != "" {
		pk.RotationPolicy = certmanagerv1.PrivateKeyRotationPolicy(tpl.RotationPolicy)
	}

	if tpl.Size > 0 {
		pk.Size = tpl.Size
	}

	return pk
}

func applyCertificateSubjectTemplate(subj *certmanagerv1.X509Subject, tpl *operatorv1alpha1.X509Subject) *certmanagerv1.X509Subject {
	if tpl == nil {
		return subj
	}

	// This should never happen.
	if subj == nil {
		subj = &certmanagerv1.X509Subject{}
	}

	subj.Organizations = mergeSlices(subj.Organizations, tpl.Organizations)
	subj.Countries = mergeSlices(subj.Countries, tpl.Countries)
	subj.OrganizationalUnits = mergeSlices(subj.OrganizationalUnits, tpl.OrganizationalUnits)
	subj.Localities = mergeSlices(subj.Localities, tpl.Localities)
	subj.Provinces = mergeSlices(subj.Provinces, tpl.Provinces)
	subj.StreetAddresses = mergeSlices(subj.StreetAddresses, tpl.StreetAddresses)
	subj.PostalCodes = mergeSlices(subj.PostalCodes, tpl.PostalCodes)

	if tpl.SerialNumber != "" {
		subj.SerialNumber = tpl.SerialNumber
	}

	return subj
}

// ValidatePEMCertificate validates that the given data contains valid PEM-encoded certificates.
func ValidatePEMCertificate(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	block, rest := pem.Decode(data)
	for block != nil {
		if block.Type != "CERTIFICATE" {
			return fmt.Errorf("invalid PEM block type: %s, expected CERTIFICATE", block.Type)
		}

		if _, err := x509.ParseCertificate(block.Bytes); err != nil {
			return fmt.Errorf("failed to parse certificate: %w", err)
		}

		block, rest = pem.Decode(rest)
	}

	if len(rest) > 0 {
		return fmt.Errorf("invalid PEM data: remaining non-PEM data found")
	}

	return nil
}

// MergeCABundles merges the CA certificate data from two secrets.
func MergeCABundles(caSecret, caBundle *corev1.Secret) ([]byte, error) {
	var merged []byte

	if caSecret != nil && caSecret.Data != nil {
		if caCrt, exists := caSecret.Data["tls.crt"]; exists {
			if err := ValidatePEMCertificate(caCrt); err != nil {
				return nil, fmt.Errorf("invalid certificate in caSecret: %w", err)
			}
			merged = append(merged, caCrt...)
		}
	}

	if caBundle != nil && caBundle.Data != nil {
		if caCrt, exists := caBundle.Data["tls.crt"]; exists {
			if err := ValidatePEMCertificate(caCrt); err != nil {
				return nil, fmt.Errorf("invalid certificate in caBundle: %w", err)
			}
			merged = append(merged, caCrt...)
		}
	}

	return merged, nil
}

// MergeCABundlesFiles merges the CA certificate data from two files.
func MergeCABundlesFiles(caFile1, caFile2 string) ([]byte, error) {
	var merged []byte

	// Read and validate the first CA file
	caFile1Content, err := os.ReadFile(caFile1)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA file 1: %w", err)
	}
	if err := ValidatePEMCertificate(caFile1Content); err != nil {
		return nil, fmt.Errorf("invalid certificate in caFile1: %w", err)
	}
	merged = append(merged, caFile1Content...)

	// Read and validate the second CA file
	caFile2Content, err := os.ReadFile(caFile2)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA file 2: %w", err)
	}
	if err := ValidatePEMCertificate(caFile2Content); err != nil {
		return nil, fmt.Errorf("invalid certificate in caFile2: %w", err)
	}
	merged = append(merged, caFile2Content...)

	return merged, nil
}
