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
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func getCommonShardBatteries() []string {
	return []string{"workspace-types"}
}

func GetShardBatteries(shard *operatorv1alpha1.Shard) []string {
	return getCommonShardBatteries()
}

func GetRootShardBatteries(rootShard *operatorv1alpha1.RootShard) []string {
	return getCommonShardBatteries()
}

func ApplyCommonShardDeploymentProperties(deployment *appsv1.Deployment) *appsv1.Deployment {
	if len(deployment.Spec.Template.Spec.Containers) == 0 {
		panic("Deployment does not contain any containers.")
	}

	container := deployment.Spec.Template.Spec.Containers[0]

	container.Ports = []corev1.ContainerPort{
		{
			Name:          "https",
			ContainerPort: 6443,
			Protocol:      corev1.ProtocolTCP,
		},
	}

	container.SecurityContext = &corev1.SecurityContext{
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
		ReadOnlyRootFilesystem:   ptr.To(true),
		AllowPrivilegeEscalation: ptr.To(false),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				corev1.Capability("ALL"),
			},
		},
	}

	container.StartupProbe = &corev1.Probe{
		FailureThreshold:    60,
		InitialDelaySeconds: 10,
		PeriodSeconds:       5,
		SuccessThreshold:    1,
		TimeoutSeconds:      10,
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/readyz",
				Port:   intstr.FromString("https"),
				Scheme: corev1.URISchemeHTTPS,
			},
		},
	}

	container.ReadinessProbe = &corev1.Probe{
		FailureThreshold:    6,
		InitialDelaySeconds: 10,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		TimeoutSeconds:      10,
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/readyz",
				Port:   intstr.FromString("https"),
				Scheme: corev1.URISchemeHTTPS,
			},
		},
	}

	container.LivenessProbe = &corev1.Probe{
		FailureThreshold:    6,
		InitialDelaySeconds: 10,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		TimeoutSeconds:      10,
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/livez",
				Port:   intstr.FromString("https"),
				Scheme: corev1.URISchemeHTTPS,
			},
		},
	}

	deployment.Spec.Template.Spec.Containers[0] = container

	return deployment
}

func ApplyCommonShardConfig(deployment *appsv1.Deployment, spec *operatorv1alpha1.CommonShardSpec) *appsv1.Deployment {
	if len(deployment.Spec.Template.Spec.Containers) == 0 {
		panic("Deployment does not contain any containers.")
	}

	container := deployment.Spec.Template.Spec.Containers[0]

	// override default resource requirements
	container = ApplyResources(container, spec.Resources)

	// explicitly set the replicas if it is configured in the spec object
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
	container.Image = image

	deployment.Spec.Template.Spec.Containers[0] = container

	deployment = applyEtcdConfiguration(deployment, spec.Etcd)
	deployment = applyAuditConfiguration(deployment, spec.Audit)
	deployment = applyAuthorizationConfiguration(deployment, spec.Authorization)

	return deployment
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

		podSpec.Volumes = append(podSpec.Volumes, corev1.Volume{
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
