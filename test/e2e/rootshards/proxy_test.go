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

package rootshards

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"
	kcpcorev1alpha1 "github.com/kcp-dev/kcp/sdk/apis/core/v1alpha1"
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

func TestRootShardProxy(t *testing.T) {
	ctrlruntime.SetLogger(logr.Discard())

	client := utils.GetKubeClient(t)
	ctx := context.Background()
	namespaceSuffix := "rootshard-proxy"

	namespace := utils.CreateSelfDestructingNamespace(t, ctx, client, namespaceSuffix)
	externalHostname := fmt.Sprintf("front-proxy-front-proxy.e2e-%s.svc.cluster.local", namespaceSuffix)

	// deploy a root shard incl. etcd
	rootShard := utils.DeployRootShard(ctx, t, client, namespace.Name, externalHostname)

	// deploy a 2nd shard incl. etcd
	shardName := "aadvark"
	utils.DeployShard(ctx, t, client, namespace.Name, shardName, rootShard.Name)

	// deploy front-proxy
	utils.DeployFrontProxy(ctx, t, client, namespace.Name, rootShard.Name, externalHostname)

	configSecretName := "kubeconfig"

	rsConfig := operatorv1alpha1.Kubeconfig{}
	rsConfig.Name = "test"
	rsConfig.Namespace = namespace.Name

	rsConfig.Spec = operatorv1alpha1.KubeconfigSpec{
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
	}

	t.Log("Creating kubeconfig for RootShard…")
	if err := client.Create(ctx, &rsConfig); err != nil {
		t.Fatal(err)
	}
	utils.WaitForObject(t, ctx, client, &corev1.Secret{}, types.NamespacedName{Namespace: rsConfig.Namespace, Name: rsConfig.Spec.SecretRef.Name})

	t.Log("Connecting to RootShard…")
	rootShardClient := utils.ConnectWithKubeconfig(t, ctx, client, namespace.Name, rsConfig.Name)

	// wait until the 2nd shard has registered itself successfully at the root shard
	shardKey := types.NamespacedName{Name: shardName}
	t.Log("Waiting for Shard to register itself on the RootShard…")
	utils.WaitForObject(t, ctx, rootShardClient, &kcpcorev1alpha1.Shard{}, shardKey)

	// create workspace that we want to have scheduled onto the 2nd shard
	t.Log("Creating workspace with its logicalcluster on the 2nd Shard…")
	workspace := &kcptenancyv1alpha1.Workspace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: kcptenancyv1alpha1.WorkspaceSpec{
			Type: kcptenancyv1alpha1.WorkspaceTypeReference{
				Name: "universal",
			},
			Location: &kcptenancyv1alpha1.WorkspaceLocation{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"name": shardName,
					},
				},
			},
		},
	}
	if err := rootShardClient.Create(ctx, workspace); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	err := wait.PollUntilContextTimeout(ctx, 500*time.Millisecond, 30*time.Second, false, func(ctx context.Context) (done bool, err error) {
		err = rootShardClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(workspace), workspace)
		if err != nil {
			return false, err
		}

		return workspace.Status.Phase == kcpcorev1alpha1.LogicalClusterPhaseReady, nil
	})
	if err != nil {
		t.Fatalf("Failed to wait for workspace to become ready: %v", err)
	}

	// build a client through the proxy to the new workspace
	proxyClient := utils.ConnectWithRootShardProxy(t, ctx, client, &rootShard, logicalcluster.NewPath("root").Join(workspace.Name))
	if err != nil {
		t.Fatalf("Failed to create root shard proxy client: %v", err)
	}

	// proof of life: list something every logicalcluster in kcp has
	t.Log("Should be able to list Secrets in the new workspace.")
	secrets := &corev1.SecretList{}
	if err := proxyClient.List(ctx, secrets); err != nil {
		t.Fatalf("Failed to list secrets in workspace: %v", err)
	}
}
