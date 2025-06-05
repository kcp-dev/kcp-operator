//go:build kcpe2e

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

package kcpe2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-logr/logr"
	kcpcorev1alpha1 "github.com/kcp-dev/kcp/sdk/apis/core/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntime "sigs.k8s.io/controller-runtime"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
	"github.com/kcp-dev/kcp-operator/test/utils"
)

func TestKcpTestSuite(t *testing.T) {
	const (
		namespace        = "kcp-e2e"
		externalHostname = "example.localhost"
	)

	testImage := os.Getenv("KCP_E2E_TEST_IMAGE")
	if testImage == "" {
		t.Skip("No $KCP_E2E_TEST_IMAGE defined.")
	}

	ctrlruntime.SetLogger(logr.Discard())

	client := utils.GetKubeClient(t)
	ctx := context.Background()

	// create namspace
	utils.CreateSelfDestructingNamespace(t, ctx, client, namespace)

	// deploy a root shard incl. etcd
	rootShard := utils.DeployRootShard(ctx, t, client, namespace, externalHostname)

	// deploy a 2nd shard incl. etcd
	shardName := "aadvark"
	utils.DeployShard(ctx, t, client, namespace, shardName, rootShard.Name)

	// deploy front-proxy
	utils.DeployFrontProxy(ctx, t, client, namespace, rootShard.Name, externalHostname)

	// create a kubeconfig to access the root shard
	rsConfigSecretName := fmt.Sprintf("%s-shard-kubeconfig", rootShard.Name)

	rsConfig := operatorv1alpha1.Kubeconfig{}
	rsConfig.Name = rsConfigSecretName
	rsConfig.Namespace = namespace

	rsConfig.Spec = operatorv1alpha1.KubeconfigSpec{
		Target: operatorv1alpha1.KubeconfigTarget{
			RootShardRef: &corev1.LocalObjectReference{
				Name: rootShard.Name,
			},
		},
		Username: "e2e",
		Validity: metav1.Duration{Duration: 2 * time.Hour},
		SecretRef: corev1.LocalObjectReference{
			Name: rsConfigSecretName,
		},
		Groups: []string{"system:masters"},
	}

	t.Log("Creating kubeconfig for RootShard…")
	if err := client.Create(ctx, &rsConfig); err != nil {
		t.Fatal(err)
	}
	utils.WaitForObject(t, ctx, client, &corev1.Secret{}, types.NamespacedName{Namespace: rsConfig.Namespace, Name: rsConfig.Spec.SecretRef.Name})

	t.Log("Connecting to RootShard…")
	rootShardClient := utils.ConnectWithKubeconfig(t, ctx, client, namespace, rsConfig.Name)

	// wait until the 2nd shard has registered itself successfully at the root shard
	shardKey := types.NamespacedName{Name: shardName}
	t.Log("Waiting for Shard to register itself on the RootShard…")
	utils.WaitForObject(t, ctx, rootShardClient, &kcpcorev1alpha1.Shard{}, shardKey)

	// create a kubeconfig to access the shard
	shardConfigSecretName := fmt.Sprintf("%s-shard-kubeconfig", shardName)

	shardConfig := operatorv1alpha1.Kubeconfig{}
	shardConfig.Name = shardConfigSecretName
	shardConfig.Namespace = namespace

	shardConfig.Spec = operatorv1alpha1.KubeconfigSpec{
		Target: operatorv1alpha1.KubeconfigTarget{
			ShardRef: &corev1.LocalObjectReference{
				Name: shardName,
			},
		},
		Username: "e2e",
		Validity: metav1.Duration{Duration: 2 * time.Hour},
		SecretRef: corev1.LocalObjectReference{
			Name: shardConfigSecretName,
		},
		Groups: []string{"system:masters"},
	}

	t.Log("Creating kubeconfig for Shard…")
	if err := client.Create(ctx, &shardConfig); err != nil {
		t.Fatal(err)
	}
	utils.WaitForObject(t, ctx, client, &corev1.Secret{}, types.NamespacedName{Namespace: shardConfig.Namespace, Name: shardConfig.Spec.SecretRef.Name})

	t.Log("Connecting to Shard…")
	kcpClient := utils.ConnectWithKubeconfig(t, ctx, client, namespace, shardConfig.Name)

	// proof of life: list something every logicalcluster in kcp has
	t.Log("Should be able to list Secrets.")
	secrets := &corev1.SecretList{}
	if err := kcpClient.List(ctx, secrets); err != nil {
		t.Fatalf("Failed to list secrets in kcp: %v", err)
	}

	// deploy kcp e2e test container into the cluster
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    namespace,
			GenerateName: "kcp-e2e-",
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{{
				Name:            "e2e",
				Image:           testImage,
				ImagePullPolicy: corev1.PullNever,
				Env: []corev1.EnvVar{{
					Name:  "KUBECONFIG",
					Value: "/opt/rootshard-kubeconfig/kubeconfig",
				}},
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "rootshard-kubeconfig",
					ReadOnly:  true,
					MountPath: "/opt/rootshard-kubeconfig",
				}},
			}},
			Volumes: []corev1.Volume{{
				Name: "rootshard-kubeconfig",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: rsConfigSecretName,
					},
				},
			}},
		},
	}

	t.Log("Creating kcp e2e test pod…")
	if err := client.Create(ctx, testPod); err != nil {
		t.Fatal(err)
	}

	t.Log("Sleeping for 10 minutes...")
	time.Sleep(10 * time.Minute)
}
