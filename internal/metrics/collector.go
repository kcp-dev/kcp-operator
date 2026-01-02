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
	"context"
	"time"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

const (
	UnknownPhase = "Unknown"
)

type MetricsCollector struct {
	client ctrlruntimeclient.Client
}

func NewMetricsCollector(client ctrlruntimeclient.Client) *MetricsCollector {
	return &MetricsCollector{
		client: client,
	}
}

func (mc *MetricsCollector) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	mc.updateObjectCounts(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mc.updateObjectCounts(ctx)
		}
	}
}

func (mc *MetricsCollector) updateObjectCounts(ctx context.Context) {
	mc.updateRootShardCounts(ctx)
	mc.updateShardCounts(ctx)
	mc.updateFrontProxyCounts(ctx)
	mc.updateCacheServerCounts(ctx)
	mc.updateKubeconfigCounts(ctx)
}

func (mc *MetricsCollector) updateRootShardCounts(ctx context.Context) {
	var rootShards operatorv1alpha1.RootShardList
	if err := mc.client.List(ctx, &rootShards); err != nil {
		return
	}

	RootShardCount.Reset()

	phaseCounts := make(map[string]map[string]int)
	for _, rs := range rootShards.Items {
		phase := string(rs.Status.Phase)
		if phase == "" {
			phase = UnknownPhase
		}
		if phaseCounts[phase] == nil {
			phaseCounts[phase] = make(map[string]int)
		}
		phaseCounts[phase][rs.Namespace]++
	}

	for phase, namespaceCounts := range phaseCounts {
		for namespace, count := range namespaceCounts {
			RootShardCount.WithLabelValues(phase, namespace).Set(float64(count))
		}
	}
}

func (mc *MetricsCollector) updateShardCounts(ctx context.Context) {
	var shards operatorv1alpha1.ShardList
	if err := mc.client.List(ctx, &shards); err != nil {
		return
	}

	ShardCount.Reset()

	phaseCounts := make(map[string]map[string]int)
	for _, s := range shards.Items {
		phase := string(s.Status.Phase)
		if phase == "" {
			phase = UnknownPhase
		}
		if phaseCounts[phase] == nil {
			phaseCounts[phase] = make(map[string]int)
		}
		phaseCounts[phase][s.Namespace]++
	}

	for phase, namespaceCounts := range phaseCounts {
		for namespace, count := range namespaceCounts {
			ShardCount.WithLabelValues(phase, namespace).Set(float64(count))
		}
	}
}

func (mc *MetricsCollector) updateFrontProxyCounts(ctx context.Context) {
	var frontProxies operatorv1alpha1.FrontProxyList
	if err := mc.client.List(ctx, &frontProxies); err != nil {
		return
	}

	FrontProxyCount.Reset()

	phaseCounts := make(map[string]map[string]int)
	for _, fp := range frontProxies.Items {
		phase := string(fp.Status.Phase)
		if phase == "" {
			phase = UnknownPhase
		}
		if phaseCounts[phase] == nil {
			phaseCounts[phase] = make(map[string]int)
		}
		phaseCounts[phase][fp.Namespace]++
	}

	for phase, namespaceCounts := range phaseCounts {
		for namespace, count := range namespaceCounts {
			FrontProxyCount.WithLabelValues(phase, namespace).Set(float64(count))
		}
	}
}

func (mc *MetricsCollector) updateCacheServerCounts(ctx context.Context) {
	var cacheServers operatorv1alpha1.CacheServerList
	if err := mc.client.List(ctx, &cacheServers); err != nil {
		return
	}

	CacheServerCount.Reset()

	namespaceCounts := make(map[string]int)
	for _, cs := range cacheServers.Items {
		namespaceCounts[cs.Namespace]++
	}

	for namespace, count := range namespaceCounts {
		CacheServerCount.WithLabelValues(namespace).Set(float64(count))
	}
}

func (mc *MetricsCollector) updateKubeconfigCounts(ctx context.Context) {
	var kubeconfigs operatorv1alpha1.KubeconfigList
	if err := mc.client.List(ctx, &kubeconfigs); err != nil {
		return
	}

	KubeconfigCount.Reset()

	namespaceCounts := make(map[string]int)
	for _, kc := range kubeconfigs.Items {
		namespaceCounts[kc.Namespace]++
	}

	for namespace, count := range namespaceCounts {
		KubeconfigCount.WithLabelValues(namespace).Set(float64(count))
	}
}
