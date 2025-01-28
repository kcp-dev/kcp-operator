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

package controller

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
)

func deploymentReady(dep appsv1.Deployment) bool {
	return dep.Status.UpdatedReplicas == dep.Status.ReadyReplicas && dep.Status.ReadyReplicas == ptr.Deref(dep.Spec.Replicas, 0)
}

func deploymentStatusString(dep appsv1.Deployment, key types.NamespacedName) string {
	msg := fmt.Sprintf("Deployment %s", key)

	if dep.Name != "" {
		if deploymentReady(dep) {
			msg += " is fully up and running"
		} else {
			msg += " is not in desired replica state"
		}
	} else {
		msg += " does not exist"
	}

	return msg
}

func updateCondition(conditions []metav1.Condition, newCondition metav1.Condition) []metav1.Condition {
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
