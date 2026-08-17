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

package compiledvirtualworkspace

import (
	"fmt"
	"slices"

	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/utils"
	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

const (
	ServerContainerName = "kcp"

	// DefaultServerCommand is the entrypoint of kcp's own virtual-workspace server image. Images
	// shipping a different binary have to set spec.command.
	DefaultServerCommand = "/virtual-workspaces"
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

// getServerKubeconfigMountPath is where a user-provided kubeconfig for the server container is
// mounted. It is deliberately not the logical-cluster-admin path, so both kubeconfigs can be
// present in the pod at once and each container picks the one it needs.
func getServerKubeconfigMountPath() string {
	return "/etc/kcp/server-kubeconfig"
}

// getInitKubeconfigMountPath is where an init container's own kubeconfig is mounted. Volume mounts
// are per-container, so every init container sees its own kubeconfig at the same path.
func getInitKubeconfigMountPath() string {
	return "/etc/kcp/init-kubeconfig"
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
func getEffectiveCacheRef(vw *deployv1alpha1.CompiledVirtualWorkspace) string {
	if shard := vw.Spec.Shard; shard != nil && shard.Spec.Cache != nil && shard.Spec.Cache.Reference != nil {
		return shard.Spec.Cache.Reference.Name
	}
	if ref := vw.Spec.RootShard.Spec.Cache.Reference; ref != nil {
		return ref.Name
	}
	return ""
}

func kubeconfigSecret(vw *deployv1alpha1.CompiledVirtualWorkspace, certName operatorv1alpha1.Certificate) string {
	if shard := vw.Spec.Shard; shard != nil {
		return resources.GetNamedShardKubeconfigSecret(*shard, certName)
	}
	return resources.GetNamedRootShardKubeconfigSecret(vw.Spec.RootShard, certName)
}

func DeploymentReconciler(vw *deployv1alpha1.CompiledVirtualWorkspace) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return resources.GetCompiledVirtualWorkspaceDeploymentName(vw), func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			labels := resources.GetCompiledVirtualWorkspaceResourceLabels(vw)
			dep.SetLabels(labels)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: labels,
			}

			dep.Spec.Template.SetLabels(labels)

			secretMounts := []utils.SecretMount{{
				VolumeName: "kcp-ca",
				SecretName: resources.GetNamedRootShardCAName(vw.Spec.RootShard, operatorv1alpha1.RootCA),
				MountPath:  getCAMountPath(operatorv1alpha1.RootCA),
			}}

			args := getArgs(vw)

			// CAs shared between rootshard and regular shards (excluding ClientCA which may need merging)
			for _, ca := range []operatorv1alpha1.CA{
				operatorv1alpha1.ServerCA,
				operatorv1alpha1.RequestHeaderClientCA,
			} {
				secretMounts = append(secretMounts, utils.SecretMount{
					VolumeName: fmt.Sprintf("%s-ca", ca),
					SecretName: resources.GetNamedRootShardCAName(vw.Spec.RootShard, ca),
					MountPath:  getCAMountPath(ca),
				})
			}

			// ClientCA: use merged secret if any ClientCABundleRef is set (inherited from RootShard or VW's own)
			if vw.Spec.RootShard.Spec.ClientCABundleRef != nil || vw.Spec.VirtualWorkspace.ClientCABundleRef != nil {
				secretMounts = append(secretMounts, utils.SecretMount{
					VolumeName: fmt.Sprintf("%s-ca", operatorv1alpha1.ClientCA),
					SecretName: fmt.Sprintf("%s-merged-client-ca", vw.Name),
					MountPath:  getCAMountPath(operatorv1alpha1.ClientCA),
				})
			} else {
				secretMounts = append(secretMounts, utils.SecretMount{
					VolumeName: fmt.Sprintf("%s-ca", operatorv1alpha1.ClientCA),
					SecretName: resources.GetNamedRootShardCAName(vw.Spec.RootShard, operatorv1alpha1.ClientCA),
					MountPath:  getCAMountPath(operatorv1alpha1.ClientCA),
				})
			}

			secretMounts = append(secretMounts, utils.SecretMount{
				VolumeName: fmt.Sprintf("%s-cert", operatorv1alpha1.ServerCertificate),
				SecretName: resources.GetCompiledVirtualWorkspaceCertificateName(vw, operatorv1alpha1.ServerCertificate),
				MountPath:  getCertificateMountPath(operatorv1alpha1.ServerCertificate),
			})

			// We use our own, custom client certificate to access our target (shard or root shard),
			// but mount it to the location expected by the logical-cluster-admin kubeconfig, which we
			// re-use (it's own by the shard/root shard) as a handy kubeconfig with the correct URL.
			//
			// This pair is the privileged fallback credential, so it is kept out of any container
			// that brought its own kubeconfig: leaving an admin key in a container that was
			// deliberately given a narrower identity would defeat the point.
			adminMounts := []utils.SecretMount{
				{
					VolumeName: fmt.Sprintf("%s-kubeconfig", operatorv1alpha1.LogicalClusterAdminCertificate),
					SecretName: kubeconfigSecret(vw, operatorv1alpha1.LogicalClusterAdminCertificate),
					MountPath:  getKubeconfigMountPath(operatorv1alpha1.LogicalClusterAdminCertificate),
				},
				{
					VolumeName: fmt.Sprintf("%s-cert", operatorv1alpha1.ClientCertificate),
					SecretName: resources.GetCompiledVirtualWorkspaceCertificateName(vw, operatorv1alpha1.ClientCertificate),
					MountPath:  getCertificateMountPath(operatorv1alpha1.LogicalClusterAdminCertificate),
				},
			}

			// If a cache server is configured (shard-specific or inherited from rootShard), mount its kubeconfig and the
			// certificates referenced in it.
			if cacheRef := getEffectiveCacheRef(vw); cacheRef != "" {
				secretMounts = append(secretMounts,
					utils.SecretMount{
						VolumeName: "cache-server-kubeconfig",
						SecretName: resources.GetCacheServerKubeconfigName(cacheRef),
						MountPath:  getCacheServerKubeconfigMountPath(),
					},
					utils.SecretMount{
						VolumeName: "cache-server-ca",
						SecretName: resources.GetCacheServerCAName(cacheRef, operatorv1alpha1.RootCA),
						MountPath:  getCacheServerCAMountPath(operatorv1alpha1.RootCA),
					},
					utils.SecretMount{
						VolumeName: "cache-server-client-cert",
						SecretName: fmt.Sprintf("%s-client-certificate", cacheRef),
						MountPath:  getCacheServerClientCertMountPath(),
					},
				)
			}

			// Volumes are pod-scoped and exist for every mount, whether or not a given container
			// uses it; only the per-container mount lists differ. sharedMounts holds just the
			// operator-managed certificates and kubeconfigs, which every container gets so that
			// bootstrapping binaries reach them without knowing the operator's mount paths.
			// User-supplied extra mounts are deliberately not shared -- the server container gets
			// the VirtualWorkspace's, each init container gets its own.
			volumes := slices.Clone(vw.Spec.VirtualWorkspace.ExtraVolumes)
			sharedMounts := []corev1.VolumeMount{}

			for _, sm := range secretMounts {
				v, vm := sm.Build()
				volumes = append(volumes, v)
				sharedMounts = append(sharedMounts, vm)
			}

			adminVolumeMounts := []corev1.VolumeMount{}
			for _, sm := range adminMounts {
				v, vm := sm.Build()
				volumes = append(volumes, v)
				adminVolumeMounts = append(adminVolumeMounts, vm)
			}

			// containerMounts assembles the mounts for one container: everything shared, then
			// either its own kubeconfig or, when it has none, the logical-cluster-admin fallback,
			// and finally that container's own extra mounts.
			containerMounts := func(ref *corev1.LocalObjectReference, volumeName, mountPath string, extra []corev1.VolumeMount) []corev1.VolumeMount {
				mounts := slices.Clone(sharedMounts)
				if ref == nil {
					mounts = append(mounts, adminVolumeMounts...)
				} else {
					volume, mount := utils.SecretMount{
						VolumeName: volumeName,
						SecretName: ref.Name,
						MountPath:  mountPath,
					}.Build()

					volumes = append(volumes, volume)
					mounts = append(mounts, mount)
				}

				return append(mounts, extra...)
			}

			image, imagePullSecrets, _ := resources.GetImageSettings(vw.Spec.VirtualWorkspace.Image)

			command := []string{DefaultServerCommand}
			if len(vw.Spec.VirtualWorkspace.Command) > 0 {
				command = vw.Spec.VirtualWorkspace.Command
			}

			container := corev1.Container{
				Name:    ServerContainerName,
				Image:   image,
				Command: command,
				Args:    args,
				VolumeMounts: containerMounts(
					vw.Spec.VirtualWorkspace.KubeconfigSecretRef,
					"server-kubeconfig",
					getServerKubeconfigMountPath(),
					vw.Spec.VirtualWorkspace.ExtraVolumeMounts,
				),
				Resources: defaultResourceRequirements,
			}
			container = utils.ApplyResources(container, vw.Spec.VirtualWorkspace.Resources)

			initContainers, initVolumes, initPullSecrets := getInitContainers(vw, image, containerMounts)

			dep.Spec.Template.Spec.Containers = []corev1.Container{container}
			dep.Spec.Template.Spec.InitContainers = initContainers
			dep.Spec.Template.Spec.Volumes = dedupeVolumes(append(volumes, initVolumes...))
			dep.Spec.Template.Spec.ImagePullSecrets = append(imagePullSecrets, initPullSecrets...)

			if vw.Spec.VirtualWorkspace.Replicas != nil {
				dep.Spec.Replicas = vw.Spec.VirtualWorkspace.Replicas
			} else if dep.Spec.Replicas == nil {
				dep.Spec.Replicas = ptr.To[int32](2)
			}

			dep = utils.ApplyCommonShardDeploymentProperties(dep)
			dep = utils.ApplyDeploymentTemplate(dep, vw.Spec.VirtualWorkspace.DeploymentTemplate)

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

// getInitContainers builds the init containers for a virtual workspace. They inherit the shared
// volume mounts, so bootstrapping binaries reach the same certificates without the user having to
// know the operator's mount paths, and each is given its own kubeconfig at a fixed path -- volume
// mounts are per-container, so one path serves them all. Image pull secrets referenced by init
// containers that bring their own image are returned to be added to the pod.
func getInitContainers(
	vw *deployv1alpha1.CompiledVirtualWorkspace,
	serverImage string,
	containerMounts func(ref *corev1.LocalObjectReference, volumeName, mountPath string, extra []corev1.VolumeMount) []corev1.VolumeMount,
) ([]corev1.Container, []corev1.Volume, []corev1.LocalObjectReference) {
	specs := vw.Spec.VirtualWorkspace.InitContainers
	if len(specs) == 0 {
		return nil, nil, nil
	}

	containers := make([]corev1.Container, 0, len(specs))
	var volumes []corev1.Volume
	var pullSecrets []corev1.LocalObjectReference

	for _, spec := range specs {
		image := serverImage
		if spec.Image != nil {
			var secrets []corev1.LocalObjectReference
			image, secrets, _ = resources.GetImageSettings(spec.Image)
			pullSecrets = append(pullSecrets, secrets...)
		}

		// Volumes are pod-scoped, so an init container's volumes join the pod's list, but only
		// this container mounts them.
		volumes = append(volumes, spec.Volumes...)

		container := corev1.Container{
			Name:    spec.Name,
			Image:   image,
			Command: spec.Command,
			Args:    spec.Args,
			VolumeMounts: containerMounts(
				spec.KubeconfigSecretRef,
				fmt.Sprintf("init-%s-kubeconfig", spec.Name),
				getInitKubeconfigMountPath(),
				spec.VolumeMounts,
			),
			Resources: defaultResourceRequirements,
		}

		containers = append(containers, utils.ApplyResources(container, spec.Resources))
	}

	return containers, volumes, pullSecrets
}

// dedupeVolumes drops volumes whose name was already seen, keeping the first declaration. Volume
// names are pod-scoped while extraVolumes can be declared on the VirtualWorkspace and on every
// init container, so the same name legitimately shows up more than once when several containers
// want the same scratch volume. Kubernetes rejects a Pod that lists a volume name twice.
func dedupeVolumes(volumes []corev1.Volume) []corev1.Volume {
	seen := make(map[string]struct{}, len(volumes))
	deduped := make([]corev1.Volume, 0, len(volumes))

	for _, volume := range volumes {
		if _, ok := seen[volume.Name]; ok {
			continue
		}

		seen[volume.Name] = struct{}{}
		deduped = append(deduped, volume)
	}

	return deduped
}

// serverKubeconfigMountPath returns the directory holding the kubeconfig the server container
// authenticates with: the user-provided one if there is any, otherwise the logical-cluster-admin
// kubeconfig borrowed from the target.
func serverKubeconfigMountPath(vw *deployv1alpha1.CompiledVirtualWorkspace) string {
	if vw.Spec.VirtualWorkspace.KubeconfigSecretRef != nil {
		return getServerKubeconfigMountPath()
	}

	return getKubeconfigMountPath(operatorv1alpha1.LogicalClusterAdminCertificate)
}

func getArgs(vw *deployv1alpha1.CompiledVirtualWorkspace) []string {
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
		fmt.Sprintf("--kubeconfig=%s/kubeconfig", serverKubeconfigMountPath(vw)),
	}

	// --shard-external-url is deprecated in v0.31 but before that a required but unused flag.
	// TODO(ntnn): We need a way to shut this off if it isn't required.
	args = append(args, "--shard-external-url=https://127.0.0.1:6443")

	// --cache-kubeconfig is required for kcp's own virtual-workspace.
	// TODO(ntnn): We need a way to shut this off if it isn't required.
	if getEffectiveCacheRef(vw) != "" {
		args = append(args, fmt.Sprintf("--cache-kubeconfig=%s/kubeconfig", getCacheServerKubeconfigMountPath()))
	}

	args = append(args, utils.GetLoggingArgs(vw.Spec.VirtualWorkspace.Logging)...)

	if vw.Spec.VirtualWorkspace.ExtraArgs != nil {
		args = append(args, vw.Spec.VirtualWorkspace.ExtraArgs...)
	}

	return args
}
