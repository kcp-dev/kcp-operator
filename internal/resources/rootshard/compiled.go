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
	"maps"

	"github.com/kcp-dev/kcp-operator/internal/controller/util"
	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// CompiledRootShardReconciler compiles the given RootShard and its optional kcp
// virtual workspace into a CompiledRootShard.
func CompiledRootShardReconciler(rootShard *operatorv1alpha1.RootShard, kcpVW *operatorv1alpha1.VirtualWorkspace, revisions map[string]string) reconciling.NamedCompiledRootShardReconcilerFactory {
	return func() (string, reconciling.CompiledRootShardReconciler) {
		return rootShard.Name, func(compiled *deployv1alpha1.CompiledRootShard) (*deployv1alpha1.CompiledRootShard, error) {
			compiled.Labels = maps.Clone(rootShard.Labels)
			if compiled.Labels == nil {
				compiled.Labels = make(map[string]string)
			}
			compiled.Labels[resources.RootShardLabel] = rootShard.Name
			compiled.Annotations = maps.Clone(rootShard.Annotations)
			if compiled.Annotations == nil {
				compiled.Annotations = make(map[string]string)
			}
			maps.Copy(compiled.Annotations, util.MutateKeys(revisions, operatorv1alpha1.GroupName+"/", ""))

			compiled.Spec.RootShard = rootShard.Spec
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
