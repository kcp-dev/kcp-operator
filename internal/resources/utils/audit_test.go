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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func TestApplyAuditWebhookConfiguration(t *testing.T) {
	tests := []struct {
		name           string
		config         *operatorv1alpha1.AuditWebhookSpec
		validateResult func(t *testing.T, deployment *appsv1.Deployment)
	}{
		{
			name:   "nil config does nothing",
			config: nil,
			validateResult: func(t *testing.T, deployment *appsv1.Deployment) {
				container := deployment.Spec.Template.Spec.Containers[0]
				assert.Equal(t, []string{"--existing-arg=value"}, container.Args)
				assert.Empty(t, deployment.Spec.Template.Spec.Volumes)
				assert.Empty(t, container.VolumeMounts)
			},
		},
		{
			name:   "empty config adds nothing",
			config: &operatorv1alpha1.AuditWebhookSpec{},
			validateResult: func(t *testing.T, deployment *appsv1.Deployment) {
				container := deployment.Spec.Template.Spec.Containers[0]
				assert.Equal(t, []string{"--existing-arg=value"}, container.Args)
				assert.Empty(t, deployment.Spec.Template.Spec.Volumes)
				assert.Empty(t, container.VolumeMounts)
			},
		},
		{
			name: "all supported fields produce expected args and mounts",
			config: &operatorv1alpha1.AuditWebhookSpec{
				BatchBufferSize:      100,
				BatchMaxSize:         50,
				BatchMaxWait:         &metav1.Duration{Duration: 30 * time.Second},
				BatchThrottleBurst:   10,
				BatchThrottleEnable:  true,
				BatchThrottleQPS:     "5.5",
				ConfigSecretName:     "audit-webhook-kubeconfig",
				InitialBackoff:       &metav1.Duration{Duration: 2 * time.Second},
				Mode:                 operatorv1alpha1.AuditWebhookBlockingStrictMode,
				TruncateEnabled:      true,
				TruncateMaxBatchSize: 1000,
				TruncateMaxEventSize: 2000,
				Version:              "audit.k8s.io/v1",
			},
			validateResult: func(t *testing.T, deployment *appsv1.Deployment) {
				container := deployment.Spec.Template.Spec.Containers[0]
				args := container.Args

				assert.Contains(t, args, "--existing-arg=value")
				assert.Contains(t, args, "--audit-webhook-batch-buffer-size=100")
				assert.Contains(t, args, "--audit-webhook-batch-max-size=50")
				assert.Contains(t, args, "--audit-webhook-batch-max-wait=30s")
				assert.Contains(t, args, "--audit-webhook-batch-throttle-burst=10")
				assert.Contains(t, args, "--audit-webhook-batch-throttle-enable")
				assert.Contains(t, args, "--audit-webhook-batch-throttle-qps=5.5")
				assert.Contains(t, args, "--audit-webhook-initial-backoff=2s")
				assert.Contains(t, args, "--audit-webhook-mode=blocking-strict")
				assert.Contains(t, args, "--audit-webhook-truncate-enabled")
				assert.Contains(t, args, "--audit-webhook-truncate-max-batch-size=1000")
				assert.Contains(t, args, "--audit-webhook-truncate-max-event-sizes=2000")
				assert.Contains(t, args, "--audit-webhook-version=audit.k8s.io/v1")
				assert.Contains(t, args, "--audit-webhook-config-file=/etc/kcp/audit/webhook/kubeconfig")

				assert.Len(t, deployment.Spec.Template.Spec.Volumes, 1)
				volume := deployment.Spec.Template.Spec.Volumes[0]
				assert.Equal(t, "audit-webhook-config", volume.Name)
				require.NotNil(t, volume.Secret)
				assert.Equal(t, "audit-webhook-kubeconfig", volume.Secret.SecretName)

				assert.Len(t, container.VolumeMounts, 1)
				mount := container.VolumeMounts[0]
				assert.Equal(t, "audit-webhook-config", mount.Name)
				assert.Equal(t, "/etc/kcp/audit/webhook", mount.MountPath)
				assert.True(t, mount.ReadOnly)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployment := &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: "kcp",
									Args: []string{"--existing-arg=value"},
								},
							},
						},
					},
				},
			}

			applyAuditWebhookConfiguration(deployment, tt.config)
			tt.validateResult(t, deployment)
		})
	}
}
