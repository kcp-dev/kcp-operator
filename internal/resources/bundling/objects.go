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

package bundling

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

var (
	// SecretGVR is the GVR for Secret resources
	SecretGVR = schema.GroupVersionResource{
		Group:    corev1.GroupName,
		Version:  "v1",
		Resource: "secrets",
	}

	// ServiceGVR is the GVR for Service resources
	ServiceGVR = schema.GroupVersionResource{
		Group:    corev1.GroupName,
		Version:  "v1",
		Resource: "services",
	}

	// ConfigMapGVR is the GVR for ConfigMap resources
	ConfigMapGVR = schema.GroupVersionResource{
		Group:    corev1.GroupName,
		Version:  "v1",
		Resource: "configmaps",
	}

	// DeploymentGVR is the GVR for Deployment resources
	DeploymentGVR = schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}
)

func NewObject(gvr schema.GroupVersionResource, name, namespace string) operatorv1alpha1.BundleObject {
	return operatorv1alpha1.BundleObject{
		GVR:       gvr,
		Name:      name,
		Namespace: namespace,
	}
}

func NewSecret(name, namespace string) operatorv1alpha1.BundleObject {
	return NewObject(SecretGVR, name, namespace)
}

func NewService(name, namespace string) operatorv1alpha1.BundleObject {
	return NewObject(ServiceGVR, name, namespace)
}

func NewConfigMap(name, namespace string) operatorv1alpha1.BundleObject {
	return NewObject(ConfigMapGVR, name, namespace)
}

func NewDeployment(name, namespace string) operatorv1alpha1.BundleObject {
	return NewObject(DeploymentGVR, name, namespace)
}
