/*
Copyright 2026 The KCP Authors.

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

package cacheserver

import (
	"fmt"

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
	ServerContainerName = "cache-server"

	// embeddedEtcdStoragePath is the emptyDir path in the Deployment where the
	// etcd data is temporarily stored. kcp's cache-server as of v0.30 does not
	// support external etcd yet.
	embeddedEtcdStoragePath = "/var/etcd"
)

var (
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("1Gi"),
			corev1.ResourceCPU:    resource.MustParse("500m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("2Gi"),
			corev1.ResourceCPU:    resource.MustParse("1"),
		},
	}
)

// Make sure the following two do not use /etc/kcp/ as their basepath, or else
// we risk conflicts with the shard-related mount paths.

func getCertificateMountPath(certName operatorv1alpha1.Certificate) string {
	return fmt.Sprintf("/etc/cache-server/tls/%s", certName)
}

func getCAMountPath(caName operatorv1alpha1.CA) string {
	return fmt.Sprintf("/etc/cache-server/tls/ca/%s", caName)
}

func DeploymentReconciler(server *operatorv1alpha1.CacheServer) reconciling.NamedDeploymentReconcilerFactory {
	const etcdScratchVolume = "etcd-scratch"

	return func() (string, reconciling.DeploymentReconciler) {
		return resources.GetCacheServerDeploymentName(server), func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			labels := resources.GetCacheServerResourceLabels(server)
			dep.SetLabels(labels)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: labels,
			}

			if r := dep.Spec.Replicas; r != nil && *r > 1 {
				// Since there is only embedded etcd, this Deployment must not be scaled up.
				dep.Spec.Replicas = ptr.To(int32(1))
			}

			dep.Spec.Template.SetLabels(labels)

			secretMounts := []utils.SecretMount{{
				VolumeName: "serving-cert",
				SecretName: resources.GetCacheServerCertificateName(server, operatorv1alpha1.ServerCertificate),
				MountPath:  getCertificateMountPath(operatorv1alpha1.ServerCertificate),
			}}

			// TODO: Why do we discard the imagePullSecrets?
			image, _ := resources.GetImageSettings(server.Spec.Image)

			args := getArgs(server)
			volumes := []corev1.Volume{{
				Name: etcdScratchVolume,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			}}
			volumeMounts := []corev1.VolumeMount{{
				Name:      etcdScratchVolume,
				MountPath: embeddedEtcdStoragePath,
			}}

			for _, sm := range secretMounts {
				v, vm := sm.Build()
				volumes = append(volumes, v)
				volumeMounts = append(volumeMounts, vm)
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{{
				Name:         ServerContainerName,
				Image:        image,
				Command:      []string{"/cache-server"},
				Args:         args,
				VolumeMounts: volumeMounts,
				Resources:    defaultResourceRequirements,
				SecurityContext: &corev1.SecurityContext{
					ReadOnlyRootFilesystem:   ptr.To(true),
					AllowPrivilegeEscalation: ptr.To(false),
				},
			}}
			dep.Spec.Template.Spec.Volumes = volumes

			dep = utils.ApplyDeploymentTemplate(dep, server.Spec.DeploymentTemplate)

			// If the cacheserver has bundle annotation, store desired replicas in annotation then scale deployment to 0 locally
			if server.Annotations != nil && server.Annotations[resources.BundleAnnotation] != "" {
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

func getArgs(server *operatorv1alpha1.CacheServer) []string {
	args := []string{
		// Configure (lack of) persistence.
		"--root-directory=",
		fmt.Sprintf("--embedded-etcd-directory=%s", embeddedEtcdStoragePath),

		// Certificate flags (server, service account signing).
		fmt.Sprintf("--tls-cert-file=%s/tls.crt", getCertificateMountPath(operatorv1alpha1.ServerCertificate)),
		fmt.Sprintf("--tls-private-key-file=%s/tls.key", getCertificateMountPath(operatorv1alpha1.ServerCertificate)),
	}

	args = append(args, utils.GetLoggingArgs(server.Spec.Logging)...)

	return args
}
