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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func TestDeploymentReconciler(t *testing.T) {
	tests := []struct {
		name           string
		frontProxy     *operatorv1alpha1.FrontProxy
		rootShard      *operatorv1alpha1.RootShard
		expectedName   string
		validateDeploy func(*testing.T, *appsv1.Deployment)
	}{
		{
			name: "basic deployment configuration",
			frontProxy: &operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-front-proxy",
				},
				Spec: operatorv1alpha1.FrontProxySpec{
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{
							Name: "test-root-shard",
						},
					},
				},
			},
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-root-shard",
				},
			},
			expectedName: resources.GetFrontProxyDeploymentName(&operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{Name: "test-front-proxy"},
			}),
			validateDeploy: func(t *testing.T, dep *appsv1.Deployment) {
				assert.Equal(t, int32(2), *dep.Spec.Replicas)
				assert.Len(t, dep.Spec.Template.Spec.Containers, 1)

				container := dep.Spec.Template.Spec.Containers[0]
				assert.Equal(t, "kcp-front-proxy", container.Name)
				assert.Equal(t, "/kcp-front-proxy", container.Command[0])

				// Check for required volume mounts
				volumeMountNames := make(map[string]bool)
				for _, vm := range container.VolumeMounts {
					volumeMountNames[vm.Name] = true
				}

				expectedMounts := []string{
					resources.GetFrontProxyDynamicKubeconfigName(&operatorv1alpha1.RootShard{ObjectMeta: metav1.ObjectMeta{Name: "test-root-shard"}}, &operatorv1alpha1.FrontProxy{ObjectMeta: metav1.ObjectMeta{Name: "test-front-proxy"}}),
					resources.GetFrontProxyCertificateName(&operatorv1alpha1.RootShard{ObjectMeta: metav1.ObjectMeta{Name: "test-root-shard"}}, &operatorv1alpha1.FrontProxy{ObjectMeta: metav1.ObjectMeta{Name: "test-front-proxy"}}, operatorv1alpha1.KubeconfigCertificate),
					resources.GetFrontProxyCertificateName(&operatorv1alpha1.RootShard{ObjectMeta: metav1.ObjectMeta{Name: "test-root-shard"}}, &operatorv1alpha1.FrontProxy{ObjectMeta: metav1.ObjectMeta{Name: "test-front-proxy"}}, operatorv1alpha1.ServerCertificate),
					resources.GetFrontProxyCertificateName(&operatorv1alpha1.RootShard{ObjectMeta: metav1.ObjectMeta{Name: "test-root-shard"}}, &operatorv1alpha1.FrontProxy{ObjectMeta: metav1.ObjectMeta{Name: "test-front-proxy"}}, operatorv1alpha1.RequestHeaderClientCertificate),
					resources.GetFrontProxyConfigName(&operatorv1alpha1.FrontProxy{ObjectMeta: metav1.ObjectMeta{Name: "test-front-proxy"}}),
					resources.GetRootShardCAName(&operatorv1alpha1.RootShard{ObjectMeta: metav1.ObjectMeta{Name: "test-root-shard"}}, operatorv1alpha1.FrontProxyClientCA),
					resources.GetRootShardCAName(&operatorv1alpha1.RootShard{ObjectMeta: metav1.ObjectMeta{Name: "test-root-shard"}}, operatorv1alpha1.RootCA),
				}

				for _, expectedMount := range expectedMounts {
					assert.True(t, volumeMountNames[expectedMount], "Expected volume mount %s not found", expectedMount)
				}

				// Check readiness probe
				assert.NotNil(t, container.ReadinessProbe)
				assert.Equal(t, "/readyz", container.ReadinessProbe.HTTPGet.Path)
				assert.Equal(t, "https", container.ReadinessProbe.HTTPGet.Port.StrVal)

				// Check liveness probe
				assert.NotNil(t, container.LivenessProbe)
				assert.Equal(t, "/livez", container.LivenessProbe.HTTPGet.Path)
				assert.Equal(t, "https", container.LivenessProbe.HTTPGet.Port.StrVal)
			},
		},
		{
			name: "custom replicas",
			frontProxy: &operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-front-proxy",
				},
				Spec: operatorv1alpha1.FrontProxySpec{
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{
							Name: "test-root-shard",
						},
					},
					Replicas: ptr.To[int32](3),
				},
			},
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-root-shard",
				},
			},
			expectedName: resources.GetFrontProxyDeploymentName(&operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{Name: "test-front-proxy"},
			}),
			validateDeploy: func(t *testing.T, dep *appsv1.Deployment) {
				assert.Equal(t, int32(3), *dep.Spec.Replicas)
			},
		},
		{
			name: "custom image",
			frontProxy: &operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-front-proxy",
				},
				Spec: operatorv1alpha1.FrontProxySpec{
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{
							Name: "test-root-shard",
						},
					},
					Image: &operatorv1alpha1.ImageSpec{
						Repository: "custom-repo/kcp",
						Tag:        "v1.0.0",
					},
				},
			},
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-root-shard",
				},
			},
			expectedName: resources.GetFrontProxyDeploymentName(&operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{Name: "test-front-proxy"},
			}),
			validateDeploy: func(t *testing.T, dep *appsv1.Deployment) {
				container := dep.Spec.Template.Spec.Containers[0]
				assert.Equal(t, "custom-repo/kcp:v1.0.0", container.Image)
			},
		},
		{
			name: "auth configuration with drop groups",
			frontProxy: &operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-front-proxy",
				},
				Spec: operatorv1alpha1.FrontProxySpec{
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{
							Name: "test-root-shard",
						},
					},
					Auth: &operatorv1alpha1.AuthSpec{
						DropGroups: []string{"group1", "group2"},
					},
				},
			},
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-root-shard",
				},
			},
			expectedName: resources.GetFrontProxyDeploymentName(&operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{Name: "test-front-proxy"},
			}),
			validateDeploy: func(t *testing.T, dep *appsv1.Deployment) {
				container := dep.Spec.Template.Spec.Containers[0]
				args := container.Args

				// Debug: print all arguments
				t.Logf("Generated args: %v", args)

				dropGroupsArg := ""
				for i, arg := range args {
					t.Logf("Checking arg %d: '%s' (len: %d)", i, arg, len(arg))
					if strings.HasPrefix(arg, "--authentication-drop-groups=") {
						dropGroupsArg = arg
						t.Logf("  Found drop groups arg: '%s'", arg)
						break
					}
				}

				assert.NotEmpty(t, dropGroupsArg, "Drop groups argument not found")
				assert.Contains(t, dropGroupsArg, "group1")
				assert.Contains(t, dropGroupsArg, "group2")
			},
		},
		{
			name: "auth configuration with pass on groups",
			frontProxy: &operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-front-proxy",
				},
				Spec: operatorv1alpha1.FrontProxySpec{
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{
							Name: "test-root-shard",
						},
					},
					Auth: &operatorv1alpha1.AuthSpec{
						PassOnGroups: []string{"group3", "group4"},
					},
				},
			},
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-root-shard",
				},
			},
			expectedName: resources.GetFrontProxyDeploymentName(&operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{Name: "test-front-proxy"},
			}),
			validateDeploy: func(t *testing.T, dep *appsv1.Deployment) {
				container := dep.Spec.Template.Spec.Containers[0]
				args := container.Args

				// Debug: print all arguments
				t.Logf("Generated args: %v", args)

				passOnGroupsArg := ""
				for i, arg := range args {
					t.Logf("Checking arg %d: '%s' (len: %d)", i, arg, len(arg))
					if strings.HasPrefix(arg, "--authentication-pass-on-groups=") {
						passOnGroupsArg = arg
						t.Logf("  Found pass on groups arg: '%s'", arg)
						break
					}
				}

				assert.NotEmpty(t, passOnGroupsArg, "Pass on groups argument not found")
				assert.Contains(t, passOnGroupsArg, "group3")
				assert.Contains(t, passOnGroupsArg, "group4")
			},
		},
		{
			name: "oidc provider configured",
			frontProxy: &operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-front-proxy",
				},
				Spec: operatorv1alpha1.FrontProxySpec{
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{
							Name: "test-root-shard",
						},
					},
					Auth: &operatorv1alpha1.AuthSpec{
						OIDC: &operatorv1alpha1.OIDCConfiguration{
							IssuerURL:     "https://issuer.example.com",
							ClientID:      "my-client-id",
							UsernameClaim: "sub",
							GroupsClaim:   "groups",
						},
					},
				},
			},
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-root-shard",
				},
			},
			expectedName: resources.GetFrontProxyDeploymentName(&operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{Name: "test-front-proxy"},
			}),
			validateDeploy: func(t *testing.T, dep *appsv1.Deployment) {
				container := dep.Spec.Template.Spec.Containers[0]
				args := container.Args
				// Debug: print all arguments
				t.Logf("Generated args: %v", args)
				// Check for OIDC flags
				foundIssuer := false
				foundClientID := false
				for _, arg := range args {
					if strings.HasPrefix(arg, "--oidc-issuer-url=") && strings.Contains(arg, "issuer.example.com") {
						foundIssuer = true
					}
					if strings.HasPrefix(arg, "--oidc-client-id=") && strings.Contains(arg, "my-client-id") {
						foundClientID = true
					}
				}
				assert.True(t, foundIssuer, "OIDC issuer flag not found or incorrect")
				assert.True(t, foundClientID, "OIDC client-id flag not found or incorrect")
			},
		},
		{
			name: "service account authentication configured",
			frontProxy: &operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-front-proxy",
				},
				Spec: operatorv1alpha1.FrontProxySpec{
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{
							Name: "test-root-shard",
						},
					},
					Auth: &operatorv1alpha1.AuthSpec{
						ServiceAccount: &operatorv1alpha1.ServiceAccountAuthentication{
							Enabled: true,
						},
					},
				},
			},
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-root-shard",
				},
				Status: operatorv1alpha1.RootShardStatus{
					Shards: []operatorv1alpha1.ShardReference{
						{
							Name: "test-root-shard",
						},
						{
							Name: "test-shard-2",
						},
					},
				},
			},
			expectedName: resources.GetFrontProxyDeploymentName(&operatorv1alpha1.FrontProxy{
				ObjectMeta: metav1.ObjectMeta{Name: "test-front-proxy"},
			}),
			validateDeploy: func(t *testing.T, dep *appsv1.Deployment) {
				container := dep.Spec.Template.Spec.Containers[0]
				args := container.Args
				// Debug: print all arguments
				t.Logf("Generated args: %v", args)
				// Check for service account lookup flag
				foundServiceAccountLookup := false
				foundShard1 := false
				foundShard2 := false

				for _, arg := range args {
					if strings.HasPrefix(arg, "--service-account-lookup=false") {
						foundServiceAccountLookup = true
					}
					if strings.HasPrefix(arg, "--service-account-key-file=/etc/kcp/tls/test-root-shard/service-account/tls.key") {
						foundShard1 = true
					}
					if strings.HasPrefix(arg, "--service-account-key-file=/etc/kcp/tls/test-shard-2/service-account/tls.key") {
						foundShard2 = true
					}
				}
				assert.True(t, foundServiceAccountLookup, "Service account lookup flag not found or incorrect")
				assert.True(t, foundShard1, "Shard 1 service account key file not found or incorrect")
				assert.True(t, foundShard2, "Shard 2 service account key file not found or incorrect")

				foundShard1Volume := false
				foundShard2Volume := false

				for _, volume := range dep.Spec.Template.Spec.Volumes {
					if volume.Name == resources.GetRootShardCertificateName(&operatorv1alpha1.RootShard{ObjectMeta: metav1.ObjectMeta{Name: "test-root-shard"}}, operatorv1alpha1.ServiceAccountCertificate) {
						foundShard1Volume = true
					}
					if volume.Name == resources.GetShardCertificateName(&operatorv1alpha1.Shard{ObjectMeta: metav1.ObjectMeta{Name: "test-shard-2"}}, operatorv1alpha1.ServiceAccountCertificate) {
						foundShard2Volume = true
					}
				}

				assert.True(t, foundShard1Volume, "Shard 1 service account volume not found or incorrect")
				assert.True(t, foundShard2Volume, "Shard 2 service account volume not found or incorrect")

				foundShard1VolumeMount := false
				foundShard2VolumeMount := false

				for _, volumeMount := range container.VolumeMounts {
					if volumeMount.Name == resources.GetRootShardCertificateName(&operatorv1alpha1.RootShard{ObjectMeta: metav1.ObjectMeta{Name: "test-root-shard"}}, operatorv1alpha1.ServiceAccountCertificate) {
						foundShard1VolumeMount = true
					}
					if volumeMount.Name == resources.GetShardCertificateName(&operatorv1alpha1.Shard{ObjectMeta: metav1.ObjectMeta{Name: "test-shard-2"}}, operatorv1alpha1.ServiceAccountCertificate) {
						foundShard2VolumeMount = true
					}
				}

				assert.True(t, foundShard1VolumeMount, "Shard 1 service account volume mount not found or incorrect")
				assert.True(t, foundShard2VolumeMount, "Shard 2 service account volume mount not found or incorrect")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fpReconciler := NewFrontProxy(tt.frontProxy, tt.rootShard)
			name, reconcilerFunc := fpReconciler.deploymentReconciler()()

			assert.Equal(t, tt.expectedName, name)

			// Create a base deployment
			dep := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:      "kcp-front-proxy",
									Resources: corev1.ResourceRequirements{},
								},
							},
						},
					},
				},
			}

			// Apply the reconciler
			result, err := reconcilerFunc(dep)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Validate the result
			tt.validateDeploy(t, result)
		})
	}
}

