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
	"errors"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func ApplyCommonShardConfig(deployment *appsv1.Deployment, spec *operatorv1alpha1.CommonShardSpec) (*appsv1.Deployment, error) {
	if len(deployment.Spec.Template.Spec.Containers) == 0 {
		//nolint:staticcheck // allow capital letter in error message
		return deployment, errors.New("Deployment does not contain any containers")
	}

	// explicitly set the replicas if it is configured in the RootShard
	// object or if the existing Deployment object doesn't have replicas
	// configured. This will allow a HPA to interact with the replica
	// count.
	if spec.Replicas != nil {
		deployment.Spec.Replicas = spec.Replicas
	} else if deployment.Spec.Replicas == nil {
		deployment.Spec.Replicas = ptr.To[int32](2)
	}

	// set container image
	image, _ := resources.GetImageSettings(spec.Image)
	deployment.Spec.Template.Spec.Containers[0].Image = image

	deployment = applyEtcdConfiguration(deployment, spec.Etcd)
	deployment = applyAuditConfiguration(deployment, spec.Audit)
	deployment = applyAuthorizationConfiguration(deployment, spec.Authorization)

	return deployment, nil
}

func applyEtcdConfiguration(deployment *appsv1.Deployment, config operatorv1alpha1.EtcdConfig) *appsv1.Deployment {
	podSpec := deployment.Spec.Template.Spec

	podSpec.Containers[0].Args = append(
		podSpec.Containers[0].Args,
		fmt.Sprintf("--etcd-servers=%s", strings.Join(config.Endpoints, ",")),
	)

	if config.TLSConfig != nil {
		volumeName := "etcd-client-cert"
		mountPath := "/etc/etcd/tls"

		podSpec.Containers[0].VolumeMounts = append(podSpec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			ReadOnly:  true,
			MountPath: mountPath,
		})

		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: config.TLSConfig.SecretRef.Name,
				},
			},
		})

		podSpec.Containers[0].Args = append(
			podSpec.Containers[0].Args,
			fmt.Sprintf("--etcd-certfile=%s/tls.crt", mountPath),
			fmt.Sprintf("--etcd-keyfile=%s/tls.key", mountPath),
			fmt.Sprintf("--etcd-cafile=%s/ca.crt", mountPath),
		)
	}

	deployment.Spec.Template.Spec = podSpec

	return deployment
}
