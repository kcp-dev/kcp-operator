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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func TestApplyServiceAccountAuthentication(t *testing.T) {
	tests := []struct {
		name           string
		rootShard      *operatorv1alpha1.RootShard
		initialDeploy  *appsv1.Deployment
		validateDeploy func(*testing.T, *appsv1.Deployment)
	}{
		{
			name: "root shard only - no additional shards",
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-root-shard",
				},
				Status: operatorv1alpha1.RootShardStatus{
					Shards: []operatorv1alpha1.ShardReference{},
				},
			},
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
			validateDeploy: func(t *testing.T, dep *appsv1.Deployment) {
				container := dep.Spec.Template.Spec.Containers[0]

				// Check volumes
				assert.Len(t, dep.Spec.Template.Spec.Volumes, 1)
				rootShardVolume := dep.Spec.Template.Spec.Volumes[0]
				expectedVolumeName := resources.GetRootShardCertificateName(&operatorv1alpha1.RootShard{ObjectMeta: metav1.ObjectMeta{Name: "test-root-shard"}}, operatorv1alpha1.ServiceAccountCertificate)
				assert.Equal(t, expectedVolumeName, rootShardVolume.Name)
				assert.NotNil(t, rootShardVolume.Secret)
				assert.Equal(t, expectedVolumeName, rootShardVolume.Secret.SecretName)

				// Check volume mounts
				assert.Len(t, container.VolumeMounts, 1)
				rootShardMount := container.VolumeMounts[0]
				assert.Equal(t, expectedVolumeName, rootShardMount.Name)
				assert.True(t, rootShardMount.ReadOnly)
				assert.Equal(t, "/etc/kcp/tls/test-root-shard/service-account", rootShardMount.MountPath)

				// Check args
				args := container.Args
				assert.Contains(t, args, "--existing-arg=value")
				assert.Contains(t, args, "--service-account-lookup=false")
				assert.Contains(t, args, "--service-account-key-file=/etc/kcp/tls/test-root-shard/service-account/tls.key")
				assert.Len(t, args, 3)
			},
		},
		{
			name: "root shard with multiple additional shards",
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-root-shard",
				},
				Status: operatorv1alpha1.RootShardStatus{
					Shards: []operatorv1alpha1.ShardReference{
						{Name: "shard-1"},
						{Name: "shard-2"},
					},
				},
			},
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
									Args: []string{},
								},
							},
							Volumes: []corev1.Volume{},
						},
					},
				},
			},
			validateDeploy: func(t *testing.T, dep *appsv1.Deployment) {
				container := dep.Spec.Template.Spec.Containers[0]

				// Check volumes - should have root shard + 2 additional shards
				assert.Len(t, dep.Spec.Template.Spec.Volumes, 3)

				volumeNames := make(map[string]bool)
				for _, volume := range dep.Spec.Template.Spec.Volumes {
					volumeNames[volume.Name] = true
					assert.NotNil(t, volume.Secret)
					assert.Equal(t, volume.Name, volume.Secret.SecretName)
				}

				expectedRootVolume := resources.GetRootShardCertificateName(&operatorv1alpha1.RootShard{ObjectMeta: metav1.ObjectMeta{Name: "test-root-shard"}}, operatorv1alpha1.ServiceAccountCertificate)
				expectedShard1Volume := resources.GetShardCertificateName(&operatorv1alpha1.Shard{ObjectMeta: metav1.ObjectMeta{Name: "shard-1"}}, operatorv1alpha1.ServiceAccountCertificate)
				expectedShard2Volume := resources.GetShardCertificateName(&operatorv1alpha1.Shard{ObjectMeta: metav1.ObjectMeta{Name: "shard-2"}}, operatorv1alpha1.ServiceAccountCertificate)

				assert.True(t, volumeNames[expectedRootVolume], "Root shard volume not found")
				assert.True(t, volumeNames[expectedShard1Volume], "Shard-1 volume not found")
				assert.True(t, volumeNames[expectedShard2Volume], "Shard-2 volume not found")

				// Check volume mounts
				assert.Len(t, container.VolumeMounts, 3)

				mountNames := make(map[string]string)
				for _, mount := range container.VolumeMounts {
					mountNames[mount.Name] = mount.MountPath
					assert.True(t, mount.ReadOnly)
				}

				assert.Equal(t, "/etc/kcp/tls/test-root-shard/service-account", mountNames[expectedRootVolume])
				assert.Equal(t, "/etc/kcp/tls/shard-1/service-account", mountNames[expectedShard1Volume])
				assert.Equal(t, "/etc/kcp/tls/shard-2/service-account", mountNames[expectedShard2Volume])

				// Check args
				args := container.Args
				assert.Contains(t, args, "--service-account-lookup=false")
				assert.Contains(t, args, "--service-account-key-file=/etc/kcp/tls/test-root-shard/service-account/tls.key")
				assert.Contains(t, args, "--service-account-key-file=/etc/kcp/tls/shard-1/service-account/tls.key")
				assert.Contains(t, args, "--service-account-key-file=/etc/kcp/tls/shard-2/service-account/tls.key")
				assert.Len(t, args, 4)
			},
		},
		{
			name: "preserves existing volumes and volume mounts",
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-root-shard",
				},
				Status: operatorv1alpha1.RootShardStatus{
					Shards: []operatorv1alpha1.ShardReference{
						{Name: "shard-1"},
					},
				},
			},
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
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "existing-mount",
											MountPath: "/existing/path",
											ReadOnly:  true,
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "existing-volume",
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "existing-configmap",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			validateDeploy: func(t *testing.T, dep *appsv1.Deployment) {
				container := dep.Spec.Template.Spec.Containers[0]

				// Check volumes - should preserve existing + add new ones
				assert.Len(t, dep.Spec.Template.Spec.Volumes, 3)

				volumeNames := make(map[string]bool)
				for _, volume := range dep.Spec.Template.Spec.Volumes {
					volumeNames[volume.Name] = true
				}

				assert.True(t, volumeNames["existing-volume"], "Existing volume should be preserved")

				// Check volume mounts - should preserve existing + add new ones
				assert.Len(t, container.VolumeMounts, 3)

				mountNames := make(map[string]bool)
				for _, mount := range container.VolumeMounts {
					mountNames[mount.Name] = true
				}

				assert.True(t, mountNames["existing-mount"], "Existing volume mount should be preserved")

				// Check args - should preserve existing + add new ones
				args := container.Args
				assert.Contains(t, args, "--existing-arg=value")
				assert.Contains(t, args, "--service-account-lookup=false")
				assert.Len(t, args, 4) // existing + service-account-lookup + 2 key-file args
			},
		},
		{
			name: "empty shards list",
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-root-shard",
				},
				Status: operatorv1alpha1.RootShardStatus{
					Shards: nil,
				},
			},
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
									Args: []string{},
								},
							},
							Volumes: []corev1.Volume{},
						},
					},
				},
			},
			validateDeploy: func(t *testing.T, dep *appsv1.Deployment) {
				container := dep.Spec.Template.Spec.Containers[0]

				// Should only have root shard volume
				assert.Len(t, dep.Spec.Template.Spec.Volumes, 1)
				assert.Len(t, container.VolumeMounts, 1)

				// Should only have root shard args
				args := container.Args
				assert.Contains(t, args, "--service-account-lookup=false")
				assert.Contains(t, args, "--service-account-key-file=/etc/kcp/tls/test-root-shard/service-account/tls.key")
				assert.Len(t, args, 2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyServiceAccountAuthentication(tt.initialDeploy, tt.rootShard)

			require.NotNil(t, result)
			assert.Equal(t, tt.initialDeploy, result, "Function should return the same deployment instance")

			tt.validateDeploy(t, result)
		})
	}
}
