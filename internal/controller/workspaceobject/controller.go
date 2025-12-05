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

package workspaceobject

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

const (
	WorkspaceObjectFinalizer = "operator.kcp.io/workspaceobject-cleanup"
	RequeueDelay             = 10 * time.Second
	fieldOwner               = "kcp-operator"
)

// WorkspaceObjectReconciler reconciles a WorkspaceObject object
type WorkspaceObjectReconciler struct {
	ctrlruntimeclient.Client
	Scheme *runtime.Scheme

	// stubs that can be replaced with mocks for testing
	getWorkspaceDynamicClient workspaceClientCreatorFunc
	getGVRFromGVK             mapperGVRFromGVKFunc
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkspaceObjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// register default handlers
	// (others are only used for tests)
	if r.getWorkspaceDynamicClient == nil {
		r.getWorkspaceDynamicClient = getWorkspaceDynamicClient
	}
	if r.getGVRFromGVK == nil {
		r.getGVRFromGVK = getGVRFromGVK
	}

	rootShardHandler := handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, obj ctrlruntimeclient.Object) []reconcile.Request {
		rootShard := obj.(*operatorv1alpha1.RootShard)

		var workspaceObjectList operatorv1alpha1.WorkspaceObjectList
		if err := mgr.GetClient().List(ctx, &workspaceObjectList); err != nil {
			return nil
		}

		requests := make([]reconcile.Request, 0, len(workspaceObjectList.Items))
		for _, workspaceObject := range workspaceObjectList.Items {
			if workspaceObject.Spec.RootShard.Reference != nil &&
				workspaceObject.Spec.RootShard.Reference.Name == rootShard.Name &&
				workspaceObject.GetNamespace() == rootShard.GetNamespace() {
				requests = append(requests, reconcile.Request{NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(&workspaceObject)})
			}
		}

		return requests
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.WorkspaceObject{}).
		Watches(&operatorv1alpha1.RootShard{}, rootShardHandler).
		Complete(r)
}

// +kubebuilder:rbac:groups=operator.kcp.io,resources=workspaceobjects,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=workspaceobjects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=workspaceobjects/finalizers,verbs=update
// +kubebuilder:rbac:groups=operator.kcp.io,resources=rootshards,verbs=get;list;watch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=workspaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *WorkspaceObjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, recErr error) {
	logger := log.FromContext(ctx)
	logger.V(4).Info("Reconciling WorkspaceObject")

	var workspaceObject operatorv1alpha1.WorkspaceObject
	if err := r.Get(ctx, req.NamespacedName, &workspaceObject); err != nil {
		if ctrlruntimeclient.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, fmt.Errorf("failed to find %s/%s: %w", req.Namespace, req.Name, err)
		}

		// Object has apparently been deleted already.
		return ctrl.Result{}, nil
	}

	// Save old state before reconciliation for status patching
	oldWorkspaceObject := workspaceObject.DeepCopy()

	conditions, recErr := r.reconcile(ctx, &workspaceObject, oldWorkspaceObject)

	if err := r.reconcileStatus(ctx, &workspaceObject, oldWorkspaceObject, conditions); err != nil {
		recErr = kerrors.NewAggregate([]error{recErr, err})
	}

	// If we encounter transient errors, requeue after delay
	if recErr != nil {
		return ctrl.Result{RequeueAfter: RequeueDelay}, recErr
	}

	return ctrl.Result{}, nil
}

