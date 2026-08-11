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

package compiledvirtualworkspace

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func newCompiledVirtualWorkspace(spec operatorv1alpha1.VirtualWorkspaceSpec) *deployv1alpha1.CompiledVirtualWorkspace {
	return &deployv1alpha1.CompiledVirtualWorkspace{
		ObjectMeta: metav1.ObjectMeta{Name: "vw", Namespace: "kcp-system"},
		Spec: deployv1alpha1.CompiledVirtualWorkspaceSpec{
			VirtualWorkspace: spec,
			RootShard: deployv1alpha1.NamedRootShardSpec{
				Name: "rooty",
				Spec: operatorv1alpha1.RootShardSpec{
					Cache: operatorv1alpha1.RootShardCacheConfig{
						Reference: &corev1.LocalObjectReference{Name: "cachey"},
					},
				},
			},
		},
	}
}

func reconcileDeployment(t *testing.T, vw *deployv1alpha1.CompiledVirtualWorkspace) *appsv1.Deployment {
	t.Helper()

	_, reconciler := DeploymentReconciler(vw)()
	dep, err := reconciler(&appsv1.Deployment{})
	require.NoError(t, err)

	return dep
}

func mountPaths(container corev1.Container) []string {
	paths := make([]string, 0, len(container.VolumeMounts))
	for _, mount := range container.VolumeMounts {
		paths = append(paths, mount.MountPath)
	}

	return paths
}

func secretNames(dep *appsv1.Deployment) []string {
	var names []string
	for _, volume := range dep.Spec.Template.Spec.Volumes {
		if volume.Secret != nil {
			names = append(names, volume.Secret.SecretName)
		}
	}

	return names
}

func hasArgPrefix(args []string, prefix string) bool {
	for _, arg := range args {
		if strings.HasPrefix(arg, prefix) {
			return true
		}
	}

	return false
}

func TestDeploymentServerCommand(t *testing.T) {
	t.Run("kcp's own server keeps the default entrypoint and its kcp-specific args", func(t *testing.T) {
		dep := reconcileDeployment(t, newCompiledVirtualWorkspace(operatorv1alpha1.VirtualWorkspaceSpec{}))

		container := dep.Spec.Template.Spec.Containers[0]
		assert.Equal(t, []string{DefaultServerCommand}, container.Command)
		assert.True(t, hasArgPrefix(container.Args, "--shard-external-url="))
		assert.True(t, hasArgPrefix(container.Args, "--cache-kubeconfig="))
	})

	t.Run("a custom command replaces the entrypoint and drops the kcp-specific args", func(t *testing.T) {
		dep := reconcileDeployment(t, newCompiledVirtualWorkspace(operatorv1alpha1.VirtualWorkspaceSpec{
			Command:   []string{"/access-vw"},
			ExtraArgs: []string{"--endpoint-base=https://kcp.example.com/clusters/"},
		}))

		container := dep.Spec.Template.Spec.Containers[0]
		assert.Equal(t, []string{"/access-vw"}, container.Command)

		// Flags a custom server would reject are gone ...
		assert.False(t, hasArgPrefix(container.Args, "--shard-external-url="))
		assert.False(t, hasArgPrefix(container.Args, "--cache-kubeconfig="))

		// ... while everything an aggregated apiserver needs is still passed.
		for _, prefix := range []string{
			"--client-ca-file=",
			"--tls-cert-file=",
			"--tls-private-key-file=",
			"--bind-address=",
			"--secure-port=",
			"--requestheader-client-ca-file=",
			"--requestheader-allowed-names=",
			"--kubeconfig=",
			"--endpoint-base=",
		} {
			assert.True(t, hasArgPrefix(container.Args, prefix), "expected an argument starting with %q", prefix)
		}
	})

	t.Run("a custom server does not get the cache server volumes", func(t *testing.T) {
		dep := reconcileDeployment(t, newCompiledVirtualWorkspace(operatorv1alpha1.VirtualWorkspaceSpec{
			Command: []string{"/access-vw"},
		}))

		for _, volume := range dep.Spec.Template.Spec.Volumes {
			assert.NotContains(t, volume.Name, "cache-server")
		}
	})
}

// TestCustomServerArgumentSurface pins the exact set of flags a custom server is asked to
// understand. Anything added to the generated arguments has to be a flag every aggregated
// apiserver accepts, or it belongs behind the isCustomServer check -- a custom server exits on an
// unknown flag rather than ignoring it, so widening this set silently breaks every non-kcp server.
func TestCustomServerArgumentSurface(t *testing.T) {
	allowed := []string{
		"--client-ca-file",
		"--tls-private-key-file",
		"--tls-cert-file",
		"--bind-address",
		"--secure-port",
		"--requestheader-client-ca-file",
		"--requestheader-allowed-names",
		"--requestheader-username-headers",
		"--requestheader-group-headers",
		"--requestheader-extra-headers-prefix",
		"--kubeconfig",
	}

	vw := newCompiledVirtualWorkspace(operatorv1alpha1.VirtualWorkspaceSpec{
		Command: []string{"/custom-server"},
		Logging: &operatorv1alpha1.LoggingSpec{Level: 4},
	})

	for _, arg := range getArgs(vw) {
		name, _, _ := strings.Cut(arg, "=")

		// Logging verbosity is not an apiserver flag, but every klog-based binary has it.
		if name == "-v" {
			continue
		}

		assert.Contains(t, allowed, name,
			"%q is generated for custom servers; either it is universally understood and belongs in this list, or it belongs behind isCustomServer()", arg)
	}
}

