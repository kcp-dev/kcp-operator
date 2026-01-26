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
	"fmt"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/component-base/featuregate"
)

const (
	// ConfigurationBundle enables the configuration bundle feature for managing
	// bundled KCP configurations and resources.
	ConfigurationBundle featuregate.Feature = "ConfigurationBundle"
)

var (
	// DefaultMutableFeatureGate is a mutable version of DefaultFeatureGate.
	// Only top-level commands should make use of this.
	DefaultMutableFeatureGate featuregate.MutableFeatureGate = featuregate.NewFeatureGate()

	// DefaultFeatureGate is a shared global FeatureGate.
	DefaultFeatureGate featuregate.FeatureGate = DefaultMutableFeatureGate

	// defaultKCPOperatorFeatureGates consists of all known KCP operator-specific feature keys.
	// To add a new feature, define a key for it above and add it here.
	defaultKCPOperatorFeatureGates = map[featuregate.Feature]featuregate.FeatureSpec{
		ConfigurationBundle: {Default: false, PreRelease: featuregate.Alpha},
	}
)

func init() {
	runtime.Must(DefaultMutableFeatureGate.Add(defaultKCPOperatorFeatureGates))
}

// Enabled returns true if the feature gate is enabled.
func Enabled(feature featuregate.Feature) bool {
	return DefaultFeatureGate.Enabled(feature)
}

// SetFeatureGateDuringTest sets the specified gate to the specified value for testing purposes.
// This should only be used in tests.
func SetFeatureGateDuringTest(feature featuregate.Feature, enabled bool) error {
	if DefaultMutableFeatureGate == nil {
		return fmt.Errorf("feature gate is not initialized")
	}
	return DefaultMutableFeatureGate.SetFromMap(map[string]bool{string(feature): enabled})
}
