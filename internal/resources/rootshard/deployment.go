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
	image, _ := resources.GetImageSettings(rootShard.Spec.Image)
	args := getArgs(rootShard)

	return func() (string, reconciling.DeploymentReconciler) {
		return fmt.Sprintf("%s-kcp", rootShard.Name), func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.SetLabels(rootShard.GetResourceLabels())
			dep.Spec.Selector = &v1.LabelSelector{
				MatchLabels: rootShard.GetResourceLabels(),
			}
			dep.Spec.Template.ObjectMeta.SetLabels(rootShard.GetResourceLabels())
			dep.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "kcp-ca",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: rootShard.GetCAName(v1alpha1.RootCA),
							Items: []corev1.KeyToPath{
								{
									Key:  "tls.crt",
									Path: "ca.crt",
								},
							},
						},
					},
				},
			}
			dep.Spec.Template.Spec.Containers = []corev1.Container{}

			container := corev1.Container{
				Name:    ServerContainerName,
				Image:   image,
				Command: []string{"/kcp", "start"},
				Args:    args,
				SecurityContext: &corev1.SecurityContext{
					ReadOnlyRootFilesystem:   ptr.To(true),
					AllowPrivilegeEscalation: ptr.To(false),
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "kcp-ca",
						MountPath: "/etc/kcp/tls/ca/root",
					},
				},
				Resources: defaultResourceRequirements,
			}

			if rootShard.Spec.Etcd.TLSConfig != nil {
				dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes, corev1.Volume{
					Name: "etcd-client-cert",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: rootShard.Spec.Etcd.TLSConfig.SecretRef.Name,
						},
					},
				})
				container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
					Name:      "etcd-client-cert",
					ReadOnly:  true,
					MountPath: "/etc/etcd/tls",
				})
			}

			for _, ca := range []v1alpha1.CA{
				v1alpha1.ClientCA,
				v1alpha1.ServiceAccountCA,
				v1alpha1.RequestHeaderClientCA,
			} {
				dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes,
					corev1.Volume{
						Name: fmt.Sprintf("%s-ca", ca),
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: rootShard.GetCAName(ca),
							},
						},
					})
				container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
					Name:      fmt.Sprintf("%s-ca", ca),
					ReadOnly:  true,
					MountPath: fmt.Sprintf("/etc/kcp/tls/ca/%s", ca),
				})
			}

			for _, cert := range []v1alpha1.Certificate{
				v1alpha1.ServerCertificate,
				v1alpha1.ServiceAccountCertificate,
			} {
				dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes, corev1.Volume{
					Name: string(cert),
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: rootShard.GetCertificateName(cert)}},
				})
				container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
					Name:      string(cert),
					ReadOnly:  true,
					MountPath: fmt.Sprintf("/etc/kcp/tls/%s", cert),
				})
			}

			dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, container)

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
		// CA configuration.
		fmt.Sprintf("--root-ca-file=/etc/kcp/tls/ca/%s/ca.crt", v1alpha1.RootCA),
		fmt.Sprintf("--client-ca-file=/etc/kcp/tls/ca/%s/tls.crt", v1alpha1.ClientCA),
		fmt.Sprintf("--requestheader-client-ca-file=/etc/kcp/tls/ca/%s/tls.crt", v1alpha1.RequestHeaderClientCA),

		// Certificate flags (server, service account signing).
		fmt.Sprintf("--tls-private-key-file=/etc/kcp/tls/%s/tls.key", v1alpha1.ServerCertificate),
		fmt.Sprintf("--tls-cert-file=/etc/kcp/tls/%s/tls.crt", v1alpha1.ServerCertificate),
		fmt.Sprintf("--service-account-key-file=/etc/kcp/tls/%s/tls.crt", v1alpha1.ServiceAccountCertificate),
		fmt.Sprintf("--service-account-private-key-file=/etc/kcp/tls/%s/tls.key", v1alpha1.ServiceAccountCertificate),

		// Etcd client configuration.
		fmt.Sprintf("--etcd-servers=%s", strings.Join(rootShard.Spec.Etcd.Endpoints, ",")),

		// General shard configuration.
		fmt.Sprintf("--shard-base-url=%s", rootShard.GetShardBaseURL()),
		fmt.Sprintf("--shard-external-url=https://%s:%d", rootShard.Spec.External.Hostname, rootShard.Spec.External.Port),
		"--root-directory=''",
	}

	if rootShard.Spec.Etcd.TLSConfig != nil {
		args = append(args, []string{" --etcd-certfile=/etc/etcd/tls/tls.crt",
			"--etcd-keyfile=/etc/etcd/tls/tls.key",
			"--etcd-cafile=/etc/etcd/tls/ca.crt",
		}...)
	}

	return args
}
