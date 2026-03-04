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

package frontproxy

import (
	"fmt"
	"strings"

	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/utils"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func (r *reconciler) deploymentReconciler() reconciling.NamedDeploymentReconcilerFactory {
	var (
		name         string
		imageSpec    *operatorv1alpha1.ImageSpec
		depResources *corev1.ResourceRequirements
		template     *operatorv1alpha1.DeploymentTemplate
		replicas     *int32
	)

	if r.frontProxy != nil {
		name = resources.GetFrontProxyDeploymentName(r.frontProxy)
		imageSpec = r.frontProxy.Spec.Image
		depResources = r.frontProxy.Spec.Resources
		template = r.frontProxy.Spec.DeploymentTemplate
		replicas = r.frontProxy.Spec.Replicas
	} else {
		name = resources.GetRootShardProxyDeploymentName(r.rootShard)

		if r.rootShard.Spec.Proxy != nil {
			imageSpec = r.rootShard.Spec.Proxy.Image
			depResources = r.rootShard.Spec.Proxy.Resources
			template = r.rootShard.Spec.Proxy.DeploymentTemplate
			replicas = r.rootShard.Spec.Proxy.Replicas
		}
	}

	return func() (string, reconciling.DeploymentReconciler) {
		return name, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			// Only set the selector on creation, as it's immutable
			if dep.Spec.Selector == nil {
				dep.Spec.Selector = &metav1.LabelSelector{
					MatchLabels: r.resourceLabels,
				}
			}
			dep.Spec.Template.SetLabels(r.resourceLabels)

			image, _ := resources.GetImageSettings(imageSpec)
			args := r.getArgs()

			container := corev1.Container{
				Name:    "kcp-front-proxy",
				Image:   image,
				Command: []string{"/kcp-front-proxy"},
				Args:    args,
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
			}

			volumes := []corev1.Volume{}
			volumeMounts := []corev1.VolumeMount{}

			mountSecret := func(secretName string, mountPath string, readOnly bool) {
				volumes = append(volumes, corev1.Volume{
					Name: secretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: secretName,
						},
					},
				})
				volumeMounts = append(volumeMounts, corev1.VolumeMount{
					Name:      secretName,
					ReadOnly:  readOnly,
					MountPath: mountPath,
				})
			}

			// front-proxy dynamic kubeconfig
			{
				var secretName string
				if r.frontProxy != nil {
					secretName = resources.GetFrontProxyDynamicKubeconfigName(r.rootShard, r.frontProxy)
				} else {
					secretName = resources.GetRootShardProxyDynamicKubeconfigName(r.rootShard)
				}

				// readonly=false because front-proxy updates the file to work with different shards
				mountSecret(secretName, frontProxyBasepath+"/kubeconfig", false)
			}

			// front-proxy kubeconfig client cert
			mountSecret(r.certName(operatorv1alpha1.KubeconfigCertificate), frontProxyBasepath+"/kubeconfig-client-cert", true)

			// front-proxy server cert
			mountSecret(r.certName(operatorv1alpha1.ServerCertificate), frontProxyBasepath+"/tls", true)

			// front-proxy requestheader client cert
			mountSecret(r.certName(operatorv1alpha1.RequestHeaderClientCertificate), frontProxyBasepath+"/requestheader-client", true)

			// kcp rootshard root ca
			mountSecret(resources.GetRootShardCAName(r.rootShard, operatorv1alpha1.RootCA), kcpBasepath+"/tls/ca", true)

			// If caBundleSecretRef is specified, mount the merged CA bundle secret.
			// This secret contains both kcp root CA and user-provided CA bundle merged together.
			if r.getCABundleSecretRef() != nil {
				mountSecret(r.mergedCABundleSecretName(), getCAMountPath(operatorv1alpha1.CABundleCA), true)
			}

			// Regular front-proxies use a dedicated client CA. However the internal rootshard proxy
			// uses the internal client CA instead to make it easier for the kcp-operator to just use
			// a single certificate to access all components.
			if r.frontProxy != nil {
				// rootshard frontproxy client ca
				mountSecret(resources.GetRootShardCAName(r.rootShard, operatorv1alpha1.FrontProxyClientCA), frontProxyBasepath+"/client-ca", true)
			} else {
				// kcp client ca
				mountSecret(resources.GetRootShardCAName(r.rootShard, operatorv1alpha1.ClientCA), kcpBasepath+"/tls/client-ca", true)
			}

			// front-proxy config
			{
				cmName := r.pathMappingConfigMapName()
				volumes = append(volumes, corev1.Volume{
					Name: cmName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: cmName,
							},
						},
					},
				})
				volumeMounts = append(volumeMounts, corev1.VolumeMount{
					Name:      cmName,
					ReadOnly:  true,
					MountPath: frontProxyBasepath + "/config",
				})
			}

			container.VolumeMounts = volumeMounts
			dep.Spec.Template.Spec.Volumes = volumes

			if replicas != nil {
				dep.Spec.Replicas = replicas
			} else if dep.Spec.Replicas == nil {
				dep.Spec.Replicas = ptr.To[int32](2)
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				utils.ApplyResources(container, depResources),
			}

			dep = utils.ApplyDeploymentTemplate(dep, template)

			if r.frontProxy != nil {
				dep = utils.ApplyFrontProxyAuthConfiguration(dep, r.frontProxy.Spec.Auth, r.rootShard)

				// If frontproxy has bundle annotation, store desired replicas in annotation then scale deployment to 0 locally
				if r.frontProxy.Annotations != nil && r.frontProxy.Annotations[resources.BundleAnnotation] != "" {
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
			} else if r.rootShard != nil {
				// If rootshard has bundle annotation, store desired replicas in annotation then scale proxy deployment to 0 locally
				if r.rootShard.Annotations != nil && r.rootShard.Annotations[resources.BundleAnnotation] != "" {
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
			}

			return dep, nil
		}
	}
}

var defaultArgs = []string{
	"--secure-port=6443",
	"--root-kubeconfig=/etc/kcp-front-proxy/kubeconfig/kubeconfig",
	"--shards-kubeconfig=/etc/kcp-front-proxy/kubeconfig/kubeconfig",
	"--tls-private-key-file=/etc/kcp-front-proxy/tls/tls.key",
	"--tls-cert-file=/etc/kcp-front-proxy/tls/tls.crt",
	"--mapping-file=/etc/kcp-front-proxy/config/path-mapping.yaml",
}

func (r *reconciler) getArgs() []string {
	args := defaultArgs

	// rootshard proxy mode
	if r.frontProxy == nil {
		args = append(args, fmt.Sprintf("--client-ca-file=%s/tls/client-ca/tls.crt", kcpBasepath))
		return args
	}

	// regular front-proxy
	args = append(args, fmt.Sprintf("--client-ca-file=%s/client-ca/tls.crt", frontProxyBasepath))

	if auth := r.frontProxy.Spec.Auth; auth != nil {
		if auth.DropGroups != nil {
			args = append(args, fmt.Sprintf("--authentication-drop-groups=%q", strings.Join(auth.DropGroups, ",")))
		}

		if auth.PassOnGroups != nil {
			args = append(args, fmt.Sprintf("--authentication-pass-on-groups=%q", strings.Join(auth.PassOnGroups, ",")))
		}
	}

	args = append(args, utils.GetLoggingArgs(r.frontProxy.Spec.Logging)...)

	if r.frontProxy.Spec.ExtraArgs != nil {
		args = append(args, r.frontProxy.Spec.ExtraArgs...)
	}

	return args
}
