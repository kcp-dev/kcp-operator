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
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubeconfigSpecApplyConfiguration represents a declarative configuration of the KubeconfigSpec type for use
// with apply.
type KubeconfigSpecApplyConfiguration struct {
	Target    *KubeconfigTargetApplyConfiguration `json:"target,omitempty"`
	Username  *string                             `json:"username,omitempty"`
	Groups    []string                            `json:"groups,omitempty"`
	Validity  *v1.Duration                        `json:"validity,omitempty"`
	SecretRef *corev1.LocalObjectReference        `json:"secretRef,omitempty"`
}

// KubeconfigSpecApplyConfiguration constructs a declarative configuration of the KubeconfigSpec type for use with
// apply.
func KubeconfigSpec() *KubeconfigSpecApplyConfiguration {
	return &KubeconfigSpecApplyConfiguration{}
}

// WithTarget sets the Target field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Target field is set to the value of the last call.
func (b *KubeconfigSpecApplyConfiguration) WithTarget(value *KubeconfigTargetApplyConfiguration) *KubeconfigSpecApplyConfiguration {
	b.Target = value
	return b
}

// WithUsername sets the Username field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Username field is set to the value of the last call.
func (b *KubeconfigSpecApplyConfiguration) WithUsername(value string) *KubeconfigSpecApplyConfiguration {
	b.Username = &value
	return b
}

// WithGroups adds the given value to the Groups field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Groups field.
func (b *KubeconfigSpecApplyConfiguration) WithGroups(values ...string) *KubeconfigSpecApplyConfiguration {
	for i := range values {
		b.Groups = append(b.Groups, values[i])
	}
	return b
}

// WithValidity sets the Validity field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Validity field is set to the value of the last call.
func (b *KubeconfigSpecApplyConfiguration) WithValidity(value v1.Duration) *KubeconfigSpecApplyConfiguration {
	b.Validity = &value
	return b
}

// WithSecretRef sets the SecretRef field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the SecretRef field is set to the value of the last call.
func (b *KubeconfigSpecApplyConfiguration) WithSecretRef(value corev1.LocalObjectReference) *KubeconfigSpecApplyConfiguration {
	b.SecretRef = &value
	return b
}
