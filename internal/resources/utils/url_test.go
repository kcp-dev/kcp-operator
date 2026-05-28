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

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractHostnameFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "empty URL",
			url:      "",
			expected: "",
		},
		{
			name:     "valid URL with https",
			url:      "https://api.example.com",
			expected: "api.example.com",
		},
		{
			name:     "valid URL with port",
			url:      "https://api.example.com:6443",
			expected: "api.example.com",
		},
		{
			name:     "valid URL with http",
			url:      "http://localhost:8080",
			expected: "localhost",
		},
		{
			name:     "URL with path",
			url:      "https://api.example.com:6443/path/to/resource",
			expected: "api.example.com",
		},
		{
			name:     "subdomain URL",
			url:      "https://root.shard.kcp.example.com:6443",
			expected: "root.shard.kcp.example.com",
		},
		{
			name:     "invalid URL",
			url:      "not-a-valid-url",
			expected: "",
		},
		{
			name:     "URL with IPv4 address",
			url:      "https://192.168.1.1:6443",
			expected: "192.168.1.1",
		},
		{
			name:     "URL with IPv6 address",
			url:      "https://[2001:db8::1]:6443",
			expected: "2001:db8::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractHostnameFromURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}
