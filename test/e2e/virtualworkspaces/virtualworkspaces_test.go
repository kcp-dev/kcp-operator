//go:build e2e

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

package virtualworkspaces

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"
	"github.com/kcp-dev/logicalcluster/v3"
	kcpcorev1alpha1 "github.com/kcp-dev/sdk/apis/core/v1alpha1"
	kcptenancyv1alpha1 "github.com/kcp-dev/sdk/apis/tenancy/v1alpha1"
	"github.com/kcp-dev/sdk/testing/server"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kcp-dev/kcp-operator/internal/resources/naming"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
	"github.com/kcp-dev/kcp-operator/test/conformance"
	"github.com/kcp-dev/kcp-operator/test/utils"
)

func skipIfNotMainBranch(t *testing.T) {
	if utils.GetKcpRelease().LessThan(semver.MustParse("0.31")) {
		t.Skip("This test requires kcp >= 0.31.")
	}
}

// TestExternalVirtualWorkspace tests deploying a RootShard with an external (standalone)
// virtual workspace server and verifies that workspaces can be created and queried
// through the tenancy virtual workspace.
func TestExternalVirtualWorkspace(t *testing.T) {
	skipIfNotMainBranch(t)

	ctrlruntime.SetLogger(logr.Discard())

	client := utils.GetKubeClient(t)
	ctx := context.Background()
	namingScheme := naming.NewVersion1()

	namespace := utils.CreateSelfDestructingNamespace(t, ctx, client, "external-vw")

	// Deploy the standalone, external kcp virtual-workspace (do not wait for readiness since both
	// the root shard and the VW will need to have a little dance before both are ready).
	vw := utils.DeployVirtualWorkspace(ctx, t, client, namingScheme, namespace.Name, "kcp-vw", false, func(vw *operatorv1alpha1.VirtualWorkspace) {
		vw.Spec.Target.RootShardRef = &corev1.LocalObjectReference{
			Name: "r00t", // Must match the rootshard name
		}
		vw.Spec.Resources = lowResourceRequirements()
	})

	// externalHostname must match whatever DeployFrontProxy chooses as the name for the FrontProxy
	externalHostname := utils.FrontProxyExternalHostname(namespace.Name, namingScheme)

	// Deploy a root shard that uses the external VW (in-process VW is disabled).
	rootShard := utils.DeployRootShard(ctx, t, client, namingScheme, namespace.Name, externalHostname, func(rs *operatorv1alpha1.RootShard) {
		// Point the root shard to use our external VW instead of the in-process one.
		rs.Spec.KCPVirtualWorkspace = &corev1.LocalObjectReference{
			Name: vw.Name,
		}
		rs.Spec.Resources = lowResourceRequirements()
		if rs.Spec.Proxy != nil {
			rs.Spec.Proxy.Resources = lowResourceRequirements()
		}
	})

	// deploy front-proxy
	utils.DeployFrontProxy(ctx, t, client, namespace.Name, rootShard.Name, externalHostname, func(fp *operatorv1alpha1.FrontProxy) {
		fp.Spec.Resources = lowResourceRequirements()
	})

	t.Log("Waiting for VirtualWorkspace pods to be ready...")
	waitForVirtualWorkspacePods(t, ctx, client, namingScheme, &vw)

	// Create a kubeconfig to access the root shard.
	configSecretName := "kubeconfig"
	rsConfig := operatorv1alpha1.Kubeconfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: namespace.Name,
		},
		Spec: operatorv1alpha1.KubeconfigSpec{
			Target: operatorv1alpha1.KubeconfigTarget{
				RootShardRef: &corev1.LocalObjectReference{
					Name: rootShard.Name,
				},
			},
			Username: "e2e",
			Validity: metav1.Duration{Duration: 2 * time.Hour},
			SecretRef: corev1.LocalObjectReference{
				Name: configSecretName,
			},
			Groups: []string{"system:kcp:admin"},
		},
	}

	t.Log("Creating kubeconfig for RootShard...")
	if err := client.Create(ctx, &rsConfig); err != nil {
		t.Fatal(err)
	}
	utils.WaitForObject(t, ctx, client, &corev1.Secret{}, types.NamespacedName{Namespace: rsConfig.Namespace, Name: rsConfig.Spec.SecretRef.Name})

	t.Log("Connecting to RootShard...")
	rootShardClient := utils.ConnectWithKubeconfig(t, ctx, client, namespace.Name, rsConfig.Name, logicalcluster.None)

	// Proof of life: list something every logicalcluster in kcp has.
	t.Log("Should be able to list Secrets in root workspace.")
	secrets := &corev1.SecretList{}
	if err := rootShardClient.List(ctx, secrets); err != nil {
		t.Fatalf("Failed to list secrets in kcp: %v", err)
	}

	// Create a workspace and wait for it to become ready.
	t.Log("Creating a workspace...")
	workspace := &kcptenancyv1alpha1.Workspace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-workspace",
		},
		Spec: kcptenancyv1alpha1.WorkspaceSpec{
			Type: &kcptenancyv1alpha1.WorkspaceTypeReference{
				Name: "universal",
			},
		},
	}
	if err := rootShardClient.Create(ctx, workspace); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	t.Log("Waiting for workspace to become ready...")
	err := wait.PollUntilContextTimeout(ctx, 500*time.Millisecond, 60*time.Second, false, func(ctx context.Context) (done bool, err error) {
		err = rootShardClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(workspace), workspace)
		if err != nil {
			return false, err
		}

		return workspace.Status.Phase == kcpcorev1alpha1.LogicalClusterPhaseReady, nil
	})
	if err != nil {
		t.Fatalf("Failed to wait for workspace to become ready: %v", err)
	}
	t.Logf("Workspace %q is ready.", workspace.Name)

	// Verify the workspace is visible by listing all workspaces through the virtual workspace.
	// When using an external VW, the tenancy APIs (workspaces.tenancy.kcp.io) are served
	// by the external VW server. This tests that the VW is properly integrated by connecting
	// directly to the VW service and listing workspaces through the tenancy apiexport.
	t.Log("Connecting to VirtualWorkspace service to list workspaces...")

	// Get the base kubeconfig secret
	configSecret := &corev1.Secret{}
	if err := client.Get(ctx, types.NamespacedName{Namespace: namespace.Name, Name: configSecretName}, configSecret); err != nil {
		t.Fatalf("Failed to get kubeconfig secret: %v", err)
	}

	// Parse the kubeconfig to get REST config with auth credentials
	rootShardRestConfig, err := clientcmd.RESTConfigFromKubeConfig(configSecret.Data["kubeconfig"])
	if err != nil {
		t.Fatalf("Failed to parse kubeconfig: %v", err)
	}

	// Set up port forwarding to the VirtualWorkspace service
	vwServiceName := namingScheme.VirtualWorkspaceServiceName(&vw)
	vwLocalPortStr, err := server.GetFreePort(t)
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}
	vwLocalPort, err := strconv.Atoi(vwLocalPortStr)
	if err != nil {
		t.Fatalf("Failed to parse port %q as number: %v", vwLocalPortStr, err)
	}
	utils.SelfDestuctingPortForward(t, ctx, namespace.Name, "svc/"+vwServiceName, 6443, vwLocalPort)

	// Create a new REST config pointing to the VirtualWorkspace service
	vwRestConfig := &rest.Config{
		Host: fmt.Sprintf("https://localhost:%d/services/apiexport/root/tenancy.kcp.io/clusters/*", vwLocalPort),
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   rootShardRestConfig.CAData,
			CertData: rootShardRestConfig.CertData,
			KeyData:  rootShardRestConfig.KeyData,
		},
	}

	// Create a client for the VirtualWorkspace
	vwClient, err := ctrlruntimeclient.New(vwRestConfig, ctrlruntimeclient.Options{Scheme: utils.NewScheme(t)})
	if err != nil {
		t.Fatalf("Failed to create VirtualWorkspace client: %v", err)
	}

	t.Log("Listing workspaces through VirtualWorkspace service...")
	workspaces := &kcptenancyv1alpha1.WorkspaceList{}
	if err := vwClient.List(ctx, workspaces); err != nil {
		t.Fatalf("Failed to list workspaces through VirtualWorkspace: %v", err)
	}

	found := false
	for _, ws := range workspaces.Items {
		t.Logf("Found workspace: %s (phase: %s)", ws.Name, ws.Status.Phase)
		if ws.Name == workspace.Name {
			found = true
		}
	}

	if !found {
		t.Fatalf("Created workspace %q not found in workspace list", workspace.Name)
	}

	t.Log("Successfully verified workspace is visible through external VW!")
}

