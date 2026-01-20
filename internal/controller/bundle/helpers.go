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

package bundle

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// EnsureBundleForOwner ensures that a Bundle object exists for the given owner if it has the bundle annotation.
// If the annotation is present and no Bundle exists, it creates one owned by the object.
// Returns the Bundle object if it exists/was created, or nil if no bundle annotation is present.
func EnsureBundleForOwner(ctx context.Context, client ctrlruntimeclient.Client, scheme *runtime.Scheme, owner ctrlruntimeclient.Object) (*operatorv1alpha1.Bundle, error) {
	annotations := owner.GetAnnotations()
	if annotations == nil || annotations[resources.BundleAnnotation] == "" {
		return nil, nil
	}

	bundleName := resources.GetBundleName(owner.GetName())
	bundle := &operatorv1alpha1.Bundle{}

	err := client.Get(ctx, types.NamespacedName{
		Name:      bundleName,
		Namespace: owner.GetNamespace(),
	}, bundle)

	if err == nil {
		// Bundle already exists
		return bundle, nil
	}

	if !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to get bundle: %w", err)
	}

	// Bundle doesn't exist, create it
	bundle = &operatorv1alpha1.Bundle{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bundleName,
			Namespace: owner.GetNamespace(),
		},
		Spec: operatorv1alpha1.BundleSpec{
			Target: buildBundleTarget(owner),
		},
	}

	// Set owner reference to make the Bundle owned by the parent object
	if err := ctrl.SetControllerReference(owner, bundle, scheme); err != nil {
		return nil, fmt.Errorf("failed to set controller reference: %w", err)
	}

	if err := client.Create(ctx, bundle); err != nil {
		return nil, fmt.Errorf("failed to create bundle: %w", err)
	}

	return bundle, nil
}

// buildBundleTarget constructs the appropriate BundleTarget based on the owner type
func buildBundleTarget(owner ctrlruntimeclient.Object) operatorv1alpha1.BundleTarget {
	target := operatorv1alpha1.BundleTarget{}
	ref := &corev1.LocalObjectReference{Name: owner.GetName()}

	switch owner.(type) {
	case *operatorv1alpha1.RootShard:
		target.RootShardRef = ref
	case *operatorv1alpha1.Shard:
		target.ShardRef = ref
	case *operatorv1alpha1.FrontProxy:
		target.FrontProxyRef = ref
	}

	return target
}

// GetBundleReadyCondition checks if the Bundle associated with the owner is ready.
// Returns a condition that can be added to the owner's status.
func GetBundleReadyCondition(ctx context.Context, client ctrlruntimeclient.Client, owner ctrlruntimeclient.Object, generation int64) metav1.Condition {
	annotations := owner.GetAnnotations()
	if annotations == nil || annotations[resources.BundleAnnotation] == "" {
		// No bundle annotation, return ready condition
		return metav1.Condition{
			Type:               string(operatorv1alpha1.ConditionTypeBundle),
			Status:             metav1.ConditionTrue,
			ObservedGeneration: generation,
			Reason:             "NoBundleRequired",
			Message:            "No bundle annotation present",
		}
	}

	bundleName := resources.GetBundleName(owner.GetName())
	bundle := &operatorv1alpha1.Bundle{}

	err := client.Get(ctx, types.NamespacedName{
		Name:      bundleName,
		Namespace: owner.GetNamespace(),
	}, bundle)

	if err != nil {
		if apierrors.IsNotFound(err) {
			return metav1.Condition{
				Type:               string(operatorv1alpha1.ConditionTypeBundle),
				Status:             metav1.ConditionFalse,
				ObservedGeneration: generation,
				Reason:             "BundleNotFound",
				Message:            fmt.Sprintf("Bundle %s not found", bundleName),
			}
		}
		return metav1.Condition{
			Type:               string(operatorv1alpha1.ConditionTypeBundle),
			Status:             metav1.ConditionFalse,
			ObservedGeneration: generation,
			Reason:             "BundleCheckFailed",
			Message:            fmt.Sprintf("Failed to get bundle: %v", err),
		}
	}

	// Check if bundle is ready
	if bundle.Status.State == operatorv1alpha1.BundleStateReady {
		return metav1.Condition{
			Type:               string(operatorv1alpha1.ConditionTypeBundle),
			Status:             metav1.ConditionTrue,
			ObservedGeneration: generation,
			Reason:             "BundleReady",
			Message:            fmt.Sprintf("Bundle %s is ready", bundleName),
		}
	}

	// Use state as reason, or "BundleNotReady" if state is empty
	reason := string(bundle.Status.State)
	if reason == "" {
		reason = "BundleNotReady"
	}

	return metav1.Condition{
		Type:               string(operatorv1alpha1.ConditionTypeBundle),
		Status:             metav1.ConditionFalse,
		ObservedGeneration: generation,
		Reason:             reason,
		Message:            fmt.Sprintf("Bundle %s is in state %s", bundleName, bundle.Status.State),
	}
}
