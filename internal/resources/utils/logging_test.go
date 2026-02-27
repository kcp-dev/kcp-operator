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

package utils

import (
	"testing"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func TestGetLogLevelArgs(t *testing.T) {
	tests := []struct {
		name     string
		logging  *operatorv1alpha1.LoggingSpec
		expected []string
	}{
		{
			name:     "no config at all",
			logging:  nil,
			expected: []string{},
		},
		{
			name: "verbosityLevel 0",
			logging: &operatorv1alpha1.LoggingSpec{
				Level: 0,
			},
			expected: []string{},
		},
		{
			name: "verbosityLevel 2",
			logging: &operatorv1alpha1.LoggingSpec{
				Level: 2,
			},
			expected: []string{"-v=2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetLoggingArgs(tt.logging)
			if len(result) != len(tt.expected) {
				t.Fatalf("Expected %d arguments, got %d.", len(tt.expected), len(result))
			}

			for i, arg := range result {
				if arg != tt.expected[i] {
					t.Errorf("Expected args.#%d to be %q, got %q.", i, tt.expected[i], arg)
				}
			}
		})
	}
}
