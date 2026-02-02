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

	resources "github.com/kcp-dev/kcp-operator/internal/resources"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
	"github.com/kcp-dev/kcp-operator/test/utils"
)

func TestCreateShard(t *testing.T) {
	ctrlruntime.SetLogger(logr.Discard())

	client := utils.GetKubeClient(t)
	ctx := context.Background()

	// create namespace
	namespace := utils.CreateSelfDestructingNamespace(t, ctx, client, "create-shard")

	// deploy a root shard incl. etcd
	rootShard := utils.DeployRootShard(ctx, t, client, namespace.Name, "example.localhost")

	// deploy a 2nd shard incl. etcd
	shardName := "aadvark"
	utils.DeployShard(ctx, t, client, namespace.Name, shardName, rootShard.Name)

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
	rootShardClient := utils.ConnectWithKubeconfig(t, ctx, client, namespace.Name, rsConfig.Name, logicalcluster.None)

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
	if err := client.Create(ctx, &shardConfig); err != nil {
		t.Fatal(err)
	}
	utils.WaitForObject(t, ctx, client, &corev1.Secret{}, types.NamespacedName{Namespace: shardConfig.Namespace, Name: shardConfig.Spec.SecretRef.Name})

	t.Log("Connecting to Shard...")
	kcpClient := utils.ConnectWithKubeconfig(t, ctx, client, namespace.Name, shardConfig.Name, logicalcluster.None)

	// proof of life: list something every logicalcluster in kcp has
	t.Log("Should be able to list Secrets.")
	secrets := &corev1.SecretList{}
	if err := kcpClient.List(ctx, secrets); err != nil {
		t.Fatalf("Failed to list secrets in kcp: %v", err)
	}
}

func TestShardBundleAnnotation(t *testing.T) {
	ctrlruntime.SetLogger(logr.Discard())

	client := utils.GetKubeClient(t)
	ctx := context.Background()

	// create namespace
	namespace := utils.CreateSelfDestructingNamespace(t, ctx, client, "shard-bundle-annotation")

	// deploy a root shard incl. etcd
	rootShard := utils.DeployRootShard(ctx, t, client, namespace.Name, "example.localhost")

	// deploy a shard without bundle annotation first
	shardName := "annotated-shard"
	t.Log("Deploying Shard without bundle annotation...")
	shard := utils.DeployShard(ctx, t, client, namespace.Name, shardName, rootShard.Name)

	// verify no bundle exists yet
	bundleName := fmt.Sprintf("%s-bundle", shard.Name)
	bundle := &operatorv1alpha1.Bundle{}
	err := client.Get(ctx, types.NamespacedName{
		Namespace: namespace.Name,
		Name:      bundleName,
	}, bundle)
	if err == nil {
		t.Fatal("Bundle should not exist before annotation is added")
	}

	// add bundle annotation to the shard
	t.Log("Adding bundle annotation to Shard...")
	if err := client.Get(ctx, types.NamespacedName{
		Namespace: namespace.Name,
		Name:      shard.Name,
	}, &shard); err != nil {
		t.Fatal(err)
	}

	if shard.Annotations == nil {
		shard.Annotations = make(map[string]string)
	}
	shard.Annotations[resources.BundleAnnotation] = "true"

	if err := client.Update(ctx, &shard); err != nil {
		t.Fatalf("Failed to update shard with bundle annotation: %v", err)
	}

	// wait for bundle to be created
	t.Log("Waiting for Bundle to be created...")
	utils.WaitForObject(t, ctx, client, bundle, types.NamespacedName{
		Namespace: namespace.Name,
		Name:      bundleName,
	})

	// verify bundle was created
	if err := client.Get(ctx, types.NamespacedName{
		Namespace: namespace.Name,
		Name:      bundleName,
	}, bundle); err != nil {
		t.Fatalf("Failed to get bundle: %v", err)
	}

	// verify bundle has the correct target
	if bundle.Spec.Target.ShardRef == nil || bundle.Spec.Target.ShardRef.Name != shard.Name {
		t.Errorf("Bundle target should reference shard %s, got %+v", shard.Name, bundle.Spec.Target)
	}
	t.Log("Successfully verified Bundle was created with correct target")

	// wait for bundle to become ready and have all objects
	// Note: Shard without CABundleSecretRef has 17 objects (no merged CA bundle)
	expectedObjects := 17
	t.Logf("Waiting for Bundle to become Ready with all %d objects...", expectedObjects)
	timeout := time.After(3 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-timeout:
			t.Fatalf("Timeout waiting for bundle to become Ready. Current state: %s, objects: %d/%d, conditions: %+v",
				bundle.Status.State, len(bundle.Status.Objects), expectedObjects, bundle.Status.Conditions)
		case <-ticker.C:
			if err := client.Get(ctx, types.NamespacedName{
				Namespace: namespace.Name,
				Name:      bundleName,
			}, bundle); err != nil {
				t.Fatalf("Failed to get bundle: %v", err)
			}

			t.Logf("Bundle status: %s, objects: %d/%d", bundle.Status.State, len(bundle.Status.Objects), expectedObjects)

			// Count ready objects
			readyCount := 0
			for _, obj := range bundle.Status.Objects {
				if obj.State == operatorv1alpha1.BundleObjectStateReady {
					readyCount++
				} else {
					t.Logf("Object %s is not ready: %s", obj.Object, obj.Message)
				}
			}

			// Check if we have expected number of objects and all are ready
			if bundle.Status.State == operatorv1alpha1.BundleStateReady &&
				len(bundle.Status.Objects) == expectedObjects &&
				readyCount == expectedObjects {
				t.Logf("Bundle is Ready with all %d objects!", expectedObjects)
				goto bundleReady
			}

			// Log current status
			if len(bundle.Status.Objects) > 0 {
				t.Logf("Ready objects: %d/%d", readyCount, expectedObjects)
			}
		}
	}