// TestMultipleShardsWithExternalVirtualWorkspacesAndExtCache deploy a full kcp setup:
// root shard with external virtual workspace, shard1 with external virtual workspace, shard2
// with integrated VW, cache-server and front-proxy.
func TestMultipleShardsWithExternalVirtualWorkspacesAndExtCache(t *testing.T) {
	skipIfNotMainBranch(t)

	ctrlruntime.SetLogger(logr.Discard())

	client := utils.GetKubeClient(t)
	ctx := context.Background()
	namingScheme := naming.NewVersion1()

	namespace := utils.CreateSelfDestructingNamespace(t, ctx, client, "multi-shard-extcache-vw")

	// deploy the cache server
	cacheServer := utils.DeployCacheServer(ctx, t, client, namespace.Name)

	// Deploy external VirtualWorkspace for root shard
	rootVW := utils.DeployVirtualWorkspace(ctx, t, client, namingScheme, namespace.Name, "root-vw", false, func(vw *operatorv1alpha1.VirtualWorkspace) {
		vw.Spec.Target.RootShardRef = &corev1.LocalObjectReference{
			Name: "r00t",
		}
		vw.Spec.Resources = lowResourceRequirements()
	})

	// externalHostname must match whatever DeployFrontProxy chooses as the name for the FrontProxy
	externalHostname := utils.FrontProxyExternalHostname(namespace.Name, namingScheme)

	// Deploy root shard with external VW
	rootShard := utils.DeployRootShard(ctx, t, client, namingScheme, namespace.Name, externalHostname, func(rs *operatorv1alpha1.RootShard) {
		rs.Spec.KCPVirtualWorkspace = &corev1.LocalObjectReference{
			Name: rootVW.Name,
		}
		rs.Spec.Cache.Reference = &corev1.LocalObjectReference{
			Name: cacheServer.Name,
		}
		rs.Spec.Resources = lowResourceRequirements()
		if rs.Spec.Proxy != nil {
			rs.Spec.Proxy.Resources = lowResourceRequirements()
		}
	})

	// Deploy external VirtualWorkspace for regular shard1
	shard1VW := utils.DeployVirtualWorkspace(ctx, t, client, namingScheme, namespace.Name, "shard1-vw", false, func(vw *operatorv1alpha1.VirtualWorkspace) {
		vw.Spec.Target.ShardRef = &corev1.LocalObjectReference{
			Name: "shard1",
		}
		vw.Spec.Resources = lowResourceRequirements()
	})

	// Deploy regular shard1 with external VW
	utils.DeployShard(ctx, t, client, namespace.Name, "shard1", rootShard.Name, func(s *operatorv1alpha1.Shard) {
		s.Spec.KCPVirtualWorkspace = &corev1.LocalObjectReference{
			Name: shard1VW.Name,
		}
		s.Spec.Resources = lowResourceRequirements()
	})

	// Deploy another regular shard with internal VW
	utils.DeployShard(ctx, t, client, namespace.Name, "shard2", rootShard.Name, func(s *operatorv1alpha1.Shard) {
		s.Spec.Resources = lowResourceRequirements()
	})

	// Deploy front-proxy
	frontProxy := utils.DeployFrontProxy(ctx, t, client, namespace.Name, rootShard.Name, externalHostname, func(fp *operatorv1alpha1.FrontProxy) {
		fp.Spec.Resources = lowResourceRequirements()
	})

	// Wait for both VirtualWorkspace pods to be ready
	t.Log("Waiting for root VirtualWorkspace pods to be ready...")
	waitForVirtualWorkspacePods(t, ctx, client, namingScheme, &rootVW)

	t.Log("Waiting for shard VirtualWorkspace pods to be ready...")
	waitForVirtualWorkspacePods(t, ctx, client, namingScheme, &shard1VW)

	// verify the setup
	test := conformance.NewWorkspaceSchedulingTest(frontProxy.Name, client, namespace.Name)

	t.Log("Verifying workspace scheduling capabilities...")
	if err := test.Run(t, ctx, logicalcluster.NewPath("root")); err != nil {
		t.Fatalf("Workspace scheduling test failed: %v", err)
	}

	t.Log("Everything seems to check out fine.")
}

// waitForVirtualWorkspacePods waits for VirtualWorkspace pods to become ready.
func waitForVirtualWorkspacePods(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, names naming.Scheme, vw *operatorv1alpha1.VirtualWorkspace) {
	t.Helper()

	opts := []ctrlruntimeclient.ListOption{
		ctrlruntimeclient.InNamespace(vw.Namespace),
		ctrlruntimeclient.MatchingLabels(names.VirtualWorkspaceResourceLabels(vw)),
	}
	utils.WaitForPods(t, ctx, client, opts...)
}

// lowResourceRequirements returns minimal resource requirements for testing.
func lowResourceRequirements() *corev1.ResourceRequirements {
	return &corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("10m"),
			corev1.ResourceMemory: resource.MustParse("32Mi"),
		},
	}
}
