/*
Copyright 2024 The KCP Authors.

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

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

// ShardSpecApplyConfiguration represents a declarative configuration of the ShardSpec type for use
// with apply.
type ShardSpecApplyConfiguration struct {
	CommonShardSpecApplyConfiguration `json:",inline"`
	RootShard                         *RootShardConfigApplyConfiguration `json:"rootShard,omitempty"`
}

// ShardSpecApplyConfiguration constructs a declarative configuration of the ShardSpec type for use with
// apply.
func ShardSpec() *ShardSpecApplyConfiguration {
	return &ShardSpecApplyConfiguration{}
}

// WithClusterDomain sets the ClusterDomain field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ClusterDomain field is set to the value of the last call.
func (b *ShardSpecApplyConfiguration) WithClusterDomain(value string) *ShardSpecApplyConfiguration {
	b.ClusterDomain = &value
	return b
}

// WithEtcd sets the Etcd field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Etcd field is set to the value of the last call.
func (b *ShardSpecApplyConfiguration) WithEtcd(value *EtcdConfigApplyConfiguration) *ShardSpecApplyConfiguration {
	b.Etcd = value
	return b
}

// WithImage sets the Image field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Image field is set to the value of the last call.
func (b *ShardSpecApplyConfiguration) WithImage(value *ImageSpecApplyConfiguration) *ShardSpecApplyConfiguration {
	b.Image = value
	return b
}

// WithReplicas sets the Replicas field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Replicas field is set to the value of the last call.
func (b *ShardSpecApplyConfiguration) WithReplicas(value int32) *ShardSpecApplyConfiguration {
	b.Replicas = &value
	return b
}

// WithRootShard sets the RootShard field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the RootShard field is set to the value of the last call.
func (b *ShardSpecApplyConfiguration) WithRootShard(value *RootShardConfigApplyConfiguration) *ShardSpecApplyConfiguration {
	b.RootShard = value
	return b
}
