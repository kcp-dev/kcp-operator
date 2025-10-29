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
	"github.com/prometheus/client_golang/prometheus"

	ctrlruntimemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// RootShardCount tracks the number of RootShard objects by their current phase.
	// Labels: phase (Provisioning|Running|Deleting), namespace
	RootShardCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kcp_operator_rootshard_count",
			Help: "Number of RootShard objects by phase",
		},
		[]string{"phase", "namespace"},
	)

	// ShardCount tracks the number of Shard objects by their current phase.
	// Labels: phase (Provisioning|Running|Deleting), namespace
	ShardCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kcp_operator_shard_count",
			Help: "Number of Shard objects by phase",
		},
		[]string{"phase", "namespace"},
	)

	// FrontProxyCount tracks the number of FrontProxy objects by their current phase.
	// Labels: phase (Provisioning|Running|Deleting), namespace
	FrontProxyCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kcp_operator_frontproxy_count",
			Help: "Number of FrontProxy objects by phase",
		},
		[]string{"phase", "namespace"},
	)

	// CacheServerCount tracks the number of CacheServer objects by namespace.
	// Labels: namespace
	CacheServerCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kcp_operator_cacheserver_count",
			Help: "Number of CacheServer objects by namespace",
		},
		[]string{"namespace"},
	)

	// KubeconfigCount tracks the number of Kubeconfig objects by namespace.
	// Labels: namespace
	KubeconfigCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kcp_operator_kubeconfig_count",
			Help: "Number of Kubeconfig objects by namespace",
		},
		[]string{"namespace"},
	)

	// ReconciliationDuration measures the time taken to reconcile kcp operator resources.
	// Labels: controller (rootshard|shard|frontproxy|kubeconfig|cacheserver), result (success|error)
	ReconciliationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kcp_operator_reconciliation_duration_seconds",
			Help:    "Time taken to reconcile objects",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"controller", "result"},
	)

	// ReconciliationErrors counts the total number of reconciliation errors by controller and error type.
	// Labels: controller (rootshard|shard|frontproxy|kubeconfig|cacheserver), error_type
	ReconciliationErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kcp_operator_reconciliation_errors_total",
			Help: "Total number of reconciliation errors",
		},
		[]string{"controller", "error_type"},
	)

	// ConditionStatus tracks the status of conditions on kcp operator resources.
	// Values: 1.0 (True), 0.0 (False), -1.0 (Unknown)
	// Labels: resource_type (rootshard|shard|frontproxy|cacheserver|kubeconfig),
	//         resource_name, namespace, condition_type (Available|RootShard)
	ConditionStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kcp_operator_condition_status",
			Help: "Status of conditions",
		},
		[]string{"resource_type", "resource_name", "namespace", "condition_type"},
	)
)

func RegisterMetrics() {
	ctrlruntimemetrics.Registry.MustRegister(
		RootShardCount,
		ShardCount,
		FrontProxyCount,
		CacheServerCount,
		KubeconfigCount,
		ReconciliationDuration,
		ReconciliationErrors,
		ConditionStatus,
	)
}
