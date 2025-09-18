//go:build e2e

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

package kubeconfigrbac

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"

	kcptenancyv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/tenancy/v1alpha1"
	"github.com/kcp-dev/logicalcluster/v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
	"github.com/kcp-dev/kcp-operator/test/utils"
)

func TestProvisionFrontProxyRBAC(t *testing.T) {
	ctrlruntime.SetLogger(logr.Discard())

	client := utils.GetKubeClient(t)
	ctx := context.Background()

	rootCluster := logicalcluster.NewPath("root")
	namespace := utils.CreateSelfDestructingNamespace(t, ctx, client, "provision-frontproxy-rbac")
	externalHostname := fmt.Sprintf("front-proxy-front-proxy.%s.svc.cluster.local", namespace.Name)

	// deploy rootshard
	rootShard := utils.DeployRootShard(ctx, t, client, namespace.Name, externalHostname)

	// deploy front-proxy
	frontProxy := utils.DeployFrontProxy(ctx, t, client, namespace.Name, rootShard.Name, externalHostname)

	// create a dummy workspace where we later want to provision RBAC in
	t.Log("Creating dummy workspace…")
	workspace := &kcptenancyv1alpha1.Workspace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: kcptenancyv1alpha1.WorkspaceSpec{
			Type: kcptenancyv1alpha1.WorkspaceTypeReference{
				Name: "universal",
			},
		},
	}

	dummyCluster := rootCluster.Join(workspace.Name)
	proxyClient := utils.ConnectWithRootShardProxy(t, ctx, client, &rootShard, rootCluster)
	if err := proxyClient.Create(ctx, workspace); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// wait for workspace to be ready
	t.Log("Waiting for workspace to be ready…")
	dummyClient := utils.ConnectWithRootShardProxy(t, ctx, client, &rootShard, dummyCluster)

	err := wait.PollUntilContextTimeout(ctx, 500*time.Millisecond, 30*time.Second, false, func(ctx context.Context) (done bool, err error) {
		return dummyClient.List(ctx, &corev1.SecretList{}) == nil, nil
	})
	if err != nil {
		t.Fatalf("Failed to wait for workspace to become available: %v", err)
	}

	// create my-config kubeconfig
	configSecretName := "kubeconfig-my-config-e2e"

	// as of now, this Kubeconfig will not grant any permissions yet
	fpConfig := operatorv1alpha1.Kubeconfig{}
	fpConfig.Name = "my-config"
	fpConfig.Namespace = namespace.Name
	fpConfig.Spec = operatorv1alpha1.KubeconfigSpec{
		Target: operatorv1alpha1.KubeconfigTarget{
			FrontProxyRef: &corev1.LocalObjectReference{
				Name: frontProxy.Name,
			},
		},
		Username: "e2e",
		Validity: metav1.Duration{Duration: 2 * time.Hour},
		SecretRef: corev1.LocalObjectReference{
			Name: configSecretName,
		},
	}

	t.Log("Creating kubeconfig with no permissions attached…")
	if err := client.Create(ctx, &fpConfig); err != nil {
		t.Fatal(err)
	}
	utils.WaitForObject(t, ctx, client, &corev1.Secret{}, types.NamespacedName{Namespace: fpConfig.Namespace, Name: fpConfig.Spec.SecretRef.Name})

	t.Log("Connecting to FrontProxy…")
	kcpClient := utils.ConnectWithKubeconfig(t, ctx, client, namespace.Name, fpConfig.Name, dummyCluster)

	// This should not work yet.
	t.Logf("Should not be able to list Secrets in %v.", dummyCluster)
	if err := kcpClient.List(ctx, &corev1.SecretList{}); err == nil {
		t.Fatal("Should not have been able to list Secrets, but was. Where have my permissions come from?")
	}

	// Now we extend the Kubeconfig with additional permissions.
	fpConfig.Spec.Authorization = &operatorv1alpha1.KubeconfigAuthorization{
		ClusterRoleBindings: operatorv1alpha1.KubeconfigClusterRoleBindings{
			WorkspacePath: dummyCluster.String(),
			ClusterRoles:  []string{"cluster-admin"},
		},
	}

	t.Log("Updating kubeconfig with permissions attached…")
	if err := client.Update(ctx, &fpConfig); err != nil {
		t.Fatal(err)
	}

	t.Logf("Should now be able to list Secrets in %v.", dummyCluster)
	err = wait.PollUntilContextTimeout(ctx, 500*time.Millisecond, 30*time.Second, false, func(ctx context.Context) (done bool, err error) {
		return kcpClient.List(ctx, &corev1.SecretList{}) == nil, nil
	})
	if err != nil {
		t.Fatalf("Failed to list Secrets in dummy workspace: %v", err)
	}

	// And now we remove the permissions again.
	t.Log("Updating kubeconfig to remove the attached permissions…")
	if err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(&fpConfig), &fpConfig); err != nil {
		t.Fatal(err)
	}

	fpConfig.Spec.Authorization = nil

	if err := client.Update(ctx, &fpConfig); err != nil {
		t.Fatal(err)
	}

	t.Logf("Should no longer be able to list Secrets in %v.", dummyCluster)
	err = wait.PollUntilContextTimeout(ctx, 500*time.Millisecond, 30*time.Second, false, func(ctx context.Context) (done bool, err error) {
		return kcpClient.List(ctx, &corev1.SecretList{}) != nil, nil
	})
	if err != nil {
		t.Fatalf("Failed to wait for permissions to be gone: %v", err)
	}
}
