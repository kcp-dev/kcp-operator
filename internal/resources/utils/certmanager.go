/*
Copyright 2025 The KCP Authors.

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

package utils

import (
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func ToCertManagerRef(ref operatorv1alpha1.ObjectReference) certmanagermetav1.ObjectReference {
	return certmanagermetav1.ObjectReference{
		Name:  ref.Name,
		Kind:  ref.Kind,
		Group: ref.Group,
	}
}
