/*
Copyright 2024 The KCP Authors.

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

package utils

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func WaitForPods(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, listOpts ...ctrlruntimeclient.ListOption) {
	t.Helper()

	t.Log("Waiting for pods to be available…")

	err := wait.PollUntilContextTimeout(ctx, 500*time.Millisecond, 3*time.Minute, false, func(ctx context.Context) (done bool, err error) {
		pods := corev1.PodList{}
		if err := client.List(ctx, &pods, listOpts...); err != nil {
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
		t.Fatalf("Failed to wait for pods to become available: %v", err)
	}

	t.Log("Pods are ready.")
}

func podIsReady(pod corev1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady {
			return cond.Status == corev1.ConditionTrue
		}
	}

	return false
}

func WaitForObject(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, obj ctrlruntimeclient.Object, key types.NamespacedName) {
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