func TestDeploymentServerKubeconfig(t *testing.T) {
	const adminArg = "--kubeconfig=/etc/kcp/logical-cluster-admin-kubeconfig/kubeconfig"

	t.Run("the target's logical-cluster-admin kubeconfig is used by default", func(t *testing.T) {
		dep := reconcileDeployment(t, newCompiledVirtualWorkspace(operatorv1alpha1.VirtualWorkspaceSpec{}))

		server := dep.Spec.Template.Spec.Containers[0]
		assert.Contains(t, server.Args, adminArg)
		assert.Contains(t, mountPaths(server), "/etc/kcp/logical-cluster-admin-kubeconfig")
		assert.Contains(t, mountPaths(server), "/etc/kcp/tls/logical-cluster-admin")

		for _, volume := range dep.Spec.Template.Spec.Volumes {
			assert.NotEqual(t, "server-kubeconfig", volume.Name)
		}
	})

	t.Run("an init container without its own kubeconfig keeps the admin fallback", func(t *testing.T) {
		dep := reconcileDeployment(t, newCompiledVirtualWorkspace(operatorv1alpha1.VirtualWorkspaceSpec{
			Command:             []string{"/access-vw"},
			KubeconfigSecretRef: &corev1.LocalObjectReference{Name: "access-vw-server-kubeconfig"},
			InitContainers: []operatorv1alpha1.VirtualWorkspaceInitContainer{{
				Name:    "init",
				Command: []string{"/access-vw-init"},
				Args:    []string{adminArg},
			}},
		}))

		require.Len(t, dep.Spec.Template.Spec.InitContainers, 1)
		init := dep.Spec.Template.Spec.InitContainers[0]

		assert.Contains(t, mountPaths(init), "/etc/kcp/logical-cluster-admin-kubeconfig")
		assert.Contains(t, mountPaths(init), "/etc/kcp/tls/logical-cluster-admin")
		assert.NotContains(t, mountPaths(dep.Spec.Template.Spec.Containers[0]), "/etc/kcp/logical-cluster-admin-kubeconfig")
	})

	t.Run("a referenced kubeconfig is mounted and used by the server", func(t *testing.T) {
		vw := newCompiledVirtualWorkspace(operatorv1alpha1.VirtualWorkspaceSpec{
			Command:             []string{"/access-vw"},
			KubeconfigSecretRef: &corev1.LocalObjectReference{Name: "access-vw-server-kubeconfig"},
		})

		dep := reconcileDeployment(t, vw)

		server := dep.Spec.Template.Spec.Containers[0]
		assert.Contains(t, server.Args, "--kubeconfig=/etc/kcp/server-kubeconfig/kubeconfig")
		assert.NotContains(t, server.Args, adminArg)

		assert.Contains(t, mountPaths(server), "/etc/kcp/server-kubeconfig")
		assert.Contains(t, secretNames(dep), "access-vw-server-kubeconfig")
	})

	t.Run("server and init containers hold separate credentials", func(t *testing.T) {
		vw := newCompiledVirtualWorkspace(operatorv1alpha1.VirtualWorkspaceSpec{
			Command:             []string{"/access-vw"},
			KubeconfigSecretRef: &corev1.LocalObjectReference{Name: "access-vw-server-kubeconfig"},
			InitContainers: []operatorv1alpha1.VirtualWorkspaceInitContainer{{
				Name:                "init",
				KubeconfigSecretRef: &corev1.LocalObjectReference{Name: "access-vw-bootstrap-kubeconfig"},
				Command:             []string{"/access-vw-init"},
				Args:                []string{"--kubeconfig=/etc/kcp/init-kubeconfig/kubeconfig"},
			}},
		})

		dep := reconcileDeployment(t, vw)
		require.Len(t, dep.Spec.Template.Spec.InitContainers, 1)

		server := dep.Spec.Template.Spec.Containers[0]
		init := dep.Spec.Template.Spec.InitContainers[0]

		assert.Contains(t, secretNames(dep), "access-vw-bootstrap-kubeconfig")

		// Each kubeconfig reaches exactly one container, so neither can read the other's
		// credential out of its own filesystem.
		assert.Contains(t, mountPaths(init), "/etc/kcp/init-kubeconfig")
		assert.NotContains(t, mountPaths(server), "/etc/kcp/init-kubeconfig")
		assert.Contains(t, mountPaths(server), "/etc/kcp/server-kubeconfig")
		assert.NotContains(t, mountPaths(init), "/etc/kcp/server-kubeconfig")

		// Neither container keeps the privileged fallback it no longer needs, so no admin key
		// material is left lying around in a pod that was given scoped identities.
		for _, container := range []corev1.Container{server, init} {
			assert.NotContains(t, mountPaths(container), "/etc/kcp/logical-cluster-admin-kubeconfig")
			assert.NotContains(t, mountPaths(container), "/etc/kcp/tls/logical-cluster-admin")
		}

		// Whereas mounts that are not credentials stay shared.
		assert.Contains(t, mountPaths(init), "/etc/kcp/tls/ca/server")
		assert.Contains(t, mountPaths(server), "/etc/kcp/tls/ca/server")
	})

	t.Run("each init container gets its own kubeconfig volume", func(t *testing.T) {
		vw := newCompiledVirtualWorkspace(operatorv1alpha1.VirtualWorkspaceSpec{
			Command: []string{"/access-vw"},
			InitContainers: []operatorv1alpha1.VirtualWorkspaceInitContainer{
				{
					Name:                "first",
					KubeconfigSecretRef: &corev1.LocalObjectReference{Name: "first-kubeconfig"},
				},
				{
					Name:                "second",
					KubeconfigSecretRef: &corev1.LocalObjectReference{Name: "second-kubeconfig"},
				},
			},
		})

		dep := reconcileDeployment(t, vw)
		require.Len(t, dep.Spec.Template.Spec.InitContainers, 2)

		// Distinct volumes, but the same path in each container.
		names := map[string]string{}
		for _, volume := range dep.Spec.Template.Spec.Volumes {
			if volume.Secret != nil {
				names[volume.Name] = volume.Secret.SecretName
			}
		}
		assert.Equal(t, "first-kubeconfig", names["init-first-kubeconfig"])
		assert.Equal(t, "second-kubeconfig", names["init-second-kubeconfig"])

		for _, init := range dep.Spec.Template.Spec.InitContainers {
			assert.Contains(t, mountPaths(init), "/etc/kcp/init-kubeconfig")
		}
	})
}

