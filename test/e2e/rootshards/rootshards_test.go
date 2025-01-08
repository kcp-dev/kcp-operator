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
	"testing"
	"time"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	kcpoperatorv1alpha1 "github.com/kcp-dev/kcp-operator/api/v1alpha1"
	"github.com/kcp-dev/kcp-operator/test/utils"
)

func TestCreateRootShard(t *testing.T) {
	ctrlruntime.SetLogger(logr.Discard())

	client := utils.GetKubeClient(t)
	ctx := context.Background()
	namespace := "create-rootshard"

	utils.CreateSelfDestructingNamespace(t, ctx, client, namespace)
	etcd := utils.DeployEtcd(t, namespace)

	rootShard := kcpoperatorv1alpha1.RootShard{}
	rootShard.Name = "test"
	rootShard.Namespace = namespace

	rootShard.Spec = kcpoperatorv1alpha1.RootShardSpec{
		External: kcpoperatorv1alpha1.ExternalConfig{
			Hostname: "example.localhost",
			Port:     6443,
		},
		Certificates: kcpoperatorv1alpha1.Certificates{
			IssuerRef: utils.GetSelfSignedIssuerRef(),
		},
		Cache: kcpoperatorv1alpha1.CacheConfig{
			Embedded: &kcpoperatorv1alpha1.EmbeddedCacheConfiguration{
				Enabled: true,
			},
		},
		CommonShardSpec: kcpoperatorv1alpha1.CommonShardSpec{
			Etcd: kcpoperatorv1alpha1.EtcdConfig{
				Endpoints: []string{etcd},
			},
		},
	}

	t.Logf("Creating RootShard %s…", rootShard.Name)
	if err := client.Create(ctx, &rootShard); err != nil {
		t.Fatal(err)
	}
	waitForRootShardPods(t, ctx, client, &rootShard)

	configSecretName := "kubeconfig"

	rsConfig := kcpoperatorv1alpha1.Kubeconfig{}
	rsConfig.Name = "test"
	rsConfig.Namespace = namespace

	rsConfig.Spec = kcpoperatorv1alpha1.KubeconfigSpec{
		Target: kcpoperatorv1alpha1.KubeconfigTarget{
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

	t.Log("Creating kubeconfig for RootShard…")
	if err := client.Create(ctx, &rsConfig); err != nil {
		t.Fatal(err)
	}
	waitForObject(t, ctx, client, &corev1.Secret{}, types.NamespacedName{Namespace: rsConfig.Namespace, Name: rsConfig.Spec.SecretRef.Name})

	t.Log("Connecting to RootShard…")
	kcpClient := utils.ConnectWithKubeconfig(t, ctx, client, namespace, rsConfig.Name)

	// proof of life: list something every logicalcluster in kcp has
	t.Log("Should be able to list Secrets.")
	secrets := &corev1.SecretList{}
	if err := kcpClient.List(ctx, secrets); err != nil {
		t.Fatalf("Failed to list secrets in kcp: %v", err)
	}
}

func waitForRootShardPods(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, rootShard *kcpoperatorv1alpha1.RootShard) {
	t.Helper()

	opts := []ctrlruntimeclient.ListOption{
		ctrlruntimeclient.InNamespace(rootShard.Namespace),
		ctrlruntimeclient.MatchingLabels{
			"app.kubernetes.io/component": "rootshard",
			"app.kubernetes.io/instance":  rootShard.Name,
		},
	}

	t.Log("Waiting for RootShard to be available…")

	err := wait.PollUntilContextTimeout(ctx, 500*time.Millisecond, 3*time.Minute, false, func(ctx context.Context) (done bool, err error) {
		pods := corev1.PodList{}
		if err := client.List(ctx, &pods, opts...); err != nil {
			return false, err
		}

		if len(pods.Items) == 0 {
			return false, nil
		}

		for _, pod := range pods.Items {
			if !podIsReady(pod) {
				return false, nil
			}
		}

		return true, nil
	})
	if err != nil {
		t.Fatalf("Failed to wait for RootShard to become available: %v", err)
	}

	t.Log("RootShard is ready.")
}

func waitForObject(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, obj ctrlruntimeclient.Object, key types.NamespacedName) {
	t.Helper()
	t.Logf("Waiting for %T to be available…", obj)

	err := wait.PollUntilContextTimeout(ctx, 500*time.Millisecond, 3*time.Minute, false, func(ctx context.Context) (done bool, err error) {
		err = client.Get(ctx, key, obj)
		return err == nil, nil
	})
	if err != nil {
		t.Fatalf("Failed to wait for %T to become available: %v", obj, err)
	}

	t.Logf("%T is ready.", obj)
}

func podIsReady(pod corev1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady {
			return cond.Status == corev1.ConditionTrue
		}
	}

	return false
}
