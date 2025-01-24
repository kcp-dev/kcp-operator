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

package kubernetes

import (
	"maps"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// HasFinalizer tells if a object has all the given finalizers.
func HasFinalizer(o metav1.Object, names ...string) bool {
	return sets.New(o.GetFinalizers()...).HasAll(names...)
}

func HasAnyFinalizer(o metav1.Object, names ...string) bool {
	return sets.New(o.GetFinalizers()...).HasAny(names...)
}

// HasOnlyFinalizer tells if an object has only the given finalizer(s).
func HasOnlyFinalizer(o metav1.Object, names ...string) bool {
	return sets.New(o.GetFinalizers()...).Equal(sets.New(names...))
}

// HasFinalizerSuperset tells if the given finalizer(s) are a superset
// of the actual finalizers.
func HasFinalizerSuperset(o metav1.Object, names ...string) bool {
	return sets.New(names...).IsSuperset(sets.New(o.GetFinalizers()...))
}

// RemoveFinalizer removes the given finalizers from the object.
func RemoveFinalizer(obj metav1.Object, toRemove ...string) {
	set := sets.New(obj.GetFinalizers()...)
	set.Delete(toRemove...)
	obj.SetFinalizers(sets.List(set))
}

func EnsureLabels(o metav1.Object, toEnsure map[string]string) {
	labels := maps.Clone(o.GetLabels())

	if labels == nil {
		labels = make(map[string]string)
	}
	for key, value := range toEnsure {
		labels[key] = value
	}
	o.SetLabels(labels)
}

func EnsureAnnotations(o metav1.Object, toEnsure map[string]string) {
	annotations := maps.Clone(o.GetAnnotations())

	if annotations == nil {
		annotations = make(map[string]string)
	}
	for key, value := range toEnsure {
		annotations[key] = value
	}
	o.SetAnnotations(annotations)
}
