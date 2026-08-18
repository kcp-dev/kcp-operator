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

package cacheserver

import (
	"maps"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/pkg/controller/util"
	"github.com/kcp-dev/kcp-operator/pkg/reconciling"
	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// CompiledCacheServerReconciler resolves a CacheServer into the render input the
// CompiledCacheServer controller consumes.
func CompiledCacheServerReconciler(server *operatorv1alpha1.CacheServer, revisions map[string]string) reconciling.NamedCompiledCacheServerReconcilerFactory {
	return func() (string, reconciling.CompiledCacheServerReconciler) {
		return server.Name, func(obj *deployv1alpha1.CompiledCacheServer) (*deployv1alpha1.CompiledCacheServer, error) {
			obj.Labels = maps.Clone(server.Labels)
			obj.Annotations = maps.Clone(server.Annotations)
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
			obj.Labels[resources.CacheServerLabel] = server.Name

			obj.Spec.CacheServer = server.Spec

			return obj, nil
		}
	}
}
