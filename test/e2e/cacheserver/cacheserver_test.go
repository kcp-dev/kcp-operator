//go:build e2e

/*
Copyright 2026 The KCP Authors.

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

package cacheserver

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/kcp-dev/logicalcluster/v3"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntime "sigs.k8s.io/controller-runtime"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
	"github.com/kcp-dev/kcp-operator/test/utils"
)

func TestCacheWithRootShard(t *testing.T) {
	ctrlruntime.SetLogger(logr.Discard())

	client := utils.GetKubeClient(t)
	ctx := context.Background()

	// create namespace
	namespace := utils.CreateSelfDestructingNamespace(t, ctx, client, "cache-rootshard")

	// deploy the cache server
	cacheServer := utils.DeployCacheServer(ctx, t, client, namespace.Name)

	// deploy a root shard that uses our cache
	rootShard := utils.DeployRootShard(ctx, t, client, namespace.Name, "example.localhost", func(rs *operatorv1alpha1.RootShard) {
		rs.Spec.Cache.Reference = &corev1.LocalObjectReference{
			Name: cacheServer.Name,
		}
	})

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
	if err := client.Create(ctx, &rsConfig); err != nil {
		t.Fatal(err)
	}
	utils.WaitForObject(t, ctx, client, &corev1.Secret{}, types.NamespacedName{Namespace: rsConfig.Namespace, Name: rsConfig.Spec.SecretRef.Name})

	t.Log("Connecting to RootShard...")
	rootShardClient := utils.ConnectWithKubeconfig(t, ctx, client, namespace.Name, rsConfig.Name, logicalcluster.NewPath("root"))

	// proof of life: list something every logicalcluster in kcp has
	t.Log("Should be able to list Secrets.")
	secrets := &corev1.SecretList{}
	if err := rootShardClient.List(ctx, secrets); err != nil {
		t.Fatalf("Failed to list secrets in kcp: %v", err)
	}
}

func TestCacheWithMultipleExplicitShards(t *testing.T) {
	ctrlruntime.SetLogger(logr.Discard())

	client := utils.GetKubeClient(t)
	ctx := context.Background()

	// create namespace
	namespace := utils.CreateSelfDestructingNamespace(t, ctx, client, "cache-sharded-explicit")

	// deploy the cache server
	cacheServer := utils.DeployCacheServer(ctx, t, client, namespace.Name)

	// deploy a root shard that uses our cache
	rootShard := utils.DeployRootShard(ctx, t, client, namespace.Name, "example.localhost", func(rs *operatorv1alpha1.RootShard) {
		rs.Spec.Cache.Reference = &corev1.LocalObjectReference{
			Name: cacheServer.Name,
		}
	})

	// deploy a 2nd shard incl. etcd
	shardName := "aadvark"
	utils.DeployShard(ctx, t, client, namespace.Name, shardName, rootShard.Name, func(s *operatorv1alpha1.Shard) {
		s.Spec.Cache = &operatorv1alpha1.ShardCacheConfig{
			Reference: &corev1.LocalObjectReference{
				Name: cacheServer.Name,
			},
		}
	})

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
	if err := client.Create(ctx, &rsConfig); err != nil {
		t.Fatal(err)
	}
	utils.WaitForObject(t, ctx, client, &corev1.Secret{}, types.NamespacedName{Namespace: rsConfig.Namespace, Name: rsConfig.Spec.SecretRef.Name})

	t.Log("Connecting to RootShard...")
	rootShardClient := utils.ConnectWithKubeconfig(t, ctx, client, namespace.Name, rsConfig.Name, logicalcluster.NewPath("root"))

	// proof of life: list something every logicalcluster in kcp has
	t.Log("Should be able to list Secrets.")
	secrets := &corev1.SecretList{}
	if err := rootShardClient.List(ctx, secrets); err != nil {
		t.Fatalf("Failed to list secrets in kcp: %v", err)
	}
}
