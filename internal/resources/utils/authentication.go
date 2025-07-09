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

func applyOIDCConfiguration(deployment *appsv1.Deployment, config operatorv1alpha1.OIDCConfiguration) *appsv1.Deployment {
	podSpec := deployment.Spec.Template.Spec

	var extraArgs []string

	if val := config.IssuerURL; len(val) > 0 {
		extraArgs = append(extraArgs, fmt.Sprintf("--oidc-issuer-url=%s", val))
	}

	if val := config.ClientID; len(val) > 0 {
		extraArgs = append(extraArgs, fmt.Sprintf("--oidc-client-id=%s", val))
	}

	if val := config.GroupsClaim; len(val) > 0 {
		extraArgs = append(extraArgs, fmt.Sprintf("--oidc-groups-claim=%s", val))
	}

	if val := config.UsernameClaim; len(val) > 0 {
		extraArgs = append(extraArgs, fmt.Sprintf("--oidc-username-claim=%s", val))
	}

	if val := config.UsernamePrefix; len(val) > 0 {
		extraArgs = append(extraArgs, fmt.Sprintf("--oidc-username-prefix=%s", val))
	}

	if val := config.GroupsPrefix; len(val) > 0 {
		extraArgs = append(extraArgs, fmt.Sprintf("--oidc-groups-prefix=%s", val))
	}

	// TODO(mjudeikis): Add support for  when OIDC is not publically trusted --oidc-ca-file=/etc/kcp/tls/oidc/<ca-secret-name>

	podSpec.Containers[0].Args = append(podSpec.Containers[0].Args, extraArgs...)
	deployment.Spec.Template.Spec = podSpec

	return deployment
}
