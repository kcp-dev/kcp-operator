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

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func TestGetLogLevelArgs(t *testing.T) {
	tests := []struct {
		name     string
		logLevel *operatorv1alpha1.LogLevelSpec
		expected []string
	}{
		{
			name: "nil verbosityLevel",
			logLevel: &operatorv1alpha1.LogLevelSpec{
				VerbosityLevel: nil,
			},
			expected: []string{},
		},
		{
			name: "verbosityLevel 0",
			logLevel: &operatorv1alpha1.LogLevelSpec{
				VerbosityLevel: func() *int32 { v := int32(0); return &v }(),
			},
			expected: []string{},
		},
		{
			name: "verbosityLevel 1",
			logLevel: &operatorv1alpha1.LogLevelSpec{
				VerbosityLevel: func() *int32 { v := int32(1); return &v }(),
			},
			expected: []string{"-v=1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetLogLevelArgs(tt.logLevel)
			if len(result) != len(tt.expected) {
				t.Errorf("GetLogLevelArgs() returned %d arguments, expected %d", len(result), len(tt.expected))
				return
			}
			for i, arg := range result {
				if arg != tt.expected[i] {
					t.Errorf("GetLogLevelArgs() argument %d = %v, expected %v", i, arg, tt.expected[i])
				}
			}
		})
	}
}
