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

package virtualworkspace

import (
	"fmt"

	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/naming"
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

func getCacheServerKubeconfigMountPath() string {
	return "/etc/cache-server/kubeconfig"
}

// getCacheServerCAMountPath has to match the code in the cacheserver package.
func getCacheServerCAMountPath(caName operatorv1alpha1.CA) string {
	return fmt.Sprintf("/etc/cache-server/tls/ca/%s", caName)
}

// getCacheServerClientCertMountPath has to match the code in the cacheserver package.
func getCacheServerClientCertMountPath() string {
	return "/etc/cache-server/tls/client-certificate"
}

// getEffectiveCacheRef returns the cache server reference to use for this virtual workspace.
// It inherits from the target (shard or rootShard). If the target is a shard with no cache config,
// the rootShard's cache config is used.
func getEffectiveCacheRef(rootShard *operatorv1alpha1.RootShard, shard *operatorv1alpha1.Shard) string {
	if shard != nil && shard.Spec.Cache != nil && shard.Spec.Cache.Reference != nil {
		return shard.Spec.Cache.Reference.Name
	}
	if rootShard.Spec.Cache.Reference != nil {
		return rootShard.Spec.Cache.Reference.Name
	}
	return ""
}

func kubeconfigSecret(rootShard *operatorv1alpha1.RootShard, shard *operatorv1alpha1.Shard, certName operatorv1alpha1.Certificate, names naming.Scheme) string {
	if shard != nil {
		return names.ShardKubeconfigSecret(shard, certName)
	} else {
		return names.RootShardKubeconfigSecret(rootShard, certName)
	}
}

func DeploymentReconciler(vw *operatorv1alpha1.VirtualWorkspace, rootShard *operatorv1alpha1.RootShard, shard *operatorv1alpha1.Shard, names naming.Scheme) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return names.VirtualWorkspaceDeploymentName(vw), func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			labels := names.VirtualWorkspaceResourceLabels(vw)
			dep.SetLabels(labels)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: labels,
			}

			dep.Spec.Template.SetLabels(labels)

			secretMounts := []utils.SecretMount{{
				VolumeName: "kcp-ca",
				SecretName: names.RootShardCAName(rootShard, operatorv1alpha1.RootCA),
				MountPath:  getCAMountPath(operatorv1alpha1.RootCA),
			}}

			args := getArgs(vw, rootShard, shard)

			// All of these CAs are shared between rootshard and regular shards.
			for _, ca := range []operatorv1alpha1.CA{
				operatorv1alpha1.ClientCA,
				operatorv1alpha1.ServerCA,
				// operatorv1alpha1.ServiceAccountCA,
				operatorv1alpha1.RequestHeaderClientCA,
			} {
				secretMounts = append(secretMounts, utils.SecretMount{
					VolumeName: fmt.Sprintf("%s-ca", ca),
					SecretName: names.RootShardCAName(rootShard, ca),
					MountPath:  getCAMountPath(ca),
				})
			}

			secretMounts = append(secretMounts, utils.SecretMount{
				VolumeName: fmt.Sprintf("%s-cert", operatorv1alpha1.ServerCertificate),
				SecretName: names.VirtualWorkspaceCertificateName(vw, operatorv1alpha1.ServerCertificate),
				MountPath:  getCertificateMountPath(operatorv1alpha1.ServerCertificate),
			})

			// We use our own, custom client certificate to access our target (shard or root shard),
			// but mount it to the location expected by the logical-cluster-admin kubeconfig, which we
			// re-use (it's own by the shard/root shard) as a handy kubeconfig with the correct URL.

			secretMounts = append(secretMounts, utils.SecretMount{
				VolumeName: fmt.Sprintf("%s-kubeconfig", operatorv1alpha1.LogicalClusterAdminCertificate),
				SecretName: kubeconfigSecret(rootShard, shard, operatorv1alpha1.LogicalClusterAdminCertificate, names),
				MountPath:  getKubeconfigMountPath(operatorv1alpha1.LogicalClusterAdminCertificate),
			})

			secretMounts = append(secretMounts, utils.SecretMount{
				VolumeName: fmt.Sprintf("%s-cert", operatorv1alpha1.ClientCertificate),
				SecretName: names.VirtualWorkspaceCertificateName(vw, operatorv1alpha1.ClientCertificate),
				MountPath:  getCertificateMountPath(operatorv1alpha1.LogicalClusterAdminCertificate),
			})

			// If a cache server is configured (shard-specific or inherited from rootShard), mount its kubeconfig and the
			// certificates referenced in it.
			if cacheRef := getEffectiveCacheRef(rootShard, shard); cacheRef != "" {
				secretMounts = append(secretMounts,
					utils.SecretMount{
						VolumeName: "cache-server-kubeconfig",
						SecretName: names.CacheServerKubeconfigName(cacheRef),
						MountPath:  getCacheServerKubeconfigMountPath(),
					},
					utils.SecretMount{
						VolumeName: "cache-server-ca",
						SecretName: names.CacheServerCAName(cacheRef, operatorv1alpha1.RootCA),
						MountPath:  getCacheServerCAMountPath(operatorv1alpha1.RootCA),
					},
					utils.SecretMount{
						VolumeName: "cache-server-client-cert",
						SecretName: names.CacheServerClientCertificateName(cacheRef),
						MountPath:  getCacheServerClientCertMountPath(),
					},
				)
			}

			volumes := []corev1.Volume{}
			volumeMounts := []corev1.VolumeMount{}

			for _, sm := range secretMounts {
				v, vm := sm.Build()
				volumes = append(volumes, v)
				volumeMounts = append(volumeMounts, vm)
			}

			image, _, _ := resources.GetImageSettings(vw.Spec.Image)

			container := corev1.Container{
				Name:         ServerContainerName,
				Image:        image,
				Command:      []string{"/virtual-workspaces"},
				Args:         args,
				VolumeMounts: volumeMounts,
				Resources:    defaultResourceRequirements,
			}
			container = utils.ApplyResources(container, vw.Spec.Resources)

			dep.Spec.Template.Spec.Containers = []corev1.Container{container}
			dep.Spec.Template.Spec.Volumes = volumes

			if vw.Spec.Replicas != nil {
				dep.Spec.Replicas = vw.Spec.Replicas
			} else if dep.Spec.Replicas == nil {
				dep.Spec.Replicas = ptr.To[int32](2)
			}

			dep = utils.ApplyCommonShardDeploymentProperties(dep)
			dep = utils.ApplyDeploymentTemplate(dep, vw.Spec.DeploymentTemplate)

			dep.Spec.Template.Spec.Containers[0].ReadinessProbe = nil
			dep.Spec.Template.Spec.Containers[0].LivenessProbe = nil
			dep.Spec.Template.Spec.Containers[0].StartupProbe = nil

			// If shard has bundle annotation, store desired replicas in annotation then scale deployment to 0 locally
			if vw.Annotations != nil && vw.Annotations[resources.BundleAnnotation] != "" {
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

func getArgs(vw *operatorv1alpha1.VirtualWorkspace, rootShard *operatorv1alpha1.RootShard, shard *operatorv1alpha1.Shard) []string {
	// Configure the cache kubeconfig to point either to an explicitly configured cache (maybe on the
	// shard, maybe on the root shard), or the root shard itself (in case no external cache is configured).
	var cacheKubeconfigMount string
	if getEffectiveCacheRef(rootShard, shard) != "" {
		cacheKubeconfigMount = getCacheServerKubeconfigMountPath()
	}

	args := []string{
		// TLS setup
		fmt.Sprintf("--client-ca-file=%s/tls.crt", getCAMountPath(operatorv1alpha1.ClientCA)),
		fmt.Sprintf("--tls-private-key-file=%s/tls.key", getCertificateMountPath(operatorv1alpha1.ServerCertificate)),
		fmt.Sprintf("--tls-cert-file=%s/tls.crt", getCertificateMountPath(operatorv1alpha1.ServerCertificate)),

		// listening
		"--bind-address=0.0.0.0",
		"--secure-port=6443",

		// requestheader CA
		fmt.Sprintf("--requestheader-client-ca-file=%s/tls.crt", getCAMountPath(operatorv1alpha1.RequestHeaderClientCA)),
		fmt.Sprintf("--requestheader-allowed-names=%s,%s", resources.FrontProxyCommonName, resources.RootShardProxyCommonName),
		"--requestheader-username-headers=X-Remote-User",
		"--requestheader-group-headers=X-Remote-Group",
		"--requestheader-extra-headers-prefix=X-Remote-Extra-",

		// kubeconfig to connect to this VW's target
		fmt.Sprintf("--kubeconfig=%s/kubeconfig", getKubeconfigMountPath(operatorv1alpha1.LogicalClusterAdminCertificate)),

		// This flag was deprecated in #3849 (kcp 0.31+), but was required in all earlier versions.
		// Since it was never actually used by kcp, the easiest way to handle this is to just provide
		// a dummy URL for now until the kcp-operator stops supporting older kcp versions.
		"--shard-external-url=https://127.0.0.1:6443",
	}

	// If a cache server is configured, add the --cache-kubeconfig flag
	if cacheKubeconfigMount != "" {
		args = append(args, fmt.Sprintf("--cache-kubeconfig=%s/kubeconfig", cacheKubeconfigMount))
	}

	args = append(args, utils.GetLoggingArgs(vw.Spec.Logging)...)

	if vw.Spec.ExtraArgs != nil {
		args = append(args, vw.Spec.ExtraArgs...)
	}

	return args
}
