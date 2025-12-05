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

package metrics

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	RootShardResourceType   = "rootshard"
	ShardResourceType       = "shard"
	FrontProxyResourceType  = "frontproxy"
	CacheServerResourceType = "cacheserver"
	KubeconfigResourceType  = "kubeconfig"
)

// RecordObjectMetrics records metrics for a Kubernetes object with phase and conditions
func RecordObjectMetrics(resourceType, resourceName, namespace, phase string, conditions []metav1.Condition) {
	if phase == "" {
		phase = "Unknown"
	}

	switch resourceType {
	case RootShardResourceType:
		RootShardCount.WithLabelValues(phase, namespace).Set(1)
	case ShardResourceType:
		ShardCount.WithLabelValues(phase, namespace).Set(1)
	case FrontProxyResourceType:
		FrontProxyCount.WithLabelValues(phase, namespace).Set(1)
	case CacheServerResourceType:
		CacheServerCount.WithLabelValues(namespace).Set(1)
	case KubeconfigResourceType:
		KubeconfigCount.WithLabelValues(namespace).Set(1)
	}

	for _, condition := range conditions {
		status := 0.0
		switch condition.Status {
		case metav1.ConditionTrue:
			status = 1.0
		case metav1.ConditionFalse:
			status = 0.0
		case metav1.ConditionUnknown:
			status = -1.0
		}

		ConditionStatus.WithLabelValues(
			resourceType,
			resourceName,
			namespace,
			condition.Type,
		).Set(status)
	}
}

// RecordReconciliationMetrics records reconciliation duration and error metrics
func RecordReconciliationMetrics(controller string, duration float64, err error) {
	result := "success"
	if err != nil {
		result = "error"
		ReconciliationErrors.WithLabelValues(controller, "reconcile_error").Inc()
	}
	ReconciliationDuration.WithLabelValues(controller, result).Observe(duration)
}

// RecordReconciliationError records a specific reconciliation error
func RecordReconciliationError(controller, errorType string) {
	ReconciliationErrors.WithLabelValues(controller, errorType).Inc()
}
