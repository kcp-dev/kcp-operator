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

package frontproxy

import (
	"fmt"

	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func DeploymentReconciler(frontProxy *operatorv1alpha1.FrontProxy, rootShard *operatorv1alpha1.RootShard) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return resources.GetFrontProxyDeploymentName(frontProxy), func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.SetLabels(resources.GetFrontProxyResourceLabels(frontProxy))
			dep.Spec.Selector = &v1.LabelSelector{
				MatchLabels: resources.GetFrontProxyResourceLabels(frontProxy),
			}
			dep.Spec.Template.ObjectMeta.SetLabels(resources.GetFrontProxyResourceLabels(frontProxy))

			image, _ := resources.GetImageSettings(frontProxy.Spec.Image)
			args := getArgs()

			container := corev1.Container{
				Name:    "kcp-front-proxy",
				Image:   image,
				Command: []string{"/kcp-front-proxy"},
				Args:    args,
				SecurityContext: &corev1.SecurityContext{
					SeccompProfile: &corev1.SeccompProfile{
						Type: corev1.SeccompProfileTypeRuntimeDefault,
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
							Path:   "/livez",
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
							Path:   "/readyz",
							Port:   intstr.FromString("https"),
							Scheme: corev1.URISchemeHTTPS,
						},
					},
				},
			}

			volumes := []corev1.Volume{}
			volumeMounts := []corev1.VolumeMount{}

			// front-proxy dynamic kubeconfig
			volumes = append(volumes, corev1.Volume{
				Name: resources.GetFrontProxyDynamicKubeconfigName(rootShard, frontProxy),
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: resources.GetFrontProxyDynamicKubeconfigName(rootShard, frontProxy),
					},
				},
			})
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      resources.GetFrontProxyDynamicKubeconfigName(rootShard, frontProxy),
				ReadOnly:  false, // as FrontProxy writes to it to work with different shards
				MountPath: frontProxyBasepath + "/kubeconfig",
			})

			// front-proxy kubeconfig client cert
			kubeconfigClientCertName := resources.GetFrontProxyCertificateName(rootShard, frontProxy, operatorv1alpha1.KubeconfigCertificate)
			volumes = append(volumes, corev1.Volume{
				Name: kubeconfigClientCertName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: kubeconfigClientCertName,
					},
				},
			})
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      kubeconfigClientCertName,
				ReadOnly:  true,
				MountPath: frontProxyBasepath + "/kubeconfig-client-cert",
			})

			// front-proxy service-account cert
			volumes = append(volumes, corev1.Volume{
				Name: resources.GetRootShardCertificateName(rootShard, operatorv1alpha1.ServiceAccountCertificate),
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: resources.GetRootShardCertificateName(rootShard, operatorv1alpha1.ServiceAccountCertificate),
					},
				},
			})
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      resources.GetRootShardCertificateName(rootShard, operatorv1alpha1.ServiceAccountCertificate),
				ReadOnly:  true,
				MountPath: fmt.Sprintf("/etc/kcp/tls/%s", string(operatorv1alpha1.ServiceAccountCertificate)),
			})

			// front-proxy server cert
			volumes = append(volumes, corev1.Volume{
				Name: resources.GetRootShardCertificateName(rootShard, operatorv1alpha1.ServerCertificate),
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: resources.GetRootShardCertificateName(rootShard, operatorv1alpha1.ServerCertificate),
					},
				},
			})
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      resources.GetRootShardCertificateName(rootShard, operatorv1alpha1.ServerCertificate),
				ReadOnly:  true,
				MountPath: frontProxyBasepath + "/tls",
			})

			// front-proxy requestheader client cert
			requestHeaderClientCertName := resources.GetFrontProxyCertificateName(rootShard, frontProxy, operatorv1alpha1.RequestHeaderClientCertificate)
			volumes = append(volumes, corev1.Volume{
				Name: requestHeaderClientCertName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: requestHeaderClientCertName,
					},
				},
			})
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      requestHeaderClientCertName,
				ReadOnly:  true,
				MountPath: frontProxyBasepath + "/requestheader-client",
			})

			// front-proxy config
			cmName := resources.GetFrontProxyConfigName(frontProxy)
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

			// rootshard frontproxy client ca
			rsClientCAName := resources.GetRootShardCAName(rootShard, operatorv1alpha1.FrontProxyClientCA)
			volumes = append(volumes, corev1.Volume{
				Name: rsClientCAName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: rsClientCAName,
					},
				},
			})
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      rsClientCAName,
				ReadOnly:  true,
				MountPath: frontProxyBasepath + "/client-ca",
			})

			// kcp rootshard root ca
			rootCAName := resources.GetRootShardCAName(rootShard, operatorv1alpha1.RootCA)
			volumes = append(volumes, corev1.Volume{
				Name: rootCAName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: rootCAName,
					},
				},
			})
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      rootCAName,
				ReadOnly:  true,
				MountPath: kcpBasepath + "/tls/ca",
			})

			container.VolumeMounts = volumeMounts

			if frontProxy.Spec.Replicas != nil {
				dep.Spec.Replicas = frontProxy.Spec.Replicas
			} else if dep.Spec.Replicas == nil {
				dep.Spec.Replicas = ptr.To[int32](2)
			}

			dep.Spec.Template.Spec.Volumes = volumes
			dep.Spec.Template.Spec.Containers = []corev1.Container{container}

			return dep, nil
		}
	}
}

func getArgs() []string {
	args := []string{
		"--secure-port=6443",
		"--root-kubeconfig=/etc/kcp-front-proxy/kubeconfig/kubeconfig",
		"--shards-kubeconfig=/etc/kcp-front-proxy/kubeconfig/kubeconfig",
		"--tls-private-key-file=/etc/kcp-front-proxy/tls/tls.key",
		"--tls-cert-file=/etc/kcp-front-proxy/tls/tls.crt",
		"--client-ca-file=/etc/kcp-front-proxy/client-ca/tls.crt",
		"--mapping-file=/etc/kcp-front-proxy/config/path-mapping.yaml",
		"--service-account-key-file=/etc/kcp/tls/service-account/tls.key",
	}

	return args
}