func (r *WorkspaceObjectReconciler) reconcile(ctx context.Context, workspaceObject, oldWorkspaceObject *operatorv1alpha1.WorkspaceObject) ([]metav1.Condition, error) {
	var (
		errs       []error
		conditions []metav1.Condition
	)

	// Handle deletion
	if workspaceObject.DeletionTimestamp != nil {
		if controllerutil.ContainsFinalizer(workspaceObject, WorkspaceObjectFinalizer) {
			if err := r.deleteWorkspaceObject(ctx, workspaceObject); err != nil {
				conditions = append(conditions, metav1.Condition{
					Type:               string(operatorv1alpha1.ConditionTypeAvailable),
					Status:             metav1.ConditionFalse,
					Reason:             "DeletionFailed",
					Message:            fmt.Sprintf("Failed to delete object in workspace: %v", err),
					LastTransitionTime: metav1.Now(),
				})
				return conditions, err
			}

			// Remove finalizer after successful deletion using server-side apply
			controllerutil.RemoveFinalizer(workspaceObject, WorkspaceObjectFinalizer)
			if err := r.applyFinalizers(ctx, workspaceObject, oldWorkspaceObject); err != nil {
				return conditions, fmt.Errorf("failed to remove finalizer: %w", err)
			}
		}
		return conditions, nil
	}

	// Add finalizer if not present (server-side apply)
	if !controllerutil.ContainsFinalizer(workspaceObject, WorkspaceObjectFinalizer) {
		controllerutil.AddFinalizer(workspaceObject, WorkspaceObjectFinalizer)
		if err := r.applyFinalizers(ctx, workspaceObject, oldWorkspaceObject); err != nil {
			return conditions, fmt.Errorf("failed to add finalizer: %w", err)
		}
	}

	// Get dynamic client for the workspace
	dynamicClient, restConfig, err := r.getWorkspaceDynamicClient(ctx, r.Client, workspaceObject)
	if err != nil {
		conditions = append(conditions, metav1.Condition{
			Type:               string(operatorv1alpha1.ConditionTypeAvailable),
			Status:             metav1.ConditionFalse,
			Reason:             "WorkspaceConnectionFailed",
			Message:            fmt.Sprintf("Failed to connect to workspace: %v", err),
			LastTransitionTime: metav1.Now(),
		})
		errs = append(errs, err)
		return conditions, kerrors.NewAggregate(errs)
	}

	// Apply the manifest in the workspace
	appliedManifest, err := r.applyManifest(ctx, dynamicClient, restConfig, workspaceObject)
	if err != nil {
		conditions = append(conditions, metav1.Condition{
			Type:               string(operatorv1alpha1.ConditionTypeAvailable),
			Status:             metav1.ConditionFalse,
			Reason:             "ManifestApplyFailed",
			Message:            fmt.Sprintf("Failed to apply manifest: %v", err),
			LastTransitionTime: metav1.Now(),
		})
		errs = append(errs, err)
		return conditions, kerrors.NewAggregate(errs)
	}

	// Update status with applied manifest
	workspaceObject.Status.Manifest = appliedManifest

	conditions = append(conditions, metav1.Condition{
		Type:               string(operatorv1alpha1.ConditionTypeAvailable),
		Status:             metav1.ConditionTrue,
		Reason:             "ManifestApplied",
		Message:            "Manifest successfully applied in workspace",
		LastTransitionTime: metav1.Now(),
	})

	return conditions, kerrors.NewAggregate(errs)
}

