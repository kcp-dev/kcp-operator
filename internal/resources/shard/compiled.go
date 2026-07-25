/*
Copyright 2026 The kcp Authors.

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

package shard

import (
	"maps"

	"github.com/kcp-dev/kcp-operator/internal/controller/util"
	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// CompiledShardReconciler compiles the given Shard, its RootShard and its
// optional kcp virtual workspace into a CompiledShard.
func CompiledShardReconciler(shard *operatorv1alpha1.Shard, rootShard *operatorv1alpha1.RootShard, kcpVW *operatorv1alpha1.VirtualWorkspace, revisions map[string]string) reconciling.NamedCompiledShardReconcilerFactory {
	return func() (string, reconciling.CompiledShardReconciler) {
		return shard.Name, func(compiled *deployv1alpha1.CompiledShard) (*deployv1alpha1.CompiledShard, error) {
			compiled.Labels = maps.Clone(shard.Labels)
			if compiled.Labels == nil {
				compiled.Labels = make(map[string]string)
			}
			compiled.Labels[resources.RootShardLabel] = rootShard.Name
			compiled.Labels[resources.ShardLabel] = shard.Name
			compiled.Annotations = maps.Clone(shard.Annotations)
			if compiled.Annotations == nil {
				compiled.Annotations = make(map[string]string)
			}
			maps.Copy(compiled.Annotations, util.MutateKeys(revisions, operatorv1alpha1.GroupName+"/", ""))

			compiled.Spec.Shard = shard.Spec
			compiled.Spec.RootShard = deployv1alpha1.NamedRootShardSpec{
				Name: rootShard.Name,
				Spec: rootShard.Spec,
			}
			compiled.Spec.VirtualWorkspace = nil
			if kcpVW != nil {
				compiled.Spec.VirtualWorkspace = &deployv1alpha1.NamedVirtualWorkspaceSpec{
					Name: kcpVW.Name,
					Spec: kcpVW.Spec,
				}
			}

			return compiled, nil
		}
	}
}
