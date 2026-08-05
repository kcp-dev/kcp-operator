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
	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// CompiledVirtualWorkspaceReconciler resolves a VirtualWorkspace and its target into the
// render input the CompiledVirtualWorkspace controller consumes.
func CompiledVirtualWorkspaceReconciler(vw *operatorv1alpha1.VirtualWorkspace, rootShard *operatorv1alpha1.RootShard, shard *operatorv1alpha1.Shard) reconciling.NamedCompiledVirtualWorkspaceReconcilerFactory {
	return func() (string, reconciling.CompiledVirtualWorkspaceReconciler) {
		return vw.Name, func(obj *deployv1alpha1.CompiledVirtualWorkspace) (*deployv1alpha1.CompiledVirtualWorkspace, error) {
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
