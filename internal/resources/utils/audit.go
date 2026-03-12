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

package utils

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func applyAuditConfiguration(deployment *appsv1.Deployment, config *operatorv1alpha1.AuditSpec) *appsv1.Deployment {
	if config == nil {
		return deployment
	}

	applyAuditPolicyConfiguration(deployment, config.Policy)
	applyAuditWebhookConfiguration(deployment, config.Webhook)
	return deployment
}

func applyAuditPolicyConfiguration(deployment *appsv1.Deployment, config *operatorv1alpha1.AuditPolicySpec) {
	if config == nil {
		return
	}

	podSpec := deployment.Spec.Template.Spec

	var extraArgs []string

	if config.ConfigMap != nil {
		volumeName := "audit-policy"
		mountPath := "/etc/kcp/audit/policy"

		podSpec.Volumes = append(podSpec.Volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: config.ConfigMap.Name,
					},
				},
			},
		})
		podSpec.Containers[0].VolumeMounts = append(podSpec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			ReadOnly:  true,
			MountPath: mountPath,
		})
		extraArgs = append(extraArgs, fmt.Sprintf("--audit-policy-file=%s/%s", mountPath, config.ConfigMap.Key))
	}

	podSpec.Containers[0].Args = append(podSpec.Containers[0].Args, extraArgs...)
	deployment.Spec.Template.Spec = podSpec
}

func applyAuditWebhookConfiguration(deployment *appsv1.Deployment, config *operatorv1alpha1.AuditWebhookSpec) {
	if config == nil {
		return
	}

	podSpec := deployment.Spec.Template.Spec

	var extraArgs []string

	if val := config.BatchBufferSize; val != 0 {
		extraArgs = append(extraArgs, fmt.Sprintf("--audit-webhook-batch-buffer-size=%d", val))
	}

	if val := config.BatchMaxSize; val != 0 {
		extraArgs = append(extraArgs, fmt.Sprintf("--audit-webhook-batch-max-size=%d", val))
	}

	if val := config.BatchMaxWait; val != nil {
		extraArgs = append(extraArgs, fmt.Sprintf("--audit-webhook-batch-max-wait=%v", val.String()))
	}

	if val := config.BatchThrottleBurst; val != 0 {
		extraArgs = append(extraArgs, fmt.Sprintf("--audit-webhook-batch-throttle-burst=%d", val))
	}

	if val := config.BatchThrottleEnable; val {
		extraArgs = append(extraArgs, "--audit-webhook-batch-throttle-enable")
	}

	if val := config.BatchThrottleQPS; val != "" {
		extraArgs = append(extraArgs, fmt.Sprintf("--audit-webhook-batch-throttle-qps=%s", val))
	}

	if val := config.InitialBackoff; val != nil {
		extraArgs = append(extraArgs, fmt.Sprintf("--audit-webhook-initial-backoff=%v", val.String()))
	}

	if val := config.Mode; val != "" {
		extraArgs = append(extraArgs, fmt.Sprintf("--audit-webhook-mode=%s", val))
	}

	if val := config.TruncateEnabled; val {
		extraArgs = append(extraArgs, "--audit-webhook-truncate-enabled")
	}

	if val := config.TruncateMaxBatchSize; val != 0 {
		extraArgs = append(extraArgs, fmt.Sprintf("--audit-webhook-truncate-max-batch-size=%d", val))
	}

	if val := config.TruncateMaxEventSize; val != 0 {
		extraArgs = append(extraArgs, fmt.Sprintf("--audit-webhook-truncate-max-event-sizes=%d", val))
	}

	if val := config.Version; val != "" {
		extraArgs = append(extraArgs, fmt.Sprintf("--audit-webhook-version=%s", val))
	}

	if val := config.ConfigSecretName; val != "" {
		volumeName := "audit-webhook-config"
		mountPath := "/etc/kcp/audit/webhook"

		extraArgs = append(extraArgs, fmt.Sprintf("--audit-webhook-config-file=%s/kubeconfig", mountPath))
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
}
