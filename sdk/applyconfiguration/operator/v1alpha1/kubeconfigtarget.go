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

import (
	v1 "k8s.io/api/core/v1"
)

// KubeconfigTargetApplyConfiguration represents a declarative configuration of the KubeconfigTarget type for use
// with apply.
type KubeconfigTargetApplyConfiguration struct {
	RootShardRef  *v1.LocalObjectReference `json:"rootShardRef,omitempty"`
	ShardRef      *v1.LocalObjectReference `json:"shardRef,omitempty"`
	FrontProxyRef *v1.LocalObjectReference `json:"frontProxyRef,omitempty"`
}

// KubeconfigTargetApplyConfiguration constructs a declarative configuration of the KubeconfigTarget type for use with
// apply.
func KubeconfigTarget() *KubeconfigTargetApplyConfiguration {
	return &KubeconfigTargetApplyConfiguration{}
}

// WithRootShardRef sets the RootShardRef field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the RootShardRef field is set to the value of the last call.
func (b *KubeconfigTargetApplyConfiguration) WithRootShardRef(value v1.LocalObjectReference) *KubeconfigTargetApplyConfiguration {
	b.RootShardRef = &value
	return b
}

// WithShardRef sets the ShardRef field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ShardRef field is set to the value of the last call.
func (b *KubeconfigTargetApplyConfiguration) WithShardRef(value v1.LocalObjectReference) *KubeconfigTargetApplyConfiguration {
	b.ShardRef = &value
	return b
}

// WithFrontProxyRef sets the FrontProxyRef field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the FrontProxyRef field is set to the value of the last call.
func (b *KubeconfigTargetApplyConfiguration) WithFrontProxyRef(value v1.LocalObjectReference) *KubeconfigTargetApplyConfiguration {
	b.FrontProxyRef = &value
	return b
}
