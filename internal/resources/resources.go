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

package resources

import (
	"fmt"

	"github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"

	corev1 "k8s.io/api/core/v1"
)

const (
	ImageRepository = "ghcr.io/kcp-dev/kcp"
	ImageTag        = "v0.26.0"

	appNameLabel      = "app.kubernetes.io/name"
	appInstanceLabel  = "app.kubernetes.io/instance"
	appManagedByLabel = "app.kubernetes.io/managed-by"
	appComponentLabel = "app.kubernetes.io/component"
)

func GetImageSettings(imageSpec *v1alpha1.ImageSpec) (string, []corev1.LocalObjectReference) {
	repository := ImageRepository
	if imageSpec != nil && imageSpec.Repository != "" {
		repository = imageSpec.Repository
	}

	tag := ImageTag
	if imageSpec != nil && imageSpec.Tag != "" {
		tag = imageSpec.Tag
	}

	imagePullSecrets := []corev1.LocalObjectReference{}
	if imageSpec != nil && len(imageSpec.ImagePullSecrets) > 0 {
		imagePullSecrets = imageSpec.ImagePullSecrets
	}

	return fmt.Sprintf("%s:%s", repository, tag), imagePullSecrets
}

func GetRootShardDeploymentName(r *v1alpha1.RootShard) string {
	return fmt.Sprintf("%s-kcp", r.Name)
}

func GetRootShardServiceName(r *v1alpha1.RootShard) string {
	return fmt.Sprintf("%s-kcp", r.Name)
}

func GetRootShardResourceLabels(r *v1alpha1.RootShard) map[string]string {
	return map[string]string{
		appNameLabel:      "kcp",
		appInstanceLabel:  r.Name,
		appManagedByLabel: "kcp-operator",
		appComponentLabel: "rootshard",
	}
}

func GetRootShardBaseURL(r *v1alpha1.RootShard) string {
	clusterDomain := r.Spec.ClusterDomain
	if clusterDomain == "" {
		clusterDomain = "cluster.local"
	}

	return fmt.Sprintf("https://%s-kcp.%s.svc.%s:6443", r.Name, r.Namespace, clusterDomain)
}

func GetRootShardCertificateName(r *v1alpha1.RootShard, certName v1alpha1.Certificate) string {
	return fmt.Sprintf("%s-%s", r.Name, certName)
}

func GetRootShardCAName(r *v1alpha1.RootShard, caName v1alpha1.CA) string {
	if caName == v1alpha1.RootCA {
		return fmt.Sprintf("%s-ca", r.Name)
	}
	return fmt.Sprintf("%s-%s-ca", r.Name, caName)
}
