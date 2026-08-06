/*
Copyright 2026 The KCP Authors.

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

package modifier

import (
	"k8c.io/reconciler/pkg/reconciling"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Capture appends the object of every ensured object through the reconciler chain to out.
// Objects that don't exist yet are empty objects.
func Capture[T ctrlruntimeclient.Object](out *[]T) reconciling.ObjectModifier {
	return func(reconciler reconciling.ObjectReconciler) reconciling.ObjectReconciler {
		return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
			obj, err := reconciler(existing)
			if err != nil {
				return obj, err
			}

			if captured, ok := existing.(T); ok {
				*out = append(*out, captured)
			}

			return obj, nil
		}
	}
}