func TestGetArgs(t *testing.T) {
	tests := []struct {
		name     string
		spec     *operatorv1alpha1.FrontProxySpec
		expected []string
	}{
		{
			name: "default args",
			spec: &operatorv1alpha1.FrontProxySpec{},
			expected: []string{
				"--secure-port=6443",
				"--root-kubeconfig=/etc/kcp-front-proxy/kubeconfig/kubeconfig",
				"--shards-kubeconfig=/etc/kcp-front-proxy/kubeconfig/kubeconfig",
				"--tls-private-key-file=/etc/kcp-front-proxy/tls/tls.key",
				"--tls-cert-file=/etc/kcp-front-proxy/tls/tls.crt",
				"--mapping-file=/etc/kcp-front-proxy/config/path-mapping.yaml",
				"--client-ca-file=/etc/kcp-front-proxy/client-ca/tls.crt",
			},
		},
		{
			name: "with drop groups",
			spec: &operatorv1alpha1.FrontProxySpec{
				Auth: &operatorv1alpha1.AuthSpec{
					DropGroups: []string{"group1", "group2"},
				},
			},
			expected: []string{
				"--secure-port=6443",
				"--root-kubeconfig=/etc/kcp-front-proxy/kubeconfig/kubeconfig",
				"--shards-kubeconfig=/etc/kcp-front-proxy/kubeconfig/kubeconfig",
				"--tls-private-key-file=/etc/kcp-front-proxy/tls/tls.key",
				"--tls-cert-file=/etc/kcp-front-proxy/tls/tls.crt",
				"--mapping-file=/etc/kcp-front-proxy/config/path-mapping.yaml",
				"--client-ca-file=/etc/kcp-front-proxy/client-ca/tls.crt",
				"--authentication-drop-groups=\"group1,group2\"",
			},
		},
		{
			name: "with pass on groups",
			spec: &operatorv1alpha1.FrontProxySpec{
				Auth: &operatorv1alpha1.AuthSpec{
					PassOnGroups: []string{"group3", "group4"},
				},
			},
			expected: []string{
				"--secure-port=6443",
				"--root-kubeconfig=/etc/kcp-front-proxy/kubeconfig/kubeconfig",
				"--shards-kubeconfig=/etc/kcp-front-proxy/kubeconfig/kubeconfig",
				"--tls-private-key-file=/etc/kcp-front-proxy/tls/tls.key",
				"--tls-cert-file=/etc/kcp-front-proxy/tls/tls.crt",
				"--mapping-file=/etc/kcp-front-proxy/config/path-mapping.yaml",
				"--client-ca-file=/etc/kcp-front-proxy/client-ca/tls.crt",
				"--authentication-pass-on-groups=\"group3,group4\"",
			},
		},
		{
			name: "with both drop and pass on groups",
			spec: &operatorv1alpha1.FrontProxySpec{
				Auth: &operatorv1alpha1.AuthSpec{
					DropGroups:   []string{"group1"},
					PassOnGroups: []string{"group2"},
				},
			},
			expected: []string{
				"--secure-port=6443",
				"--root-kubeconfig=/etc/kcp-front-proxy/kubeconfig/kubeconfig",
				"--shards-kubeconfig=/etc/kcp-front-proxy/kubeconfig/kubeconfig",
				"--tls-private-key-file=/etc/kcp-front-proxy/tls/tls.key",
				"--tls-cert-file=/etc/kcp-front-proxy/tls/tls.crt",
				"--mapping-file=/etc/kcp-front-proxy/config/path-mapping.yaml",
				"--client-ca-file=/etc/kcp-front-proxy/client-ca/tls.crt",
				"--authentication-drop-groups=\"group1\"",
				"--authentication-pass-on-groups=\"group2\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := NewFrontProxy(
				&operatorv1alpha1.FrontProxy{Spec: *tt.spec},
				&operatorv1alpha1.RootShard{},
			)

			result := rec.getArgs()
			assert.Equal(t, tt.expected, result)
		})
	}
}
