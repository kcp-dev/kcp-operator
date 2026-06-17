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
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func TestGetImageSettings(t *testing.T) {
	// Derive the expected default values from the ImageTag constant so this test
	// does not need to be updated on every version bump.
	defaultVersion := semver.MustParse(ImageTag)

	tests := []struct {
		name            string
		imageSpec       *operatorv1alpha1.ImageSpec
		expectedImage   string
		expectedVersion string // "major.minor" or empty if unparseable
	}{
		{
			name:            "default settings",
			imageSpec:       nil,
			expectedImage:   fmt.Sprintf("%s:%s", ImageRepository, ImageTag),
			expectedVersion: fmt.Sprintf("%d.%d", defaultVersion.Major(), defaultVersion.Minor()),
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

// TestImageTagMatchesSDKVersion ensures the default kcp image tag stays in sync
// with the github.com/kcp-dev/sdk dependency. When bumping kcp to a new minor
// version, both the go.mod dependency and the ImageTag constant must be updated
// together; this test (run as part of CI) guards against forgetting one of them.
func TestImageTagMatchesSDKVersion(t *testing.T) {
	const sdkModule = "github.com/kcp-dev/sdk"

	sdkVersion, err := requiredModuleVersion(sdkModule)
	if err != nil {
		t.Fatalf("could not determine %s version: %v", sdkModule, err)
	}

	sdkSemver, err := semver.NewVersion(sdkVersion)
	if err != nil {
		t.Fatalf("could not parse %s version %q: %v", sdkModule, sdkVersion, err)
	}

	imageSemver, err := semver.NewVersion(ImageTag)
	if err != nil {
		t.Fatalf("could not parse ImageTag %q: %v", ImageTag, err)
	}

	if imageSemver.Major() != sdkSemver.Major() || imageSemver.Minor() != sdkSemver.Minor() {
		t.Errorf("ImageTag %q (%d.%d) does not match %s %q (%d.%d); update internal/resources/resources.go and .prow.yaml when bumping the kcp SDK",
			ImageTag, imageSemver.Major(), imageSemver.Minor(),
			sdkModule, sdkVersion, sdkSemver.Major(), sdkSemver.Minor())
	}
}

// requiredModuleVersion reads the module's go.mod (found by walking up from the
// working directory) and returns the version of the given required module.
func requiredModuleVersion(module string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	var goModPath string
	for {
		candidate := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(candidate); err == nil {
			goModPath = candidate
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not locate go.mod above working directory")
		}
		dir = parent
	}

	file, err := os.Open(goModPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// Strip the optional "require " prefix and any inline comment.
		line := strings.TrimSpace(scanner.Text())
		line = strings.TrimPrefix(line, "require ")
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}

		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == module {
			return fields[1], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("module %s not found in %s", module, goModPath)
}
