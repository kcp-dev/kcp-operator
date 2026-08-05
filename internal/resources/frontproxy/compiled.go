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

package frontproxy

import (
	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/utils"
	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// CompiledFrontProxyReconciler resolves a FrontProxy and everything it references into the
// render input the CompiledFrontProxy controller consumes.
func CompiledFrontProxyReconciler(frontProxy *operatorv1alpha1.FrontProxy, rootShard *operatorv1alpha1.RootShard, shards []operatorv1alpha1.Shard) reconciling.NamedCompiledFrontProxyReconcilerFactory {
	return func() (string, reconciling.CompiledFrontProxyReconciler) {
		return frontProxy.Name, func(obj *deployv1alpha1.CompiledFrontProxy) (*deployv1alpha1.CompiledFrontProxy, error) {
			obj.Spec.FrontProxy = frontProxy.Spec

			obj.Spec.RootShard = deployv1alpha1.NamedRootShardSpec{
				Name: rootShard.Name,
				Spec: rootShard.Spec,
			}

			obj.Spec.Shards = utils.ShardNames(shards)

			resources.CopyBundleAnnotation(frontProxy, obj)

			return obj, nil
		}
	}
}
