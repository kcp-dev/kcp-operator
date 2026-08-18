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

package shards

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/kcp-dev/logicalcluster/v3"
	kcpcorev1alpha1 "github.com/kcp-dev/sdk/apis/core/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntime "sigs.k8s.io/controller-runtime"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
	"github.com/kcp-dev/kcp-operator/test/utils"
)

func TestCreateShard(t *testing.T) {
	// TODO: Same limitation as TestProvisionFrontProxyRBAC. Deleting the Shard runs a
	// finalizer that deregisters it through the kcp API, which the config controllers
	// reach at <svc>.<ns>.svc.cluster.local - unresolvable from the config cluster once
	// kcp runs on the workload one.
	utils.SkipUnlessTopology(t, utils.TopologySingle)

	ctrlruntime.SetLogger(logr.Discard())

	configClient := utils.GetConfigKubeClient(t)
	workloadClient := utils.GetWorkloadKubeClient(t)
	ctx := context.Background()

	// create namespace
	namespace := utils.CreateSelfDestructingNamespace(t, ctx, configClient, "create-shard")

	// deploy a root shard incl. etcd
	rootShard := utils.DeployRootShard(ctx, t, configClient, workloadClient, namespace.Name, "")

	// deploy a 2nd shard incl. etcd
	shardName := "aadvark"
	utils.DeployShard(ctx, t, configClient, workloadClient, namespace.Name, shardName, rootShard.Name)

	// create a kubeconfig to access the root shard
	configSecretName := fmt.Sprintf("%s-shard-kubeconfig", rootShard.Name)

	rsConfig := operatorv1alpha1.Kubeconfig{}
	rsConfig.Name = configSecretName
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
		Groups: []string{"system:masters"},
	}

	t.Log("Creating kubeconfig for RootShard...")
	if err := configClient.Create(ctx, &rsConfig); err != nil {
		t.Fatal(err)
	}
	utils.WaitForObject(t, ctx, configClient, &corev1.Secret{}, types.NamespacedName{Namespace: rsConfig.Namespace, Name: rsConfig.Spec.SecretRef.Name})

	t.Log("Connecting to RootShard...")
	rootShardClient := utils.ConnectWithKubeconfig(t, ctx, configClient, namespace.Name, rsConfig.Name, logicalcluster.None)

	// wait until the 2nd shard has registered itself successfully at the root shard
	shardKey := types.NamespacedName{Name: shardName}
	t.Log("Waiting for Shard to register itself on the RootShard...")
	utils.WaitForObject(t, ctx, rootShardClient, &kcpcorev1alpha1.Shard{}, shardKey)

	// create a kubeconfig to access the shard
	configSecretName = fmt.Sprintf("%s-shard-kubeconfig", shardName)

	shardConfig := operatorv1alpha1.Kubeconfig{}
	shardConfig.Name = configSecretName
	shardConfig.Namespace = namespace.Name

	shardConfig.Spec = operatorv1alpha1.KubeconfigSpec{
		Target: operatorv1alpha1.KubeconfigTarget{
			ShardRef: &corev1.LocalObjectReference{
				Name: shardName,
			},
		},
		Username: "e2e",
		Validity: metav1.Duration{Duration: 2 * time.Hour},
		SecretRef: corev1.LocalObjectReference{
			Name: configSecretName,
		},
		Groups: []string{"system:masters"},
	}

	t.Log("Creating kubeconfig for Shard...")
	if err := configClient.Create(ctx, &shardConfig); err != nil {
		t.Fatal(err)
	}
	utils.WaitForObject(t, ctx, configClient, &corev1.Secret{}, types.NamespacedName{Namespace: shardConfig.Namespace, Name: shardConfig.Spec.SecretRef.Name})

	t.Log("Connecting to Shard...")
	kcpClient := utils.ConnectWithKubeconfig(t, ctx, configClient, namespace.Name, shardConfig.Name, logicalcluster.None)

	// proof of life: list something every logicalcluster in kcp has
	t.Log("Should be able to list Secrets.")
	secrets := &corev1.SecretList{}
	if err := kcpClient.List(ctx, secrets); err != nil {
		t.Fatalf("Failed to list secrets in kcp: %v", err)
	}

	// Test cleanup: delete the operator's Shard CR and verify the kcp Shard object is cleaned up
	t.Log("Deleting operator Shard CR...")
	shard := &operatorv1alpha1.Shard{}
	if err := configClient.Get(ctx, types.NamespacedName{Namespace: namespace.Name, Name: shardName}, shard); err != nil {
		t.Fatalf("Failed to get Shard: %v", err)
	}
	t.Logf("Shard finalizers before delete: %v", shard.Finalizers)
	if err := configClient.Delete(ctx, shard); err != nil {
		t.Fatalf("Failed to delete Shard: %v", err)
	}

	// Wait for the kcp Shard object to be deleted from the root shard.
	// This proves the operator's cleanup finalizer ran successfully.
	t.Log("Waiting for kcp Shard to be deleted from root shard...")
	utils.WaitForObjectDeletion(t, ctx, rootShardClient, &kcpcorev1alpha1.Shard{}, shardKey)
	t.Log("kcp Shard has been deleted from root shard, cleanup was successful.")
}
