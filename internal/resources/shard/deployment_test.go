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

package shard

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func TestDeploymentReconciler(t *testing.T) {
	tests := []struct {
		name           string
		shard          *operatorv1alpha1.Shard
		rootShard      *operatorv1alpha1.RootShard
		expectedName   string
		validateDeploy func(*testing.T, *appsv1.Deployment)
	}{
		{
			name: "basic deployment configuration",
			shard: &operatorv1alpha1.Shard{
				ObjectMeta: metav1.ObjectMeta{
					Name: "shardy",
				},
			},
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name: "rooty",
				},
			},
			expectedName: resources.GetShardDeploymentName(&operatorv1alpha1.Shard{
				ObjectMeta: metav1.ObjectMeta{Name: "shardy"},
			}),
			validateDeploy: func(t *testing.T, dep *appsv1.Deployment) {
				assert.Equal(t, int32(2), *dep.Spec.Replicas)
				assert.Len(t, dep.Spec.Template.Spec.Containers, 1)

				container := dep.Spec.Template.Spec.Containers[0]
				assert.Equal(t, "kcp", container.Name)
				assert.Equal(t, "/kcp", container.Command[0])

				// Check for required volume mounts
				volumeMountNames := make(map[string]bool)
				for _, vm := range container.VolumeMounts {
					volumeMountNames[vm.Name] = true
				}

				expectedMounts := []string{
					"client-ca",
					"server-ca",
					"server-cert",
					"client-cert",
					"service-account-ca",
					"kcp-ca",
					"client-kubeconfig",
					"service-account-cert",
				}

				for _, expectedMount := range expectedMounts {
					assert.True(t, volumeMountNames[expectedMount], "Expected volume mount %s not found", expectedMount)
				}

				// Check readiness probe
				assert.NotNil(t, container.ReadinessProbe)
				assert.Equal(t, "/readyz", container.ReadinessProbe.HTTPGet.Path)
				assert.Equal(t, "https", container.ReadinessProbe.HTTPGet.Port.StrVal)

				// Check liveness probe
				assert.NotNil(t, container.LivenessProbe)
				assert.Equal(t, "/livez", container.LivenessProbe.HTTPGet.Path)
				assert.Equal(t, "https", container.LivenessProbe.HTTPGet.Port.StrVal)
			},
		},
		{
			name: "basic deployment configuration with authentication webhook",
			shard: &operatorv1alpha1.Shard{
				ObjectMeta: metav1.ObjectMeta{
					Name: "shardy",
				},
				Spec: operatorv1alpha1.ShardSpec{
					CommonShardSpec: operatorv1alpha1.CommonShardSpec{
						Auth: &operatorv1alpha1.AuthSpec{
							Webhook: &operatorv1alpha1.AuthenticationWebhookSpec{
								ConfigSecretName: "test-webhook-config",
							},
						},
					},
				},
			},
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name: "rooty",
				},
			},
			expectedName: resources.GetShardDeploymentName(&operatorv1alpha1.Shard{
				ObjectMeta: metav1.ObjectMeta{Name: "shardy"},
			}),
			validateDeploy: func(t *testing.T, dep *appsv1.Deployment) {
				assert.Equal(t, int32(2), *dep.Spec.Replicas)
				assert.Len(t, dep.Spec.Template.Spec.Containers, 1)

				container := dep.Spec.Template.Spec.Containers[0]
				assert.Equal(t, "kcp", container.Name)
				assert.Equal(t, "/kcp", container.Command[0])

				// Check for required volume mounts
				volumeMountNames := make(map[string]bool)
				for _, vm := range container.VolumeMounts {
					volumeMountNames[vm.Name] = true
				}

				expectedMounts := []string{
					"authentication-webhook-config",
				}

				for _, expectedMount := range expectedMounts {
					assert.True(t, volumeMountNames[expectedMount], "Expected volume mount %s not found", expectedMount)
				}

				// Check for authentication webhook args
				assert.Contains(t, container.Args, "--authentication-token-webhook-config-file=/etc/kcp/authentication/webhook/kubeconfig")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := DeploymentReconciler(tt.shard, tt.rootShard, nil)
			name, reconcilerFunc := factory()

			assert.Equal(t, tt.expectedName, name)

			// Create a base deployment
			dep := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:      "kcp",
									Resources: corev1.ResourceRequirements{},
								},
							},
						},
					},
				},
			}

			// Apply the reconciler
			result, err := reconcilerFunc(dep)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Validate the result
			tt.validateDeploy(t, result)
		})
	}
}
