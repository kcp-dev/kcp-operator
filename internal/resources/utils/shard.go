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
	"fmt"

	appsv1 "k8s.io/api/apps/v1"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func ApplyCommonShardConfig(deployment *appsv1.Deployment, spec *operatorv1alpha1.CommonShardSpec) (*appsv1.Deployment, error) {
	deployment, err := applyAuditConfiguration(deployment, spec.Audit)
	if err != nil {
		return nil, fmt.Errorf("failed to apply audit configuration: %w", err)
	}

	deployment, err = applyAuthorizationConfiguration(deployment, spec.Authorization)
	if err != nil {
		return nil, fmt.Errorf("failed to apply authorization configuration: %w", err)
	}

	return deployment, nil
}