// applyManifest applies the manifest from the WorkspaceObject spec to the workspace
func (r *WorkspaceObjectReconciler) applyManifest(ctx context.Context, dynamicClient dynamic.Interface, restConfig *rest.Config, workspaceObject *operatorv1alpha1.WorkspaceObject) (*apiextensionsv1.JSON, error) {
	// Parse the manifest into an unstructured object
	var obj unstructured.Unstructured
	if err := json.Unmarshal(workspaceObject.Spec.Manifest.Raw, &obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	gvk := obj.GroupVersionKind()

	// Get the GVR from the discovery client
	gvr, err := r.getGVRFromGVK(restConfig, gvk)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource mapping for %s: %w", gvk.String(), err)
	}

	// Check management policies
	shouldCreate := r.shouldManage(workspaceObject, operatorv1alpha1.WorkspaceObjectManagementPolicyCreate)
	shouldUpdate := r.shouldManage(workspaceObject, operatorv1alpha1.WorkspaceObjectManagementPolicyUpdate)

	// Get the resource interface
	var resourceInterface dynamic.ResourceInterface
	if obj.GetNamespace() != "" {
		resourceInterface = dynamicClient.Resource(gvr).Namespace(obj.GetNamespace())
	} else {
		resourceInterface = dynamicClient.Resource(gvr)
	}

	// Determine existence for policy enforcement
	existing, err := resourceInterface.Get(ctx, obj.GetName(), metav1.GetOptions{})
	notFound := apierrors.IsNotFound(err)
	if err != nil && !notFound {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	if notFound && !shouldCreate {
		return nil, fmt.Errorf("object does not exist and create policy is not enabled")
	}
	if !notFound && !shouldUpdate {
		return marshalToJSON(existing)
	}

	// Server-side apply: always send entire desired object.
	obj.SetManagedFields(nil)
	// Ensure GVK present (usually already from manifest)
	patchBytes, err := json.Marshal(&obj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal object for apply: %w", err)
	}
	force := true
	applied, err := resourceInterface.Patch(ctx, obj.GetName(), types.ApplyPatchType, patchBytes, metav1.PatchOptions{FieldManager: fieldOwner, Force: &force})
	if err != nil {
		return nil, fmt.Errorf("failed to apply object: %w", err)
	}
	return marshalToJSON(applied)
}

// applyFinalizers updates the finalizers using a merge patch.
func (r *WorkspaceObjectReconciler) applyFinalizers(ctx context.Context, workspaceObject, oldWorkspaceObject *operatorv1alpha1.WorkspaceObject) error {
	patch := ctrlruntimeclient.MergeFrom(oldWorkspaceObject)
	if err := r.Patch(ctx, workspaceObject, patch); err != nil {
		return err
	}
	return nil
}

// deleteWorkspaceObject removes the object from the workspace
func (r *WorkspaceObjectReconciler) deleteWorkspaceObject(ctx context.Context, workspaceObject *operatorv1alpha1.WorkspaceObject) error {
	// Check if delete is allowed by management policies
	if !r.shouldManage(workspaceObject, operatorv1alpha1.WorkspaceObjectManagementPolicyDelete) {
		// Deletion not managed, just remove finalizer
		return nil
	}

	dynamicClient, restConfig, err := r.getWorkspaceDynamicClient(ctx, r.Client, workspaceObject)
	if err != nil {
		// If we can't connect to workspace, log but don't block deletion
		return nil
	}

	// Parse the manifest to get object details
	var obj unstructured.Unstructured
	if err := json.Unmarshal(workspaceObject.Spec.Manifest.Raw, &obj); err != nil {
		return fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	gvk := obj.GroupVersionKind()

	// Get the GVR from the discovery client
	gvr, err := r.getGVRFromGVK(restConfig, gvk)
	if err != nil {
		return fmt.Errorf("failed to get resource mapping for %s: %w", gvk.String(), err)
	}

	// Get the resource interface
	var resourceInterface dynamic.ResourceInterface
	if obj.GetNamespace() != "" {
		resourceInterface = dynamicClient.Resource(gvr).Namespace(obj.GetNamespace())
	} else {
		resourceInterface = dynamicClient.Resource(gvr)
	}

	// Delete the object
	err = resourceInterface.Delete(ctx, obj.GetName(), metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete object %s: %w", obj.GetName(), err)
	}

	return nil
}

// reconcileStatus updates the status of the WorkspaceObject
func (r *WorkspaceObjectReconciler) reconcileStatus(ctx context.Context, workspaceObject *operatorv1alpha1.WorkspaceObject, oldWorkspaceObject *operatorv1alpha1.WorkspaceObject, conditions []metav1.Condition) error {
	// Update conditions
	for _, condition := range conditions {
		condition.ObservedGeneration = workspaceObject.Generation
		workspaceObject.Status.Conditions = updateCondition(workspaceObject.Status.Conditions, condition)
	}

	// Only patch the status if there are actual changes
	if !equality.Semantic.DeepEqual(oldWorkspaceObject.Status, workspaceObject.Status) {
		if err := r.Status().Patch(ctx, workspaceObject, ctrlruntimeclient.MergeFrom(oldWorkspaceObject)); err != nil {
			return fmt.Errorf("failed to update status: %w", err)
		}
	}

	return nil
}

// updateCondition updates or appends a condition to the condition list
func updateCondition(conditions []metav1.Condition, newCondition metav1.Condition) []metav1.Condition {
	existingCondition := apimeta.FindStatusCondition(conditions, newCondition.Type)
	if existingCondition == nil {
		return append(conditions, newCondition)
	}

	if existingCondition.Status != newCondition.Status ||
		existingCondition.Reason != newCondition.Reason ||
		existingCondition.Message != newCondition.Message {
		existingCondition.Status = newCondition.Status
		existingCondition.Reason = newCondition.Reason
		existingCondition.Message = newCondition.Message
		existingCondition.LastTransitionTime = metav1.Now()
		existingCondition.ObservedGeneration = newCondition.ObservedGeneration
	}

	return conditions
}

// shouldManage checks if a given management policy is enabled
func (r *WorkspaceObjectReconciler) shouldManage(workspaceObject *operatorv1alpha1.WorkspaceObject, policy operatorv1alpha1.WorkspaceObjectManagementPolicy) bool {
	for _, p := range workspaceObject.Spec.ManagementPolicies {
		if p == operatorv1alpha1.WorkspaceObjectManagementPolicyAll || p == policy {
			return true
		}
	}
	return false
}

// marshalToJSON converts an unstructured object to extv1.JSON
func marshalToJSON(obj any) (*apiextensionsv1.JSON, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal object: %w", err)
	}
	return &apiextensionsv1.JSON{Raw: data}, nil
}
