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
	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/utils"
	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// CompiledShardReconciler resolves a Shard and everything it references into the render
// input the CompiledShard controller consumes.
func CompiledShardReconciler(shard *operatorv1alpha1.Shard, rootShard *operatorv1alpha1.RootShard, kcpVW *operatorv1alpha1.VirtualWorkspace, shards []operatorv1alpha1.Shard) reconciling.NamedCompiledShardReconcilerFactory {
	return func() (string, reconciling.CompiledShardReconciler) {
		return shard.Name, func(obj *deployv1alpha1.CompiledShard) (*deployv1alpha1.CompiledShard, error) {
			// The syncer selects the Secrets a compiled object needs by these labels.
			if obj.Labels == nil {
				obj.Labels = make(map[string]string)
			}
			obj.Labels[resources.ShardLabel] = shard.Name
			obj.Labels[resources.RootShardLabel] = rootShard.Name

			obj.Spec.Shard = shard.Spec

			obj.Spec.RootShard = deployv1alpha1.NamedRootShardSpec{
				Name: rootShard.Name,
				Spec: rootShard.Spec,
			}

			obj.Spec.VirtualWorkspace = nil
			if kcpVW != nil {
				obj.Spec.VirtualWorkspace = &deployv1alpha1.NamedVirtualWorkspaceSpec{
					Name: kcpVW.Name,
					Spec: kcpVW.Spec,
				}
			}

			obj.Spec.Shards = utils.ShardNames(shards)

			resources.CopyBundleAnnotation(shard, obj)

			return obj, nil
		}
	}
}
