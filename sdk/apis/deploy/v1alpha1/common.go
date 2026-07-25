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

package v1alpha1

import (
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// NamedRootShardSpec is the resolved copy of a RootShard spec.
type NamedRootShardSpec struct {
	// Name is the name of the RootShard object the spec was compiled from.
	Name string `json:"name"`

	// Spec is the resolved copy of the RootShard spec.
	Spec operatorv1alpha1.RootShardSpec `json:"spec"`
}

// NamedShardSpec is the resolved copy of a Shard spec.
type NamedShardSpec struct {
	// Name is the name of the Shard object the spec was compiled from.
	Name string `json:"name"`

	// Spec is the resolved copy of the Shard spec.
	Spec operatorv1alpha1.ShardSpec `json:"spec"`
}

// NamedVirtualWorkspaceSpec is the resolved copy of a VirtualWorkspace spec.
type NamedVirtualWorkspaceSpec struct {
	// Name is the name of the VirtualWorkspace object the spec was compiled from.
	Name string `json:"name"`

	// Spec is the resolved copy of the VirtualWorkspace spec.
	Spec operatorv1alpha1.VirtualWorkspaceSpec `json:"spec"`
}
