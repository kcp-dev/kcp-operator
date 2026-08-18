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

	"k8s.io/component-base/featuregate"
)

// testFeature is a feature key that only exists for the tests in this package; the operator
// currently ships without any feature gates of its own.
const testFeature featuregate.Feature = "TestFeature"

var testFeatureGates = map[featuregate.Feature]featuregate.FeatureSpec{
	testFeature: {Default: false, PreRelease: featuregate.Alpha},
}

func TestDefaultFeatureGates(t *testing.T) {
	// All shipped feature gates must be registered without errors.
	fg := featuregate.NewFeatureGate()
	if err := fg.Add(defaultKCPOperatorFeatureGates); err != nil {
		t.Fatalf("failed to add feature gates: %v", err)
	}

	for feature, spec := range defaultKCPOperatorFeatureGates {
		if enabled := fg.Enabled(feature); enabled != spec.Default {
			t.Errorf("expected feature %q to be %v by default, got %v", feature, spec.Default, enabled)
		}
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
	if err := DefaultMutableFeatureGate.Add(testFeatureGates); err != nil {
		t.Fatalf("failed to add feature gates: %v", err)
	}

	tests := []struct {
		name        string
		feature     featuregate.Feature
		enableValue bool
	}{
		{
			name:        "enable TestFeature",
			feature:     testFeature,
			enableValue: true,
		},
		{
			name:        "disable TestFeature",
			feature:     testFeature,
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
	if err := DefaultMutableFeatureGate.Add(testFeatureGates); err != nil {
		t.Fatalf("failed to add feature gates: %v", err)
	}

	// Test default (disabled)
	if Enabled(testFeature) {
		t.Error("TestFeature should be disabled by default")
	}

	// Enable and test
	if err := SetFeatureGateDuringTest(testFeature, true); err != nil {
		t.Fatalf("failed to enable feature gate: %v", err)
	}

	if !Enabled(testFeature) {
		t.Error("TestFeature should be enabled after setting")
	}
}
