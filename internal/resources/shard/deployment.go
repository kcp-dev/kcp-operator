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

package shard

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

func DeploymentReconciler(shard *operatorv1alpha1.Shard, rootShard *operatorv1alpha1.RootShard) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return resources.GetShardDeploymentName(shard), func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			labels := resources.GetShardResourceLabels(shard)
			dep.SetLabels(labels)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: labels,
			}

			dep.Spec.Template.SetLabels(labels)

			secretMounts := []utils.SecretMount{{
				VolumeName: "kcp-ca",
				SecretName: resources.GetRootShardCAName(rootShard, operatorv1alpha1.RootCA),
				MountPath:  getCAMountPath(operatorv1alpha1.RootCA),
			}}

			args := getArgs(shard, rootShard)

			for _, cert := range []operatorv1alpha1.Certificate{
				// requires server CA and the shard client cert to be mounted
				operatorv1alpha1.ClientCertificate,
				operatorv1alpha1.LogicalClusterAdminCertificate,
				operatorv1alpha1.ExternalLogicalClusterAdminCertificate,
			} {
				secretMounts = append(secretMounts, utils.SecretMount{
					VolumeName: fmt.Sprintf("%s-kubeconfig", cert),
					SecretName: kubeconfigSecret(shard, cert),
					MountPath:  getKubeconfigMountPath(cert),
				})
			}

			// All of these CAs are shared between rootshard and regular shards.
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
				operatorv1alpha1.ClientCertificate,
				operatorv1alpha1.LogicalClusterAdminCertificate,
				operatorv1alpha1.ExternalLogicalClusterAdminCertificate,
			} {
				secretMounts = append(secretMounts, utils.SecretMount{
					VolumeName: fmt.Sprintf("%s-cert", cert),
					SecretName: resources.GetShardCertificateName(shard, cert),
					MountPath:  getCertificateMountPath(cert),
				})
			}

			// If CABundle is specified, mount the merged CA bundle secret.
			// This secret contains both ServerCA and user-provided CA bundle merged together.
			// It will not be used for the API server itself, but only for the "external-logical-cluster-admin-kubeconfig" kubeconfig.
			// See the comment in the RootShard spec for more details.
			if shard.Spec.CABundleSecretRef != nil {
				secretMounts = append(secretMounts, utils.SecretMount{
					VolumeName: "ca-bundle",
					SecretName: fmt.Sprintf("%s-merged-ca-bundle", shard.Name),
					MountPath:  getCAMountPath(operatorv1alpha1.CABundleCA),
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
				Command:      []string{"/kcp", "start"},
				Args:         args,
				VolumeMounts: volumeMounts,
				Resources:    defaultResourceRequirements,
			}}
			dep.Spec.Template.Spec.Volumes = volumes

			dep = utils.ApplyCommonShardDeploymentProperties(dep)
			dep = utils.ApplyCommonShardConfig(dep, &shard.Spec.CommonShardSpec)
			dep = utils.ApplyDeploymentTemplate(dep, shard.Spec.DeploymentTemplate)
			dep = utils.ApplyAuthConfiguration(dep, shard.Spec.Auth)

			// If shard has bundle annotation, store desired replicas in annotation then scale deployment to 0 locally
			if shard.Annotations != nil && shard.Annotations[resources.BundleAnnotation] != "" {
				// Store the desired replicas in an annotation so bundle can capture the correct value
				if dep.Spec.Replicas != nil && *dep.Spec.Replicas > 0 {
					if dep.Annotations == nil {
						dep.Annotations = make(map[string]string)
					}
					dep.Annotations[resources.BundleDesiredReplicasAnnotation] = fmt.Sprintf("%d", *dep.Spec.Replicas)
				}
				// Scale to 0 locally
				dep.Spec.Replicas = ptr.To(int32(0))
			}

			return dep, nil
		}
	}
}

func getArgs(shard *operatorv1alpha1.Shard, rootShard *operatorv1alpha1.RootShard) []string {
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
		"--service-account-lookup=false",

		fmt.Sprintf("--shard-client-key-file=%s/tls.crt", getCertificateMountPath(operatorv1alpha1.ClientCertificate)),
		fmt.Sprintf("--shard-client-cert-file=%s/tls.key", getCertificateMountPath(operatorv1alpha1.ClientCertificate)),

		// General shard configuration.
		fmt.Sprintf("--shard-name=%s", shard.Name),
		fmt.Sprintf("--shard-base-url=%s", resources.GetShardBaseURL(shard)),
		fmt.Sprintf("--shard-external-url=https://%s:%d", rootShard.Spec.External.Hostname, rootShard.Spec.External.Port),
		fmt.Sprintf("--external-hostname=%s", rootShard.Spec.External.Hostname),

		fmt.Sprintf("--root-shard-kubeconfig-file=%s/kubeconfig", getKubeconfigMountPath(operatorv1alpha1.ClientCertificate)),
		fmt.Sprintf("--cache-kubeconfig=%s/kubeconfig", getKubeconfigMountPath(operatorv1alpha1.ClientCertificate)),
		fmt.Sprintf("--logical-cluster-admin-kubeconfig=%s/kubeconfig", getKubeconfigMountPath(operatorv1alpha1.LogicalClusterAdminCertificate)),
		fmt.Sprintf("--external-logical-cluster-admin-kubeconfig=%s/kubeconfig", getKubeconfigMountPath(operatorv1alpha1.ExternalLogicalClusterAdminCertificate)),

		fmt.Sprintf("--batteries-included=%s", strings.Join(utils.GetShardBatteries(shard), ",")),

		"--root-directory=",
		"--enable-leader-election=true",
		"--logging-format=json",
	}

	args = append(args, utils.GetLoggingArgs(shard.Spec.Logging)...)

	if shard.Spec.ExtraArgs != nil {
		args = append(args, shard.Spec.ExtraArgs...)
	}

	return args
}
