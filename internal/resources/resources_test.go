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

package resources

import (
	"testing"

	"github.com/Masterminds/semver/v3"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func TestGetImageSettings(t *testing.T) {
	tests := []struct {
		name            string
		imageSpec       *operatorv1alpha1.ImageSpec
		expectedImage   string
		expectedVersion string // "major.minor" or empty if unparseable
	}{
		{
			name:            "default settings",
			imageSpec:       nil,
			expectedImage:   "ghcr.io/kcp-dev/kcp:v0.31.0",
			expectedVersion: "0.31",
		},
		{
			name: "custom tag with valid semver",
			imageSpec: &operatorv1alpha1.ImageSpec{
				Tag: "v0.28.1",
			},
			expectedImage:   "ghcr.io/kcp-dev/kcp:v0.28.1",
			expectedVersion: "0.28",
		},
		{
			name: "custom tag with invalid semver",
			imageSpec: &operatorv1alpha1.ImageSpec{
				Tag: "latest",
			},
			expectedImage:   "ghcr.io/kcp-dev/kcp:latest",
			expectedVersion: "",
		},
		{
			name: "custom repository and tag",
			imageSpec: &operatorv1alpha1.ImageSpec{
				Repository: "custom.registry/kcp",
				Tag:        "v0.29.0",
			},
			expectedImage:   "custom.registry/kcp:v0.29.0",
			expectedVersion: "0.29",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			image, _, version := GetImageSettings(tt.imageSpec)
			if image != tt.expectedImage {
				t.Errorf("expected image %q, got %q", tt.expectedImage, image)
			}

			if version != nil {
				v, _ := semver.NewVersion(tt.expectedVersion)
				if v == nil || version.Major() != v.Major() || version.Minor() != v.Minor() {
					t.Errorf("expected version %q, got %d.%d", tt.expectedVersion, version.Major(), version.Minor())
				}
			} else if tt.expectedVersion != "" {
				t.Errorf("expected version %q, got nil", tt.expectedVersion)
			}
		})
	}
}
