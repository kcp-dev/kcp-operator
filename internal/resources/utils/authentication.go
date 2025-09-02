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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kcp-dev/kcp-operator/internal/resources"
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

	if val := config.CAFileRef; val != nil {
		extraArgs = append(extraArgs, fmt.Sprintf("--oidc-ca-file=/etc/kcp/tls/oidc/%s", val.Key))

		podSpec.Volumes = append(deployment.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: "oidc-ca-file",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: val.Name,
				},
			},
		})

		podSpec.Containers[0].VolumeMounts = append(podSpec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      "oidc-ca-file",
			MountPath: "/etc/kcp/tls/oidc",
			ReadOnly:  true,
		})
	}

	podSpec.Containers[0].Args = append(podSpec.Containers[0].Args, extraArgs...)
	deployment.Spec.Template.Spec = podSpec

	return deployment
}

func applyServiceAccountAuthentication(deployment *appsv1.Deployment, rootShard *operatorv1alpha1.RootShard) *appsv1.Deployment {
	// Secrets and volumes

	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}

	// Root shard is not on the list, so we add it manually
	volumes = append(volumes, corev1.Volume{
		Name: resources.GetRootShardCertificateName(rootShard, operatorv1alpha1.ServiceAccountCertificate),
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: resources.GetRootShardCertificateName(rootShard, operatorv1alpha1.ServiceAccountCertificate),
			},
		},
	})

	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      resources.GetRootShardCertificateName(rootShard, operatorv1alpha1.ServiceAccountCertificate),
		ReadOnly:  true,
		MountPath: fmt.Sprintf("/etc/kcp/tls/%s/%s", rootShard.Name, string(operatorv1alpha1.ServiceAccountCertificate)),
	})

	for _, shard := range rootShard.Status.Shards {
		volumes = append(volumes, corev1.Volume{
			Name: resources.GetShardCertificateName(&operatorv1alpha1.Shard{ObjectMeta: metav1.ObjectMeta{Name: shard.Name}}, operatorv1alpha1.ServiceAccountCertificate),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.GetShardCertificateName(&operatorv1alpha1.Shard{ObjectMeta: metav1.ObjectMeta{Name: shard.Name}}, operatorv1alpha1.ServiceAccountCertificate),
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      resources.GetShardCertificateName(&operatorv1alpha1.Shard{ObjectMeta: metav1.ObjectMeta{Name: shard.Name}}, operatorv1alpha1.ServiceAccountCertificate),
			ReadOnly:  true,
			MountPath: fmt.Sprintf("/etc/kcp/tls/%s/%s", shard.Name, string(operatorv1alpha1.ServiceAccountCertificate)),
		})
	}

	deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, volumes...)
	deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(deployment.Spec.Template.Spec.Containers[0].VolumeMounts, volumeMounts...)

	podSpec := deployment.Spec.Template.Spec

	extraArgs := []string{}
	extraArgs = append(extraArgs, "--service-account-lookup=false")
	extraArgs = append(extraArgs, fmt.Sprintf("--service-account-key-file=/etc/kcp/tls/%s/service-account/tls.key", rootShard.Name))

	for _, shard := range rootShard.Status.Shards {
		extraArgs = append(extraArgs, fmt.Sprintf("--service-account-key-file=/etc/kcp/tls/%s/service-account/tls.key", shard.Name))
	}

	podSpec.Containers[0].Args = append(podSpec.Containers[0].Args, extraArgs...)
	deployment.Spec.Template.Spec = podSpec

	return deployment
}
