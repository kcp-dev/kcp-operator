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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePEMCertificate(t *testing.T) {
	// Valid certificate for testing valid case
	validCert := `-----BEGIN CERTIFICATE-----
MIICMzCCAZygAwIBAgIJALiPnVsvq8dsMA0GCSqGSIb3DQEBBQUAMFMxCzAJBgNV
BAYTAlVTMQwwCgYDVQQIEwNmb28xDDAKBgNVBAcTA2ZvbzEMMAoGA1UEChMDZm9v
MQwwCgYDVQQLEwNmb28xDDAKBgNVBAMTA2ZvbzAeFw0xMzAzMTkxNTQwMTlaFw0x
ODAzMTgxNTQwMTlaMFMxCzAJBgNVBAYTAlVTMQwwCgYDVQQIEwNmb28xDDAKBgNV
BAcTA2ZvbzEMMAoGA1UEChMDZm9vMQwwCgYDVQQLEwNmb28xDDAKBgNVBAMTA2Zv
bzCBnzANBgkqhkiG9w0BAQEFAAOBjQAwgYkCgYEAzdGfxi9CNbMf1UUcvDQh7MYB
OveIHyc0E0KIbhjK5FkCBU4CiZrbfHagaW7ZEcN0tt3EvpbOMxxc/ZQU2WN/s/wP
xph0pSfsfFsTKM4RhTWD2v4fgk+xZiKd1p0+L4hTtpwnEw0uXRVd0ki6muwV5y/P
+5FHUeldq+pgTcgzuK8CAwEAAaMPMA0wCwYDVR0PBAQDAgLkMA0GCSqGSIb3DQEB
BQUAA4GBAJiDAAtY0mQQeuxWdzLRzXmjvdSuL9GoyT3BF/jSnpxz5/58dba8pWen
v3pj4P3w5DoOso0rzkZy2jEsEitlVM2mLSbQpMM+MUVQCQoiG6W9xuCFuxSrwPIS
pAqEAuV4DNoxQKKWmhVv+J0ptMWD25Pnpxeq5sXzghfJnslJlQND
-----END CERTIFICATE-----`

	// Valid RSA private key for testing invalid block type
	validPrivateKey := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAzdGfxi9CNbMf1UUcvDQh7MYBOveIHyc0E0KIbhjK5FkCBU4C
iZrbfHagaW7ZEcN0tt3EvpbOMxxc/ZQU2WN/s/wPxph0pSfsfFsTKM4RhTWD2v4f
gk+xZiKd1p0+L4hTtpwnEw0uXRVd0ki6muwV5y/P+5FHUeldq+pgTcgzuK8CAwEA
AQKCAQEAkJSaCFJXJlBqUiYF4QUVrVGfn7XCCbBQbEhJ3R1fMfWKzVh0Q3F1fJpL
SbB2J0tYOQ5VbFQc5D1oFqWuQVMZ3rFKgQo3F5J3F3K1F4J3F2K2F3J0F2K3F5J2
F3K0F2J4F5K3F2J1F4K2F3J3F5K0F2J2F4K1F3J4F5K2F3J0F4K3F2J1F5K4F3J2
F0K2F4J5F3K1F2J3F4K0F5J4F3K2F1J0F5K3F4J1F2K4F3J5F0K1F4J2F5K3F1J4
F0K2F5J3F4K1F2J0F5K4F3J1F2K5F4J0F1K3F5J2F4K0F1J5F3K4F2J1F0K5F4J3
F1K2F5J0F4K3F1J2F5K4F0J1F3K5F2J4F1K0F5J3F4K2F1J5F0K4F3J2F1KQKBgQD
4K5F2J1F0K5F3J4F2K1F5J0F4K3F1J2F5K4F0J1F3K5F2J4F1K0F5J3F4K2F1J5F0K4
F3J2F1K0F5J4F3K2F1J0F5K4F3J1F2K5F4J0F1K3F5J2F4K0F1J5F3K4F2J1F0K5
F4J3F1K2F5J0F4K3F1J2F5K4F0J1F3K5F2J4F1K0F5J3F4K2F1J5F0K4F3J2F1KQ
-----END RSA PRIVATE KEY-----`

	// Invalid certificate data (proper PEM format with valid base64 but invalid certificate)
	invalidCertData := `-----BEGIN CERTIFICATE-----
YWJjZGVmZ2hpams=
-----END CERTIFICATE-----`

	tests := []struct {
		name    string
		data    []byte
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty data",
			data:    []byte{},
			wantErr: false,
		},
		{
			name:    "nil data",
			data:    nil,
			wantErr: false,
		},
		{
			name:    "valid certificate",
			data:    []byte(validCert),
			wantErr: false,
		},
		{
			name:    "invalid PEM format",
			data:    []byte("not-a-pem-certificate"),
			wantErr: true,
			errMsg:  "invalid PEM data: remaining non-PEM data found",
		},
		{
			name:    "invalid certificate data",
			data:    []byte(invalidCertData),
			wantErr: true,
			errMsg:  "failed to parse certificate",
		},
		{
			name:    "PEM block with wrong type only",
			data:    []byte(validPrivateKey),
			wantErr: true,
			errMsg:  "invalid PEM block type: RSA PRIVATE KEY, expected CERTIFICATE",
		},
		{
			name:    "trailing non-PEM data after empty certificate block",
			data:    []byte("some-non-pem-data"),
			wantErr: true,
			errMsg:  "invalid PEM data: remaining non-PEM data found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePEMCertificate(tt.data)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
