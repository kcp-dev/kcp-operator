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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func newTestDeploymentWithContainer() *appsv1.Deployment {
	return &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name: "kcp",
						Args: []string{"--existing-arg=value"},
					}},
				},
			},
		},
	}
}

func TestApplyExtraVolumes(t *testing.T) {
	tests := []struct {
		name           string
		volumes        []corev1.Volume
		volumeMounts   []corev1.VolumeMount
		validateResult func(t *testing.T, deployment *appsv1.Deployment)
	}{
		{
			name:         "no extra volumes or mounts does nothing",
			volumes:      nil,
			volumeMounts: nil,
			validateResult: func(t *testing.T, deployment *appsv1.Deployment) {
				assert.Empty(t, deployment.Spec.Template.Spec.Volumes)
				assert.Empty(t, deployment.Spec.Template.Spec.Containers[0].VolumeMounts)
			},
		},
		{
			name: "encryption config volume and mount are added",
			volumes: []corev1.Volume{{
				Name: "encryption-config",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "kcp-encryption-config",
					},
				},
			}},
			volumeMounts: []corev1.VolumeMount{{
				Name:      "encryption-config",
				ReadOnly:  true,
				MountPath: "/etc/kcp/encryption",
			}},
			validateResult: func(t *testing.T, deployment *appsv1.Deployment) {
				require.Len(t, deployment.Spec.Template.Spec.Volumes, 1)
				assert.Equal(t, "encryption-config", deployment.Spec.Template.Spec.Volumes[0].Name)
				require.NotNil(t, deployment.Spec.Template.Spec.Volumes[0].Secret)
				assert.Equal(t, "kcp-encryption-config", deployment.Spec.Template.Spec.Volumes[0].Secret.SecretName)

				container := deployment.Spec.Template.Spec.Containers[0]
				require.Len(t, container.VolumeMounts, 1)
				assert.Equal(t, "encryption-config", container.VolumeMounts[0].Name)
				assert.Equal(t, "/etc/kcp/encryption", container.VolumeMounts[0].MountPath)
				assert.True(t, container.VolumeMounts[0].ReadOnly)
			},
		},
		{
			name: "extra volumes are appended, not replacing existing ones",
			volumes: []corev1.Volume{{
				Name: "extra",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			}},
			volumeMounts: []corev1.VolumeMount{{
				Name:      "extra",
				MountPath: "/extra",
			}},
			validateResult: func(t *testing.T, deployment *appsv1.Deployment) {
				volumeNames := make([]string, 0, len(deployment.Spec.Template.Spec.Volumes))
				for _, v := range deployment.Spec.Template.Spec.Volumes {
					volumeNames = append(volumeNames, v.Name)
				}
				assert.Contains(t, volumeNames, "pre-existing")
				assert.Contains(t, volumeNames, "extra")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployment := newTestDeploymentWithContainer()

			if tt.name == "extra volumes are appended, not replacing existing ones" {
				deployment.Spec.Template.Spec.Volumes = []corev1.Volume{{
					Name: "pre-existing",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				}}
			}

			result := applyExtraVolumes(deployment, tt.volumes, tt.volumeMounts)

			tt.validateResult(t, result)
		})
	}
}

func TestApplyCommonShardConfigAppliesExtraVolumes(t *testing.T) {
	deployment := newTestDeploymentWithContainer()

	spec := &operatorv1alpha1.CommonShardSpec{
		Etcd: operatorv1alpha1.EtcdConfig{
			Endpoints: []string{"https://etcd:2379"},
		},
		ExtraVolumes: []corev1.Volume{{
			Name: "encryption-config",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "kcp-encryption-config",
				},
			},
		}},
		ExtraVolumeMounts: []corev1.VolumeMount{{
			Name:      "encryption-config",
			ReadOnly:  true,
			MountPath: "/etc/kcp/encryption",
		}},
	}

	result := ApplyCommonShardConfig(deployment, spec)

	volumeNames := make(map[string]corev1.Volume)
	for _, v := range result.Spec.Template.Spec.Volumes {
		volumeNames[v.Name] = v
	}

	assert.Contains(t, volumeNames, "encryption-config")
	assert.Equal(t, "kcp-encryption-config", volumeNames["encryption-config"].Secret.SecretName)

	container := result.Spec.Template.Spec.Containers[0]
	mountFound := false
	for _, vm := range container.VolumeMounts {
		if vm.Name == "encryption-config" {
			mountFound = true
			assert.Equal(t, "/etc/kcp/encryption", vm.MountPath)
			assert.True(t, vm.ReadOnly)
		}
	}
	assert.True(t, mountFound, "expected encryption-config volume mount on container")
}
