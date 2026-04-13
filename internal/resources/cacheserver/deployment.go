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

package cacheserver

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/utils"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

const (
	ServerContainerName = "cache-server"

	// embeddedEtcdStoragePath is the emptyDir path in the Deployment where the
	// embedded etcd data is temporarily stored.
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

func getEtcdCertificateMountPath() string {
	return "/etc/cache-server/etcd"
}

func getCAMountPath(caName operatorv1alpha1.CA) string {
	return fmt.Sprintf("/etc/cache-server/tls/ca/%s", caName)
}

func getClientCertificateMountPath() string {
	return "/etc/cache-server/tls/client-certificate"
}

func DeploymentReconciler(server *operatorv1alpha1.CacheServer) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return resources.GetCacheServerDeploymentName(server), func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			labels := resources.GetCacheServerResourceLabels(server)
			dep.SetLabels(labels)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: labels,
			}

			if server.Spec.Etcd == nil {
				if r := dep.Spec.Replicas; r != nil && *r > 1 {
					// Since there is only embedded etcd, this Deployment must not be scaled up.
					dep.Spec.Replicas = ptr.To(int32(1))
				}
			} else {
				// We are running with an external etcd. Default to 2 replicas.
				dep.Spec.Replicas = ptr.To(ptr.Deref(server.Spec.Replicas, 2))
			}

			dep.Spec.Template.SetLabels(labels)

			// TODO: Why do we discard the imagePullSecrets?
			image, _, version := resources.GetImageSettings(server.Spec.Image)

			volumes, volumeMounts := getVolumeMounts(server)

			for _, sm := range getSecretMounts(server, version) {
				v, vm := sm.Build()
				volumes = append(volumes, v)
				volumeMounts = append(volumeMounts, vm)
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{{
				Name:         ServerContainerName,
				Image:        image,
				Command:      []string{"/cache-server"},
				Args:         getArgs(server, version),
				VolumeMounts: volumeMounts,
				Resources:    defaultResourceRequirements,
				SecurityContext: &corev1.SecurityContext{
					SeccompProfile: &corev1.SeccompProfile{
						Type: corev1.SeccompProfileTypeRuntimeDefault,
					},
					ReadOnlyRootFilesystem:   ptr.To(true),
					AllowPrivilegeEscalation: ptr.To(false),
					Capabilities: &corev1.Capabilities{
						Drop: []corev1.Capability{
							corev1.Capability("ALL"),
						},
					},
				},
				Ports: []corev1.ContainerPort{
					{
						Name:          "https",
						ContainerPort: 6443,
						Protocol:      corev1.ProtocolTCP,
					},
				},
				ReadinessProbe: &corev1.Probe{
					FailureThreshold:    3,
					InitialDelaySeconds: 15,
					PeriodSeconds:       10,
					SuccessThreshold:    1,
					TimeoutSeconds:      10,
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path:   "/readyz",
							Port:   intstr.FromString("https"),
							Scheme: corev1.URISchemeHTTPS,
						},
					},
				},
				LivenessProbe: &corev1.Probe{
					FailureThreshold:    3,
					InitialDelaySeconds: 15,
					PeriodSeconds:       10,
					SuccessThreshold:    1,
					TimeoutSeconds:      10,
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path:   "/livez",
							Port:   intstr.FromString("https"),
							Scheme: corev1.URISchemeHTTPS,
						},
					},
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

func getVolumeMounts(server *operatorv1alpha1.CacheServer) (volumes []corev1.Volume, volumeMounts []corev1.VolumeMount) {
	const etcdScratchVolume = "etcd-scratch"

	if server.Spec.Etcd == nil {
		volumes = []corev1.Volume{{
			Name: etcdScratchVolume,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}}
		volumeMounts = []corev1.VolumeMount{{
			Name:      etcdScratchVolume,
			MountPath: embeddedEtcdStoragePath,
		}}
	}

	return
}

func getArgs(server *operatorv1alpha1.CacheServer, version *semver.Version) []string {
	args := []string{
		// Certificate flags (server, service account signing).
		fmt.Sprintf("--tls-cert-file=%s/tls.crt", getCertificateMountPath(operatorv1alpha1.ServerCertificate)),
		fmt.Sprintf("--tls-private-key-file=%s/tls.key", getCertificateMountPath(operatorv1alpha1.ServerCertificate)),
		// Configure (lack of) persistence.
		"--root-directory=",
	}

	if hasAuthenticatedCache(version) {
		// Client CA for authenticating clients connecting to the cache server.
		args = append(args, fmt.Sprintf("--client-ca-file=%s/tls.crt", getCAMountPath(operatorv1alpha1.RootCA)))
	}

	if server.Spec.Etcd == nil {
		// The CacheServer is configured with an embedded etcd store.
		args = append(args,
			fmt.Sprintf("--embedded-etcd-directory=%s", embeddedEtcdStoragePath),
		)
	} else {
		// The CacheServer is configured with a dedicated etcd store.
		args = append(args,
			fmt.Sprintf("--etcd-servers=%s", strings.Join(server.Spec.Etcd.Endpoints, ",")),
		)
		if server.Spec.Etcd.TLSConfig != nil {
			args = append(args,
				fmt.Sprintf("--etcd-cafile=%s/ca.crt", getEtcdCertificateMountPath()),
				fmt.Sprintf("--etcd-certfile=%s/tls.crt", getEtcdCertificateMountPath()),
				fmt.Sprintf("--etcd-keyfile=%s/tls.key", getEtcdCertificateMountPath()),
			)
		}
	}

	args = append(args, utils.GetLoggingArgs(server.Spec.Logging)...)

	return args
}

func getSecretMounts(server *operatorv1alpha1.CacheServer, version *semver.Version) []utils.SecretMount {
	secretMounts := []utils.SecretMount{
		{
			VolumeName: "serving-cert",
			SecretName: resources.GetCacheServerCertificateName(server, operatorv1alpha1.ServerCertificate),
			MountPath:  getCertificateMountPath(operatorv1alpha1.ServerCertificate),
		},
	}

	if hasAuthenticatedCache(version) {
		secretMounts = append(secretMounts, utils.SecretMount{
			VolumeName: "client-ca",
			SecretName: resources.GetCacheServerCAName(server.Name, operatorv1alpha1.RootCA),
			MountPath:  getCAMountPath(operatorv1alpha1.RootCA),
		})
	}

	if server.Spec.Etcd != nil && server.Spec.Etcd.TLSConfig != nil {
		secretMounts = append(secretMounts, utils.SecretMount{
			VolumeName: "etcd-cert",
			SecretName: server.Spec.Etcd.TLSConfig.SecretRef.Name,
			MountPath:  getEtcdCertificateMountPath(),
		})
	}

	return secretMounts
}

func hasAuthenticatedCache(version *semver.Version) bool {
	if version == nil {
		// If we can't parse the version, assume the best and include the client CA.
		return true
	}

	// Only include client-ca for kcp >= 0.29; 0.28 users will have to ensure their
	// tags parse as 0.28 to ensure compatibility with recent kcp-operators.
	constraint, _ := semver.NewConstraint("~0.29.2 || ~0.30.2 || >=0.31.0")

	return constraint.Check(version)
}
