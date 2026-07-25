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

package config

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/sets"
)

func TestParseControllerGroups(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected sets.Set[ControllerGroup]
		wantErr  bool
	}{
		{
			name:     "all groups",
			input:    []string{"config", "workload"},
			expected: sets.New(ControllerGroupConfig, ControllerGroupWorkload),
		},
		{
			name:     "config only",
			input:    []string{"config"},
			expected: sets.New(ControllerGroupConfig),
		},
		{
			name:     "workload only",
			input:    []string{"workload"},
			expected: sets.New(ControllerGroupWorkload),
		},
		{
			name:     "duplicates are collapsed",
			input:    []string{"config", "config"},
			expected: sets.New(ControllerGroupConfig),
		},
		{
			name:     "whitespace is trimmed",
			input:    []string{" config ", "workload"},
			expected: sets.New(ControllerGroupConfig, ControllerGroupWorkload),
		},
		{
			name:    "unknown group",
			input:   []string{"config", "bogus"},
			wantErr: true,
		},
		{
			name:    "empty list",
			input:   []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseControllerGroups(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", parsed)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !parsed.Equal(tt.expected) {
				t.Errorf("expected %v, got %v", sets.List(tt.expected), sets.List(parsed))
			}
		})
	}
}