func TestDeploymentInitContainers(t *testing.T) {
	t.Run("none are added by default", func(t *testing.T) {
		dep := reconcileDeployment(t, newCompiledVirtualWorkspace(operatorv1alpha1.VirtualWorkspaceSpec{}))
		assert.Empty(t, dep.Spec.Template.Spec.InitContainers)
	})

	t.Run("they default to the server image and inherit its volume mounts", func(t *testing.T) {
		vw := newCompiledVirtualWorkspace(operatorv1alpha1.VirtualWorkspaceSpec{
			Command: []string{"/access-vw"},
			Image: &operatorv1alpha1.ImageSpec{
				Repository: "ghcr.io/kcp-dev/contrib-access-virtual-workspace",
				Tag:        "latest",
			},
			InitContainers: []operatorv1alpha1.VirtualWorkspaceInitContainer{{
				Name:    "init",
				Command: []string{"/access-vw-init"},
				Args:    []string{"--workspace-prefix=root:access"},
			}},
		})

		dep := reconcileDeployment(t, vw)
		require.Len(t, dep.Spec.Template.Spec.InitContainers, 1)

		init := dep.Spec.Template.Spec.InitContainers[0]
		server := dep.Spec.Template.Spec.Containers[0]

		assert.Equal(t, "init", init.Name)
		assert.Equal(t, server.Image, init.Image)
		assert.Equal(t, []string{"/access-vw-init"}, init.Command)
		assert.Equal(t, []string{"--workspace-prefix=root:access"}, init.Args)
		assert.Equal(t, server.VolumeMounts, init.VolumeMounts)
	})

	t.Run("an own image is used together with its pull secrets", func(t *testing.T) {
		vw := newCompiledVirtualWorkspace(operatorv1alpha1.VirtualWorkspaceSpec{
			Command: []string{"/access-vw"},
			InitContainers: []operatorv1alpha1.VirtualWorkspaceInitContainer{{
				Name: "init",
				Image: &operatorv1alpha1.ImageSpec{
					Repository:       "registry.example.com/bootstrapper",
					Tag:              "v1",
					ImagePullSecrets: []corev1.LocalObjectReference{{Name: "pull-secret"}},
				},
			}},
		})

		dep := reconcileDeployment(t, vw)
		require.Len(t, dep.Spec.Template.Spec.InitContainers, 1)

		assert.Equal(t, "registry.example.com/bootstrapper:v1", dep.Spec.Template.Spec.InitContainers[0].Image)
		assert.Contains(t, dep.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: "pull-secret"})
	})
}
