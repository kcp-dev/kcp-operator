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

package util

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func deploymentReady(dep appsv1.Deployment) bool {
	return dep.Status.UpdatedReplicas == dep.Status.ReadyReplicas && dep.Status.ReadyReplicas == ptr.Deref(dep.Spec.Replicas, 0)
}

func GetDeploymentAvailableCondition(ctx context.Context, client ctrlruntimeclient.Client, key types.NamespacedName) (metav1.Condition, error) {
	var dep appsv1.Deployment
	if err := client.Get(ctx, key, &dep); ctrlruntimeclient.IgnoreNotFound(err) != nil {
		return metav1.Condition{}, err
	}

	available := metav1.ConditionFalse
	reason := operatorv1alpha1.ConditionReasonDeploymentUnavailable
	msg := fmt.Sprintf("Deployment %s", key)

	if dep.Name != "" {
		if deploymentReady(dep) {
			available = metav1.ConditionTrue
			reason = operatorv1alpha1.ConditionReasonReplicasUp
			msg += " is fully up and running."
		} else {
			available = metav1.ConditionFalse
			reason = operatorv1alpha1.ConditionReasonReplicasUnavailable
			msg += " is not in desired replica state."
		}
	} else {
		msg += " does not exist."
	}

	return metav1.Condition{
		Type:    string(operatorv1alpha1.ConditionTypeAvailable),
		Status:  available,
		Reason:  string(reason),
		Message: msg,
	}, nil
}

func UpdateCondition(conditions []metav1.Condition, newCondition metav1.Condition) []metav1.Condition {
	if conditions == nil {
		conditions = make([]metav1.Condition, 0)
	}

	cond := apimeta.FindStatusCondition(conditions, newCondition.Type)

	if cond == nil || cond.ObservedGeneration != newCondition.ObservedGeneration || cond.Status != newCondition.Status {
		transitionTime := metav1.Now()
		if cond != nil && cond.Status == newCondition.Status {
			// We only need to set LastTransitionTime if we are actually toggling the status
			// or if no transition time was set.
			transitionTime = cond.LastTransitionTime
		}

		newCondition.LastTransitionTime = transitionTime

		apimeta.SetStatusCondition(&conditions, newCondition)
	}

	return conditions
}

func FetchRootShard(ctx context.Context, client ctrlruntimeclient.Client, namespace string, ref *corev1.LocalObjectReference) (metav1.Condition, *operatorv1alpha1.RootShard) {
	if ref == nil {
		return metav1.Condition{
			Type:    string(operatorv1alpha1.ConditionTypeRootShard),
			Status:  metav1.ConditionFalse,
			Reason:  string(operatorv1alpha1.ConditionReasonRootShardRefInvalid),
			Message: "No valid RootShard defined in spec.",
		}, nil
	}

	rootShard := &operatorv1alpha1.RootShard{}
	if err := client.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: namespace}, rootShard); err != nil {
		return metav1.Condition{
			Type:    string(operatorv1alpha1.ConditionTypeRootShard),
			Status:  metav1.ConditionFalse,
			Reason:  string(operatorv1alpha1.ConditionReasonRootShardRefInvalid),
			Message: fmt.Sprintf("Failed to retrieve RootShard: %v.", err),
		}, nil
	}

	return metav1.Condition{
		Type:    string(operatorv1alpha1.ConditionTypeRootShard),
		Status:  metav1.ConditionTrue,
		Reason:  string(operatorv1alpha1.ConditionReasonRootShardRefValid),
		Message: "RootShard reference is valid.",
	}, rootShard
}

// GetRootShardChildren returns all shards that are currently registered with the given root shard.
func GetRootShardChildren(ctx context.Context, client ctrlruntimeclient.Client, rootShard *operatorv1alpha1.RootShard) ([]operatorv1alpha1.Shard, error) {
	var errs []error

	var shards operatorv1alpha1.ShardList
	err := client.List(ctx, &shards, &ctrlruntimeclient.ListOptions{
		Namespace: rootShard.Namespace,
	})
	if err != nil {
		errs = append(errs, err)
	}

	var result []operatorv1alpha1.Shard
	for _, shard := range shards.Items {
		if shard.Spec.RootShard.Reference.Name == rootShard.Name {
			result = append(result, shard)
		}
	}

	return result, kerrors.NewAggregate(errs)
}
