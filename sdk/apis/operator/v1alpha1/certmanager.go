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

package v1alpha1

// The types in this file are copied directly from cert-manager/v1 to avoid a
// costly dependency on the cert-manager Go module from the operator's SDK.

// Denotes how private keys should be generated or sourced when a Certificate
// is being issued.
// +kubebuilder:validation:Enum=Never;Always
type PrivateKeyRotationPolicy string

var (
	// RotationPolicyNever means a private key will only be generated if one
	// does not already exist in the target `spec.secretName`.
	// If one does exist but it does not have the correct algorithm or size,
	// a warning will be raised to await user intervention.
	RotationPolicyNever PrivateKeyRotationPolicy = "Never"

	// RotationPolicyAlways means a private key matching the specified
	// requirements will be generated whenever a re-issuance occurs.
	RotationPolicyAlways PrivateKeyRotationPolicy = "Always"
)

// +kubebuilder:validation:Enum=PKCS1;PKCS8
type PrivateKeyEncoding string

const (
	// PKCS1 private key encoding.
	// PKCS1 produces a PEM block that contains the private key algorithm
	// in the header and the private key in the body. A key that uses this
	// can be recognised by its `BEGIN RSA PRIVATE KEY` or `BEGIN EC PRIVATE KEY` header.
	// NOTE: This encoding is not supported for Ed25519 keys. Attempting to use
	// this encoding with an Ed25519 key will be ignored and default to PKCS8.
	PKCS1 PrivateKeyEncoding = "PKCS1"

	// PKCS8 private key encoding.
	// PKCS8 produces a PEM block with a static header and both the private
	// key algorithm and the private key in the body. A key that uses this
	// encoding can be recognised by its `BEGIN PRIVATE KEY` header.
	PKCS8 PrivateKeyEncoding = "PKCS8"
)

// +kubebuilder:validation:Enum=RSA;ECDSA;Ed25519
type PrivateKeyAlgorithm string

const (
	// RSA private key algorithm.
	RSAKeyAlgorithm PrivateKeyAlgorithm = "RSA"

	// ECDSA private key algorithm.
	ECDSAKeyAlgorithm PrivateKeyAlgorithm = "ECDSA"

	// Ed25519 private key algorithm.
	Ed25519KeyAlgorithm PrivateKeyAlgorithm = "Ed25519"
)

// X509Subject Full X509 name specification
type X509Subject struct {
	// Organizations to be used on the Certificate.
	// +optional
	Organizations []string `json:"organizations,omitempty"`
	// Countries to be used on the Certificate.
	// +optional
	Countries []string `json:"countries,omitempty"`
	// Organizational Units to be used on the Certificate.
	// +optional
	OrganizationalUnits []string `json:"organizationalUnits,omitempty"`
	// Cities to be used on the Certificate.
	// +optional
	Localities []string `json:"localities,omitempty"`
	// State/Provinces to be used on the Certificate.
	// +optional
	Provinces []string `json:"provinces,omitempty"`
	// Street addresses to be used on the Certificate.
	// +optional
	StreetAddresses []string `json:"streetAddresses,omitempty"`
	// Postal codes to be used on the Certificate.
	// +optional
	PostalCodes []string `json:"postalCodes,omitempty"`
	// Serial number to be used on the Certificate.
	// +optional
	SerialNumber string `json:"serialNumber,omitempty"`
}
