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
	"maps"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func ApplyDeploymentTemplate(dep *appsv1.Deployment, tpl *operatorv1alpha1.DeploymentTemplate) *appsv1.Deployment {
	if tpl == nil {
		return dep
	}

	if metadata := tpl.Metadata; metadata != nil {
		dep.Annotations = addNewKeys(dep.Annotations, metadata.Annotations)
		dep.Labels = addNewKeys(dep.Labels, metadata.Labels)
	}

	applyDeploymentSpecTemplate(&dep.Spec, tpl.Spec)

	return dep
}

func applyDeploymentSpecTemplate(spec *appsv1.DeploymentSpec, tpl *operatorv1alpha1.DeploymentSpecTemplate) {
	if tpl == nil {
		return
	}

	applyPodTemplateSpec(&spec.Template, tpl.Template)
}

func applyPodTemplateSpec(templateSpec *corev1.PodTemplateSpec, tpl *operatorv1alpha1.PodTemplateSpec) {
	if tpl == nil {
		return
	}

	if metadata := tpl.Metadata; metadata != nil {
		templateSpec.Annotations = addNewKeys(templateSpec.Annotations, metadata.Annotations)
		templateSpec.Labels = addNewKeys(templateSpec.Labels, metadata.Labels)
	}

	applyPodSpecTemplate(&templateSpec.Spec, tpl.Spec)
}

func applyPodSpecTemplate(spec *corev1.PodSpec, tpl *operatorv1alpha1.PodSpecTemplate) {
	if tpl == nil {
		return
	}

	spec.Affinity = tpl.Affinity
	spec.NodeSelector = tpl.NodeSelector
	spec.Tolerations = tpl.Tolerations
	spec.HostAliases = tpl.HostAliases
	spec.ImagePullSecrets = tpl.ImagePullSecrets
}

func ApplyResources(container corev1.Container, resources *corev1.ResourceRequirements) corev1.Container {
	if resources == nil {
		return container
	}

	if container.Resources.Limits == nil {
		container.Resources.Limits = make(corev1.ResourceList)
	}
	if container.Resources.Requests == nil {
		container.Resources.Requests = make(corev1.ResourceList)
	}

	maps.Copy(container.Resources.Limits, resources.Limits)
	maps.Copy(container.Resources.Requests, resources.Requests)

	return container
}

func ApplyAuthConfiguration(deployment *appsv1.Deployment, config *operatorv1alpha1.AuthSpec) *appsv1.Deployment {
	if config == nil {
		return deployment
	}

	if config.OIDC != nil {
		deployment = applyOIDCConfiguration(deployment, *config.OIDC)
	}

	return deployment
}

func ApplyFrontProxyAuthConfiguration(deployment *appsv1.Deployment, config *operatorv1alpha1.AuthSpec, rootShard *operatorv1alpha1.RootShard) *appsv1.Deployment {
	if config == nil {
		return deployment
	}
	deployment = ApplyAuthConfiguration(deployment, config)

	if config.ServiceAccount != nil && config.ServiceAccount.Enabled {
		deployment = applyServiceAccountAuthentication(deployment, rootShard)
	}

	return deployment
}
