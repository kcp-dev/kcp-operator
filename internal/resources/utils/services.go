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
	corev1 "k8s.io/api/core/v1"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func ApplyServiceTemplate(svc *corev1.Service, tpl *operatorv1alpha1.ServiceTemplate) *corev1.Service {
	if tpl == nil {
		return svc
	}

	if metadata := tpl.Metadata; metadata != nil {
		svc.Annotations = addNewKeys(svc.Annotations, metadata.Annotations)
		svc.Labels = addNewKeys(svc.Labels, metadata.Labels)
	}

	if spec := tpl.Spec; spec != nil {
		if spec.Type != "" {
			svc.Spec.Type = spec.Type
		}

		if spec.ClusterIP != "" {
			svc.Spec.ClusterIP = spec.ClusterIP
		}
	}

	return svc
}
