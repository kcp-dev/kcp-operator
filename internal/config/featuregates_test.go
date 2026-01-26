/*
Copyright 2026 The KCP Authors.

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

	"k8s.io/component-base/featuregate"
)

func TestFeatureGates(t *testing.T) {
	tests := []struct {
		name            string
		feature         featuregate.Feature
		expectedDefault bool
	}{
		{
			name:            "ConfigurationBundle is disabled by default",
			feature:         ConfigurationBundle,
			expectedDefault: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset feature gate to defaults for each test
			fg := featuregate.NewFeatureGate()
			if err := fg.Add(defaultKCPOperatorFeatureGates); err != nil {
				t.Fatalf("failed to add feature gates: %v", err)
			}

			enabled := fg.Enabled(tt.feature)
			if enabled != tt.expectedDefault {
				t.Errorf("expected feature %q to be %v by default, got %v", tt.feature, tt.expectedDefault, enabled)
			}
		})
	}
}

func TestSetFeatureGateDuringTest(t *testing.T) {
	// Save original state
	originalMutableGate := DefaultMutableFeatureGate
	originalGate := DefaultFeatureGate
	defer func() {
		DefaultMutableFeatureGate = originalMutableGate
		DefaultFeatureGate = originalGate
	}()

	// Create a new feature gate for testing
	DefaultMutableFeatureGate = featuregate.NewFeatureGate()
	DefaultFeatureGate = DefaultMutableFeatureGate
	if err := DefaultMutableFeatureGate.Add(defaultKCPOperatorFeatureGates); err != nil {
		t.Fatalf("failed to add feature gates: %v", err)
	}

	tests := []struct {
		name        string
		feature     featuregate.Feature
		enableValue bool
	}{
		{
			name:        "enable ConfigurationBundle",
			feature:     ConfigurationBundle,
			enableValue: true,
		},
		{
			name:        "disable ConfigurationBundle",
			feature:     ConfigurationBundle,
			enableValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SetFeatureGateDuringTest(tt.feature, tt.enableValue); err != nil {
				t.Fatalf("failed to set feature gate: %v", err)
			}

			if enabled := DefaultFeatureGate.Enabled(tt.feature); enabled != tt.enableValue {
				t.Errorf("expected feature %q to be %v, got %v", tt.feature, tt.enableValue, enabled)
			}
		})
	}
}

func TestEnabledFunction(t *testing.T) {
	// Save original state
	originalMutableGate := DefaultMutableFeatureGate
	originalGate := DefaultFeatureGate
	defer func() {
		DefaultMutableFeatureGate = originalMutableGate
		DefaultFeatureGate = originalGate
	}()

	// Create a new feature gate for testing
	DefaultMutableFeatureGate = featuregate.NewFeatureGate()
	DefaultFeatureGate = DefaultMutableFeatureGate
	if err := DefaultMutableFeatureGate.Add(defaultKCPOperatorFeatureGates); err != nil {
		t.Fatalf("failed to add feature gates: %v", err)
	}

	// Test default (disabled)
	if Enabled(ConfigurationBundle) {
		t.Error("ConfigurationBundle should be disabled by default")
	}

	// Enable and test
	if err := SetFeatureGateDuringTest(ConfigurationBundle, true); err != nil {
		t.Fatalf("failed to enable feature gate: %v", err)
	}

	if !Enabled(ConfigurationBundle) {
		t.Error("ConfigurationBundle should be enabled after setting")
	}
}
