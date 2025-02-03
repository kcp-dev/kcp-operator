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

package rootshard

import (
	"fmt"
	"strings"

	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/utils"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
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

func getCertificateMountPath(certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("/etc/kcp/tls/%s", certName)
}

func getCAMountPath(caName operatorv1alpha1.CA) string {
	return fmt.Sprintf("/etc/kcp/tls/ca/%s", caName)
}

func getKubeconfigMountPath(certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("/etc/kcp/%s-kubeconfig", certName)
}

func DeploymentReconciler(rootShard *operatorv1alpha1.RootShard) reconciling.NamedDeploymentReconcilerFactory {

	return func() (string, reconciling.DeploymentReconciler) {
		return resources.GetRootShardDeploymentName(rootShard), func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			labels := resources.GetRootShardResourceLabels(rootShard)
			dep.SetLabels(labels)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: labels,
			}
			dep.Spec.Template.ObjectMeta.SetLabels(labels)

			secretMounts := []utils.SecretMount{{
				VolumeName: "kcp-ca",
				SecretName: resources.GetRootShardCAName(rootShard, operatorv1alpha1.RootCA),
				MountPath:  getCAMountPath(operatorv1alpha1.RootCA),
			}}

			image, _ := resources.GetImageSettings(rootShard.Spec.Image)
			args := getArgs(rootShard)

			if rootShard.Spec.Etcd.TLSConfig != nil {
				secretMounts = append(secretMounts, utils.SecretMount{
					VolumeName: "etcd-client-cert",
					SecretName: rootShard.Spec.Etcd.TLSConfig.SecretRef.Name,
					MountPath:  "/etc/etcd/tls",
				})

				args = append(args,
					"--etcd-certfile=/etc/etcd/tls/tls.crt",
					"--etcd-keyfile=/etc/etcd/tls/tls.key",
					"--etcd-cafile=/etc/etcd/tls/ca.crt",
				)
			}

			for _, cert := range []operatorv1alpha1.Certificate{
				// requires server CA and the logical-cluster-admin cert to be mounted
				operatorv1alpha1.LogicalClusterAdminCertificate,
				// requires server CA and the external-logical-cluster-admin cert to be mounted
				operatorv1alpha1.ExternalLogicalClusterAdminCertificate,
			} {
				secretMounts = append(secretMounts, utils.SecretMount{
					VolumeName: fmt.Sprintf("%s-kubeconfig", cert),
					SecretName: kubeconfigSecret(rootShard, cert),
					MountPath:  getKubeconfigMountPath(cert),
				})
			}

			for _, ca := range []operatorv1alpha1.CA{
				operatorv1alpha1.ClientCA,
				operatorv1alpha1.ServerCA,
				operatorv1alpha1.ServiceAccountCA,
				operatorv1alpha1.RequestHeaderClientCA,
			} {
				secretMounts = append(secretMounts, utils.SecretMount{
					VolumeName: fmt.Sprintf("%s-ca", ca),
					SecretName: resources.GetRootShardCAName(rootShard, ca),
					MountPath:  getCAMountPath(ca),
				})
			}

			for _, cert := range []operatorv1alpha1.Certificate{
				operatorv1alpha1.ServerCertificate,
				operatorv1alpha1.ServiceAccountCertificate,
				operatorv1alpha1.LogicalClusterAdminCertificate,
				operatorv1alpha1.ExternalLogicalClusterAdminCertificate,
			} {
				secretMounts = append(secretMounts, utils.SecretMount{
					VolumeName: fmt.Sprintf("%s-cert", cert),
					SecretName: resources.GetRootShardCertificateName(rootShard, cert),
					MountPath:  getCertificateMountPath(cert),
				})
			}

			volumes := []corev1.Volume{}
			volumeMounts := []corev1.VolumeMount{}

			for _, sm := range secretMounts {
				v, vm := sm.Build()
				volumes = append(volumes, v)
				volumeMounts = append(volumeMounts, vm)
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{{
				Name:         ServerContainerName,
				Image:        image,
				Command:      []string{"/kcp", "start"},
				Args:         args,
				VolumeMounts: volumeMounts,
				Resources:    defaultResourceRequirements,
				SecurityContext: &corev1.SecurityContext{
					ReadOnlyRootFilesystem:   ptr.To(true),
					AllowPrivilegeEscalation: ptr.To(false),
				},
			}}
			dep.Spec.Template.Spec.Volumes = volumes

			// explicitly set the replicas if it is configured in the RootShard
			// object or if the existing Deployment object doesn't have replicas
			// configured. This will allow a HPA to interact with the replica
			// count.
			if rootShard.Spec.Replicas != nil {
				dep.Spec.Replicas = rootShard.Spec.Replicas
			} else if dep.Spec.Replicas == nil {
				dep.Spec.Replicas = ptr.To[int32](2)
			}

			dep, err := utils.ApplyAuditConfiguration(dep, rootShard.Spec.Audit)
			if err != nil {
				return nil, fmt.Errorf("failed to apply audit configuration: %w", err)
			}

			return dep, nil
		}
	}
}

func getArgs(rootShard *operatorv1alpha1.RootShard) []string {
	args := []string{
		// CA configuration.
		fmt.Sprintf("--root-ca-file=%s/tls.crt", getCAMountPath(operatorv1alpha1.RootCA)),
		fmt.Sprintf("--client-ca-file=%s/tls.crt", getCAMountPath(operatorv1alpha1.ClientCA)),

		// Requestheader configuration.
		fmt.Sprintf("--requestheader-client-ca-file=%s/tls.crt", getCAMountPath(operatorv1alpha1.RequestHeaderClientCA)),
		"--requestheader-username-headers=X-Remote-User",
		"--requestheader-group-headers=X-Remote-Group",
		"--requestheader-extra-headers-prefix=X-Remote-Extra-",

		// Certificate flags (server, service account signing).
		fmt.Sprintf("--tls-private-key-file=%s/tls.key", getCertificateMountPath(operatorv1alpha1.ServerCertificate)),
		fmt.Sprintf("--tls-cert-file=%s/tls.crt", getCertificateMountPath(operatorv1alpha1.ServerCertificate)),
		fmt.Sprintf("--service-account-key-file=%s/tls.crt", getCertificateMountPath(operatorv1alpha1.ServiceAccountCertificate)),
		fmt.Sprintf("--service-account-private-key-file=%s/tls.key", getCertificateMountPath(operatorv1alpha1.ServiceAccountCertificate)),

		// Etcd client configuration.
		fmt.Sprintf("--etcd-servers=%s", strings.Join(rootShard.Spec.Etcd.Endpoints, ",")),

		// General shard configuration.
		fmt.Sprintf("--shard-base-url=%s", resources.GetRootShardBaseURL(rootShard)),
		fmt.Sprintf("--shard-external-url=https://%s:%d", rootShard.Spec.External.Hostname, rootShard.Spec.External.Port),
		fmt.Sprintf("--logical-cluster-admin-kubeconfig=%s/kubeconfig", getKubeconfigMountPath(operatorv1alpha1.LogicalClusterAdminCertificate)),
		fmt.Sprintf("--external-logical-cluster-admin-kubeconfig=%s/kubeconfig", getKubeconfigMountPath(operatorv1alpha1.ExternalLogicalClusterAdminCertificate)),
		"--root-directory=",
		"--enable-leader-election=true",
		"--logging-format=json",
	}

	return args
}
