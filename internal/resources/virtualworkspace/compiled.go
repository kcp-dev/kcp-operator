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

package virtualworkspace

import (
	"maps"

	"github.com/kcp-dev/kcp-operator/internal/controller/util"
	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// CompiledVirtualWorkspaceReconciler compiles the given VirtualWorkspace, its
// RootShard and its optional Shard target into a CompiledVirtualWorkspace.
func CompiledVirtualWorkspaceReconciler(vw *operatorv1alpha1.VirtualWorkspace, rootShard *operatorv1alpha1.RootShard, shard *operatorv1alpha1.Shard, revisions map[string]string) reconciling.NamedCompiledVirtualWorkspaceReconcilerFactory {
	return func() (string, reconciling.CompiledVirtualWorkspaceReconciler) {
		return vw.Name, func(compiled *deployv1alpha1.CompiledVirtualWorkspace) (*deployv1alpha1.CompiledVirtualWorkspace, error) {
			compiled.Labels = maps.Clone(vw.Labels)
			if compiled.Labels == nil {
				compiled.Labels = make(map[string]string)
			}
			compiled.Labels[resources.RootShardLabel] = rootShard.Name
			compiled.Labels[resources.VirtualWorkspaceLabel] = vw.Name
			compiled.Annotations = maps.Clone(vw.Annotations)
			if compiled.Annotations == nil {
				compiled.Annotations = make(map[string]string)
			}
			maps.Copy(compiled.Annotations, util.MutateKeys(revisions, operatorv1alpha1.GroupName+"/", ""))

			compiled.Spec.VirtualWorkspace = vw.Spec
			compiled.Spec.RootShard = deployv1alpha1.NamedRootShardSpec{
				Name: rootShard.Name,
				Spec: rootShard.Spec,
			}
			compiled.Spec.Shard = nil
			if shard != nil {
				compiled.Spec.Shard = &deployv1alpha1.NamedShardSpec{
					Name: shard.Name,
					Spec: shard.Spec,
				}
			}

			return compiled, nil
		}
	}
}
