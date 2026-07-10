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

func TestApplyAuthorizationConfiguration(t *testing.T) {
	tests := []struct {
		name              string
		initialDeploy     *appsv1.Deployment
		authorizationSpec *operatorv1alpha1.AuthorizationSpec
		validateDeploy    func(*testing.T, *appsv1.Deployment)
	}{
		{
			name: "authorization fully configured",
			initialDeploy: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-deployment",
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: "test-container",
									Args: []string{"--existing-arg=value"},
								},
							},
							Volumes: []corev1.Volume{},
						},
					},
				},
			},
			authorizationSpec: &operatorv1alpha1.AuthorizationSpec{
				AllowPaths: &[]string{"/healthz", "/readyz", "/livez"},
				Order:      &[]string{"AlwaysAllowGroups", "AlwaysAllowPaths", "RBAC", "Webhook"},
				Webhook: &operatorv1alpha1.AuthorizationWebhookSpec{
					CacheAuthorizedTTL: &metav1.Duration{
						Duration: time.Second * 5,
					},
					CacheUnauthorizedTTL: &metav1.Duration{
						Duration: time.Second * 10,
					},
					ConfigSecretName: "test-webhook-config",
				},
			},
			validateDeploy: func(t *testing.T, dep *appsv1.Deployment) {
				container := dep.Spec.Template.Spec.Containers[0]
				// assert args
				args := container.Args
				assert.Contains(t, args, "--authorization-always-allow-paths=/healthz,/readyz,/livez")
				assert.Contains(t, args, "--authorization-order=AlwaysAllowGroups,AlwaysAllowPaths,RBAC,Webhook")
				assert.Contains(t, args, "--authorization-webhook-config-file=/etc/kcp/authorization/webhook/kubeconfig")
				assert.Contains(t, args, "--authorization-webhook-cache-authorized-ttl=5s")
				assert.Contains(t, args, "--authorization-webhook-cache-unauthorized-ttl=10s")
				assert.Contains(t, args, "--existing-arg=value")
				assert.Len(t, args, 6)
				// assert volumes
				assert.Len(t, dep.Spec.Template.Spec.Volumes, 1)
				volume := dep.Spec.Template.Spec.Volumes[0]
				assert.Equal(t, "authorization-webhook-config", volume.Name)
				require.NotNil(t, volume.Secret)
				assert.Equal(t, "test-webhook-config", volume.Secret.SecretName)
				// assert volume mounts
				assert.Len(t, container.VolumeMounts, 1)
				volumeMount := container.VolumeMounts[0]
				assert.Equal(t, "authorization-webhook-config", volumeMount.Name)
				assert.True(t, volumeMount.ReadOnly)
				assert.Equal(t, "/etc/kcp/authorization/webhook", volumeMount.MountPath)
			},
		},
		{
			name: "empty order and empty allow paths set empty flag",
			initialDeploy: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-deployment",
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: "test-container",
									Args: []string{"--existing-arg=value"},
								},
							},
							Volumes: []corev1.Volume{},
						},
					},
				},
			},
			authorizationSpec: &operatorv1alpha1.AuthorizationSpec{
				AllowPaths: &[]string{},
				Order:      &[]string{},
			},
			validateDeploy: func(t *testing.T, dep *appsv1.Deployment) {
				container := dep.Spec.Template.Spec.Containers[0]
				// assert args
				args := container.Args
				assert.Contains(t, args, "--authorization-always-allow-paths=")
				assert.Contains(t, args, "--authorization-order=")
				assert.Contains(t, args, "--existing-arg=value")
				assert.Len(t, args, 3)
				// assert volumes are unchanged
				assert.Len(t, dep.Spec.Template.Spec.Volumes, 0)
				// assert volume mounts are unchanged
				assert.Len(t, container.VolumeMounts, 0)
			},
		},
		{
			name: "nil order and nil allow paths do not set flag",
			initialDeploy: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-deployment",
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: "test-container",
									Args: []string{"--existing-arg=value"},
								},
							},
							Volumes: []corev1.Volume{},
						},
					},
				},
			},
			authorizationSpec: &operatorv1alpha1.AuthorizationSpec{
				AllowPaths: nil,
				Order:      nil,
			},
			validateDeploy: func(t *testing.T, dep *appsv1.Deployment) {
				container := dep.Spec.Template.Spec.Containers[0]
				// assert args
				args := container.Args
				assert.Contains(t, args, "--existing-arg=value")
				assert.Len(t, args, 1)
				// assert volumes are unchanged
				assert.Len(t, dep.Spec.Template.Spec.Volumes, 0)
				// assert volume mounts are unchanged
				assert.Len(t, container.VolumeMounts, 0)
			},
		},
		{
			name: "nil authorization spec - should not modify deployment",
			initialDeploy: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-deployment",
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: "test-container",
									Args: []string{"--existing-arg=value"},
								},
							},
							Volumes: []corev1.Volume{},
						},
					},
				},
			},
			authorizationSpec: nil,
			validateDeploy: func(t *testing.T, dep *appsv1.Deployment) {
				container := dep.Spec.Template.Spec.Containers[0]
				// assert args are unchanged
				args := container.Args
				assert.Contains(t, args, "--existing-arg=value")
				assert.Len(t, args, 1)
				// assert volumes are unchanged
				assert.Len(t, dep.Spec.Template.Spec.Volumes, 0)
				// assert volume mounts are unchanged
				assert.Len(t, container.VolumeMounts, 0)
			},
		},
		{
			name: "empty authorization spec - should not modify deployment",
			initialDeploy: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-deployment",
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: "test-container",
									Args: []string{"--existing-arg=value"},
								},
							},
							Volumes: []corev1.Volume{},
						},
					},
				},
			},
			authorizationSpec: &operatorv1alpha1.AuthorizationSpec{},
			validateDeploy: func(t *testing.T, dep *appsv1.Deployment) {
				container := dep.Spec.Template.Spec.Containers[0]
				args := container.Args
				assert.Contains(t, args, "--existing-arg=value")
				assert.Len(t, args, 1)
				// assert volumes are unchanged
				assert.Len(t, dep.Spec.Template.Spec.Volumes, 0)
				// assert volume mounts are unchanged
				assert.Len(t, container.VolumeMounts, 0)
			},
		},
		{
			name: "nil durations in spec - should not modify deployment",
			initialDeploy: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-deployment",
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: "test-container",
									Args: []string{
										"--existing-arg=value",
										"--authorization-webhook-cache-authorized-ttl=5s",
									},
								},
							},
							Volumes: []corev1.Volume{},
						},
					},
				},
			},
			authorizationSpec: &operatorv1alpha1.AuthorizationSpec{
				Webhook: &operatorv1alpha1.AuthorizationWebhookSpec{
					CacheAuthorizedTTL:   nil,
					CacheUnauthorizedTTL: nil,
				},
			},
			validateDeploy: func(t *testing.T, dep *appsv1.Deployment) {
				container := dep.Spec.Template.Spec.Containers[0]
				// assert args are unchanged
				args := container.Args
				assert.Contains(t, args, "--authorization-webhook-cache-authorized-ttl=5s")
				assert.Contains(t, args, "--existing-arg=value")
				assert.Len(t, args, 2)
				// assert volumes are unchanged
				assert.Len(t, dep.Spec.Template.Spec.Volumes, 0)
				// assert volume mounts are unchanged
				assert.Len(t, container.VolumeMounts, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyAuthorizationConfiguration(tt.initialDeploy, tt.authorizationSpec)

			require.NotNil(t, result)
			assert.Equal(t, tt.initialDeploy, result, "Function should return the same deployment instance")

			tt.validateDeploy(t, result)
		})
	}
}
