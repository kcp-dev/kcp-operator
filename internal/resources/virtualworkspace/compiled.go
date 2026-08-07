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

	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/pkg/controller/util"
	"github.com/kcp-dev/kcp-operator/pkg/reconciling"
	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// CompiledVirtualWorkspaceReconciler resolves a VirtualWorkspace and its target into the
// render input the CompiledVirtualWorkspace controller consumes.
func CompiledVirtualWorkspaceReconciler(vw *operatorv1alpha1.VirtualWorkspace, rootShard *operatorv1alpha1.RootShard, shard *operatorv1alpha1.Shard, revisions map[string]string) reconciling.NamedCompiledVirtualWorkspaceReconcilerFactory {
	return func() (string, reconciling.CompiledVirtualWorkspaceReconciler) {
		return vw.Name, func(obj *deployv1alpha1.CompiledVirtualWorkspace) (*deployv1alpha1.CompiledVirtualWorkspace, error) {
			// Certificate revisions ride along so a renewal changes this object, which is what
			// tells anything watching it that the Secrets it mounts have moved on.
			if obj.Annotations == nil {
				obj.Annotations = make(map[string]string)
			}
			maps.Copy(obj.Annotations, util.MutateKeys(revisions, operatorv1alpha1.GroupName+"/", ""))

			// The syncer selects the Secrets a compiled object needs by these labels.
			if obj.Labels == nil {
				obj.Labels = make(map[string]string)
			}
			obj.Labels[resources.VirtualWorkspaceLabel] = vw.Name
			obj.Labels[resources.RootShardLabel] = rootShard.Name

			obj.Spec.VirtualWorkspace = vw.Spec

			obj.Spec.RootShard = deployv1alpha1.NamedRootShardSpec{
				Name: rootShard.Name,
				Spec: rootShard.Spec,
			}

			obj.Spec.Shard = nil
			if shard != nil {
				obj.Spec.Shard = &deployv1alpha1.NamedShardSpec{
					Name: shard.Name,
					Spec: shard.Spec,
				}
			}

			resources.CopyBundleAnnotation(vw, obj)

			return obj, nil
		}
	}
}
