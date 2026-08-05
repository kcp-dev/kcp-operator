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

package rootshard

import (
	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/utils"
	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// CompiledRootShardReconciler resolves a RootShard and everything it references into the
// render input the CompiledRootShard controller consumes.
func CompiledRootShardReconciler(rootShard *operatorv1alpha1.RootShard, kcpVW *operatorv1alpha1.VirtualWorkspace, shards []operatorv1alpha1.Shard) reconciling.NamedCompiledRootShardReconcilerFactory {
	return func() (string, reconciling.CompiledRootShardReconciler) {
		return rootShard.Name, func(obj *deployv1alpha1.CompiledRootShard) (*deployv1alpha1.CompiledRootShard, error) {
			obj.Spec.RootShard = rootShard.Spec

			obj.Spec.VirtualWorkspace = nil
			if kcpVW != nil {
				obj.Spec.VirtualWorkspace = &deployv1alpha1.NamedVirtualWorkspaceSpec{
					Name: kcpVW.Name,
					Spec: kcpVW.Spec,
				}
			}

			obj.Spec.Shards = utils.ShardNames(shards)

			resources.CopyBundleAnnotation(rootShard, obj)

			return obj, nil
		}
	}
}
