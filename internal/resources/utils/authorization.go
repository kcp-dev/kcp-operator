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
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func applyAuthorizationConfiguration(deployment *appsv1.Deployment, config *operatorv1alpha1.AuthorizationSpec) *appsv1.Deployment {
	if config == nil || config.Webhook == nil {
		return deployment
	}

	return applyAuthorizationWebhookConfiguration(deployment, *config.Webhook)
}

func applyAuthorizationWebhookConfiguration(deployment *appsv1.Deployment, config operatorv1alpha1.AuthorizationWebhookSpec) *appsv1.Deployment {
	podSpec := deployment.Spec.Template.Spec

	var extraArgs []string

	if vals := config.AllowPaths; len(vals) > 0 {
		extraArgs = append(extraArgs, fmt.Sprintf("--authorization-always-allow-paths=%s", strings.Join(vals, ",")))
	}

	if val := config.CacheAuthorizedTTL; val != nil {
		extraArgs = append(extraArgs, fmt.Sprintf("--authorization-webhook-cache-authorized-ttl=%v", val.String()))
	}

	if val := config.CacheUnauthorizedTTL; val != nil {
		extraArgs = append(extraArgs, fmt.Sprintf("--authorization-webhook-cache-unauthorized-ttl=%v", val.String()))
	}

	if val := config.Version; val != "" {
		extraArgs = append(extraArgs, fmt.Sprintf("--authorization-webhook-version=%s", val))
	}

	if val := config.ConfigSecretName; val != "" {
		volumeName := "authorization-webhook-config"
		mountPath := "/etc/kcp/authorization/webhook"

		extraArgs = append(extraArgs, fmt.Sprintf("--authorization-webhook-config-file=%s/kubeconfig", mountPath))
		podSpec.Containers[0].VolumeMounts = append(podSpec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			ReadOnly:  true,
			MountPath: mountPath,
		})

		podSpec.Volumes = append(podSpec.Volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: val,
				},
			},
		})
	}

	podSpec.Containers[0].Args = append(podSpec.Containers[0].Args, extraArgs...)
	deployment.Spec.Template.Spec = podSpec

	return deployment
}