bundleReady:
	// verify we have exactly the expected number of objects
	if len(bundle.Status.Objects) != expectedObjects {
		t.Errorf("Expected %d objects in bundle, got %d", expectedObjects, len(bundle.Status.Objects))
		for i, obj := range bundle.Status.Objects {
			t.Logf("Object %d: %s (state: %s)", i+1, obj.Object, obj.State)
		}
	}

	// verify all objects are ready
	for _, obj := range bundle.Status.Objects {
		if obj.State != operatorv1alpha1.BundleObjectStateReady {
			t.Errorf("Object %s is not ready: %s", obj.Object, obj.Message)
		}
	}

	// verify specific expected objects exist
	expectedObjectsList := []string{
		// CA certificates from RootShard (6 objects)
		fmt.Sprintf("secrets.core.v1:%s/%s-front-proxy-client-ca", namespace.Name, rootShard.Name),
		fmt.Sprintf("secrets.core.v1:%s/%s-requestheader-client-ca", namespace.Name, rootShard.Name),
		fmt.Sprintf("secrets.core.v1:%s/%s-server-ca", namespace.Name, rootShard.Name),
		fmt.Sprintf("secrets.core.v1:%s/%s-ca", namespace.Name, rootShard.Name),
		fmt.Sprintf("secrets.core.v1:%s/%s-client-ca", namespace.Name, rootShard.Name),
		fmt.Sprintf("secrets.core.v1:%s/%s-service-account-ca", namespace.Name, rootShard.Name),

		// Shard-specific certificates and secrets (9 objects)
		fmt.Sprintf("secrets.core.v1:%s/%s-logical-cluster-admin", namespace.Name, shard.Name),
		fmt.Sprintf("secrets.core.v1:%s/%s-logical-cluster-admin-kubeconfig", namespace.Name, shard.Name),
		fmt.Sprintf("secrets.core.v1:%s/%s-client-kubeconfig", namespace.Name, shard.Name),
		fmt.Sprintf("secrets.core.v1:%s/%s-external-logical-cluster-admin", namespace.Name, shard.Name),
		fmt.Sprintf("secrets.core.v1:%s/%s-external-logical-cluster-admin-kubeconfig", namespace.Name, shard.Name),
		fmt.Sprintf("secrets.core.v1:%s/%s-virtual-workspaces", namespace.Name, shard.Name),
		fmt.Sprintf("secrets.core.v1:%s/%s-client", namespace.Name, shard.Name),
		fmt.Sprintf("secrets.core.v1:%s/%s-server", namespace.Name, shard.Name),
		fmt.Sprintf("secrets.core.v1:%s/%s-service-account", namespace.Name, shard.Name),

		// Deployment (1 object)
		fmt.Sprintf("deployments.apps.v1:%s/%s-shard-kcp", namespace.Name, shard.Name),

		// Service (1 object)
		fmt.Sprintf("services.core.v1:%s/%s-shard-kcp", namespace.Name, shard.Name),
	}

	t.Log("Verifying all expected objects are present in bundle...")
	foundObjects := make(map[string]bool)
	for _, obj := range bundle.Status.Objects {
		foundObjects[obj.Object] = true
	}

	missingObjects := []string{}
	for _, expected := range expectedObjectsList {
		if !foundObjects[expected] {
			missingObjects = append(missingObjects, expected)
		}
	}

	if len(missingObjects) > 0 {
		t.Errorf("Missing expected objects in bundle:")
		for _, missing := range missingObjects {
			t.Errorf("  - %s", missing)
		}
		t.Log("Actual objects in bundle:")
		for _, obj := range bundle.Status.Objects {
			t.Logf("  - %s", obj.Object)
		}
	}

	t.Logf("Successfully verified Bundle has exactly %d objects, all in Ready state", expectedObjects)
}
