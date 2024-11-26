/*
Copyright 2024 The KCP Authors.

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

package rootshard

import (
	"fmt"
	"strings"

	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/kcp-dev/kcp-operator/api/v1alpha1"
	"github.com/kcp-dev/kcp-operator/internal/resources"
)

const (
	ServerContainerName = "kcp"
)

var (
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("1Gi"),
			corev1.ResourceCPU:    resource.MustParse("1"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("2Gi"),
			corev1.ResourceCPU:    resource.MustParse("2"),
		},
	}
)

func DeploymentReconciler(rootShard *v1alpha1.RootShard) reconciling.NamedDeploymentReconcilerFactory {
	image, imagePullSecrets := resources.GetImageSettings(rootShard.Spec.Image)
	args := getArgs(rootShard)

	return func() (string, reconciling.DeploymentReconciler) {
		return rootShard.Name, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Spec = appsv1.DeploymentSpec{
				Selector: &v1.LabelSelector{
					MatchLabels: rootShard.GetResourceLabels(),
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: v1.ObjectMeta{
						Labels: rootShard.GetResourceLabels(),
					},
					Spec: corev1.PodSpec{
						ImagePullSecrets: imagePullSecrets,
						Containers: []corev1.Container{
							{
								Name:    ServerContainerName,
								Image:   image,
								Command: []string{"/kcp", "start"},
								Args:    args,
							},
						},
					},
				},
			}

			// explicitly set the replicas if it is configured in the RootShard
			// object or if the existing Deployment object doesn't have replicas
			// configured. This will allow a HPA to interact with the replica
			// count.
			if rootShard.Spec.Replicas != nil {
				dep.Spec.Replicas = rootShard.Spec.Replicas
			} else if dep.Spec.Replicas == nil {
				dep.Spec.Replicas = ptr.To[int32](2)
			}

			return dep, nil
		}
	}
}

func getArgs(rootShard *v1alpha1.RootShard) []string {
	args := []string{
		fmt.Sprintf("--etcd-servers=%s", strings.Join(rootShard.Spec.Etcd.Endpoints, ",")),

		"--shard-base-url=''",
		fmt.Sprintf("--shard-external-url=https://%s:6443", rootShard.Spec.Hostname),
		"--root-directory=''",
	}

	return args
}
