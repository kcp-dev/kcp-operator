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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	k8creconciling "k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// BundleReconciler reconciles a Bundle object
type BundleReconciler struct {
	ctrlruntimeclient.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *BundleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Handler for RootShard changes - enqueue all Bundles targeting this RootShard
	rootShardHandler := handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, obj ctrlruntimeclient.Object) []reconcile.Request {
		rootShard := obj.(*operatorv1alpha1.RootShard)

		var bundles operatorv1alpha1.BundleList
		if err := mgr.GetClient().List(ctx, &bundles, &ctrlruntimeclient.ListOptions{Namespace: rootShard.Namespace}); err != nil {
			utilruntime.HandleError(err)
			return nil
		}

		var requests []reconcile.Request
		for _, bundle := range bundles.Items {
			if bundle.Spec.Target.RootShardRef != nil && bundle.Spec.Target.RootShardRef.Name == rootShard.Name {
				requests = append(requests, reconcile.Request{NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(&bundle)})
			}
		}
		return requests
	})

	// Handler for Shard changes - enqueue all Bundles targeting this Shard
	shardHandler := handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, obj ctrlruntimeclient.Object) []reconcile.Request {
		shard := obj.(*operatorv1alpha1.Shard)

		var bundles operatorv1alpha1.BundleList
		if err := mgr.GetClient().List(ctx, &bundles, &ctrlruntimeclient.ListOptions{Namespace: shard.Namespace}); err != nil {
			utilruntime.HandleError(err)
			return nil
		}

		var requests []reconcile.Request
		for _, bundle := range bundles.Items {
			if bundle.Spec.Target.ShardRef != nil && bundle.Spec.Target.ShardRef.Name == shard.Name {
				requests = append(requests, reconcile.Request{NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(&bundle)})
			}
		}
		return requests
	})

	// Handler for FrontProxy changes - enqueue all Bundles targeting this FrontProxy
	frontProxyHandler := handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, obj ctrlruntimeclient.Object) []reconcile.Request {
		frontProxy := obj.(*operatorv1alpha1.FrontProxy)

		var bundles operatorv1alpha1.BundleList
		if err := mgr.GetClient().List(ctx, &bundles, &ctrlruntimeclient.ListOptions{Namespace: frontProxy.Namespace}); err != nil {
			utilruntime.HandleError(err)
			return nil
		}

		var requests []reconcile.Request
		for _, bundle := range bundles.Items {
			if bundle.Spec.Target.FrontProxyRef != nil && bundle.Spec.Target.FrontProxyRef.Name == frontProxy.Name {
				requests = append(requests, reconcile.Request{NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(&bundle)})
			}
		}
		return requests
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.Bundle{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Watches(&operatorv1alpha1.RootShard{}, rootShardHandler).
		Watches(&operatorv1alpha1.Shard{}, shardHandler).
		Watches(&operatorv1alpha1.FrontProxy{}, frontProxyHandler).
		Complete(r)
}

// +kubebuilder:rbac:groups=operator.kcp.io,resources=bundles,verbs=get;list;watch;update;patch;create;delete
// +kubebuilder:rbac:groups=operator.kcp.io,resources=bundles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=bundles/finalizers,verbs=update
// +kubebuilder:rbac:groups=operator.kcp.io,resources=rootshards;shards;frontproxies,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=configmaps;secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *BundleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, recErr error) {
	logger := log.FromContext(ctx)
	logger.V(4).Info("Reconciling Bundle object")

	var bundle operatorv1alpha1.Bundle
	if err := r.Get(ctx, req.NamespacedName, &bundle); err != nil {
		if ctrlruntimeclient.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get bundle: %w", err)
		}

		return ctrl.Result{}, nil
	}

	recErr = r.reconcile(ctx, &bundle)

	return ctrl.Result{}, recErr
}

func (r *BundleReconciler) reconcile(ctx context.Context, bundle *operatorv1alpha1.Bundle) error {
	oldStatus := bundle.Status.DeepCopy()
	var (
		errs       []error
		conditions []metav1.Condition
	)

	if bundle.DeletionTimestamp != nil {
		return r.updateStatus(ctx, bundle, oldStatus, conditions, errs)
	}

	// Determine which target the bundle is for and get the list of required objects
	target := bundle.Spec.Target
	var (
		targetExists    bool
		targetName      string
		requiredObjects []operatorv1alpha1.BundleObject
		rootShard       *operatorv1alpha1.RootShard
		shard           *operatorv1alpha1.Shard
		frontProxy      *operatorv1alpha1.FrontProxy
	)

	switch {
	case target.RootShardRef != nil:
		targetName = target.RootShardRef.Name
		rootShard = &operatorv1alpha1.RootShard{}
		err := r.Get(ctx, ctrlruntimeclient.ObjectKey{
			Name:      targetName,
			Namespace: bundle.Namespace,
		}, rootShard)
		if err == nil {
			targetExists = true
			requiredObjects = getBundleObjectsForRootShard(rootShard)
		} else if ctrlruntimeclient.IgnoreNotFound(err) != nil {
			errs = append(errs, fmt.Errorf("failed to get RootShard %s: %w", targetName, err))
		}

	case target.ShardRef != nil:
		targetName = target.ShardRef.Name
		shard = &operatorv1alpha1.Shard{}
		err := r.Get(ctx, ctrlruntimeclient.ObjectKey{
			Name:      targetName,
			Namespace: bundle.Namespace,
		}, shard)
		if err == nil {
			targetExists = true
			// Need to get RootShard for Shard bundles
			if shard.Spec.RootShard.Reference != nil {
				rootShard = &operatorv1alpha1.RootShard{}
				err := r.Get(ctx, ctrlruntimeclient.ObjectKey{
					Name:      shard.Spec.RootShard.Reference.Name,
					Namespace: bundle.Namespace,
				}, rootShard)
				if err == nil {
					requiredObjects = getBundleObjectsForShard(shard, rootShard.Name)
				} else {
					errs = append(errs, fmt.Errorf("failed to get RootShard for Shard %s: %w", targetName, err))
				}
			}
		} else if ctrlruntimeclient.IgnoreNotFound(err) != nil {
			errs = append(errs, fmt.Errorf("failed to get Shard %s: %w", targetName, err))
		}

	case target.FrontProxyRef != nil:
		targetName = target.FrontProxyRef.Name
		frontProxy = &operatorv1alpha1.FrontProxy{}
		err := r.Get(ctx, ctrlruntimeclient.ObjectKey{
			Name:      targetName,
			Namespace: bundle.Namespace,
		}, frontProxy)
		if err == nil {
			targetExists = true
			// Need to get RootShard for FrontProxy bundles
			if frontProxy.Spec.RootShard.Reference != nil {
				rootShard = &operatorv1alpha1.RootShard{}
				err := r.Get(ctx, ctrlruntimeclient.ObjectKey{
					Name:      frontProxy.Spec.RootShard.Reference.Name,
					Namespace: bundle.Namespace,
				}, rootShard)
				if err == nil {
					requiredObjects = getBundleObjectsForFrontProxy(frontProxy, rootShard.Name)
				} else {
					errs = append(errs, fmt.Errorf("failed to get RootShard for FrontProxy %s: %w", targetName, err))
				}
			}
		} else if ctrlruntimeclient.IgnoreNotFound(err) != nil {
			errs = append(errs, fmt.Errorf("failed to get FrontProxy %s: %w", targetName, err))
		}

	default:
		errs = append(errs, fmt.Errorf("bundle has no target reference"))
	}

	if !targetExists {
		conditions = append(conditions, metav1.Condition{
			Type:    string(operatorv1alpha1.ConditionTypeReady),
			Status:  metav1.ConditionFalse,
			Reason:  "TargetNotFound",
			Message: fmt.Sprintf("Target object %s not found", targetName),
		})
		return r.updateStatus(ctx, bundle, oldStatus, conditions, errs)
	}

	// Check the status of all required objects
	objectStatuses, allReady := r.checkBundleObjects(ctx, requiredObjects)

	// Update the bundle status with object statuses
	bundle.Status.Objects = objectStatuses

	if allReady {
		conditions = append(conditions, metav1.Condition{
			Type:    string(operatorv1alpha1.ConditionTypeReady),
			Status:  metav1.ConditionTrue,
			Reason:  "BundleReady",
			Message: fmt.Sprintf("All %d required objects are ready", len(requiredObjects)),
		})

		// Create bundle objects (ConfigMap/Secret) once all required objects are ready
		if err := r.createBundleObjects(ctx, bundle, requiredObjects); err != nil {
			errs = append(errs, fmt.Errorf("failed to create bundle objects: %w", err))
			conditions = append(conditions, metav1.Condition{
				Type:    string(operatorv1alpha1.ConditionTypeObjectsCreated),
				Status:  metav1.ConditionFalse,
				Reason:  "CreationFailed",
				Message: fmt.Sprintf("Failed to create bundle objects: %v", err),
			})
		} else {
			conditions = append(conditions, metav1.Condition{
				Type:    string(operatorv1alpha1.ConditionTypeObjectsCreated),
				Status:  metav1.ConditionTrue,
				Reason:  "ObjectsCreated",
				Message: "Bundle objects created successfully",
			})
		}
	} else {
		notReadyCount := 0
		for _, obj := range objectStatuses {
			if obj.State != operatorv1alpha1.BundleObjectStateReady {
				notReadyCount++
			}
		}
		conditions = append(conditions, metav1.Condition{
			Type:    string(operatorv1alpha1.ConditionTypeReady),
			Status:  metav1.ConditionFalse,
			Reason:  "ObjectsNotReady",
			Message: fmt.Sprintf("%d of %d objects are not ready", notReadyCount, len(requiredObjects)),
		})
		conditions = append(conditions, metav1.Condition{
			Type:    string(operatorv1alpha1.ConditionTypeObjectsCreated),
			Status:  metav1.ConditionFalse,
			Reason:  "WaitingForObjects",
			Message: "Waiting for all required objects to be ready before creating bundle objects",
		})
	}

	return r.updateStatus(ctx, bundle, oldStatus, conditions, errs)
}

// checkBundleObjects checks the status of all required objects and returns their statuses
func (r *BundleReconciler) checkBundleObjects(ctx context.Context, requiredObjects []operatorv1alpha1.BundleObject) ([]operatorv1alpha1.BundleObjectStatus, bool) {
	objectStatuses := make([]operatorv1alpha1.BundleObjectStatus, 0, len(requiredObjects))
	allReady := true

	for _, obj := range requiredObjects {
		status := r.checkObject(ctx, obj)
		objectStatuses = append(objectStatuses, status)
		if status.State != operatorv1alpha1.BundleObjectStateReady {
			allReady = false
		}
	}

	return objectStatuses, allReady
}

// newResourceForGVR creates an empty resource object based on the GVR resource type
func newResourceForGVR(gvrResource string) ctrlruntimeclient.Object {
	switch gvrResource {
	case "secrets":
		return &corev1.Secret{}
	case "services":
		return &corev1.Service{}
	case "configmaps":
		return &corev1.ConfigMap{}
	case "deployments":
		return &appsv1.Deployment{}
	default:
		return nil
	}
}

// checkObject checks if a specific object exists and is ready
func (r *BundleReconciler) checkObject(ctx context.Context, obj operatorv1alpha1.BundleObject) operatorv1alpha1.BundleObjectStatus {
	status := operatorv1alpha1.BundleObjectStatus{
		Object: obj.String(),
	}

	resource := newResourceForGVR(obj.GVR.Resource)
	if resource == nil {
		status.State = operatorv1alpha1.BundleObjectStateNotReady
		status.Message = fmt.Sprintf("unsupported resource type: %s", obj.GVR.Resource)
		return status
	}

	err := r.Get(ctx, ctrlruntimeclient.ObjectKey{
		Name:      obj.Name,
		Namespace: obj.Namespace,
	}, resource)

	if err != nil {
		if apierrors.IsNotFound(err) {
			status.State = operatorv1alpha1.BundleObjectStateNotReady
			status.Message = "object not found"
		} else {
			status.State = operatorv1alpha1.BundleObjectStateNotReady
			status.Message = fmt.Sprintf("error checking object: %v", err)
		}
		return status
	}

	// Object exists, mark as ready
	status.State = operatorv1alpha1.BundleObjectStateReady
	status.Message = ""

	return status
}

// createBundleObjects creates a Secret objects containing the bundle data
func (r *BundleReconciler) createBundleObjects(ctx context.Context, bundle *operatorv1alpha1.Bundle, requiredObjects []operatorv1alpha1.BundleObject) error {
	// Collect all required objects
	objects := make([]ctrlruntimeclient.Object, 0, len(requiredObjects))
	for _, obj := range requiredObjects {
		resource := newResourceForGVR(obj.GVR.Resource)
		if resource == nil {
			continue
		}

		if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{
			Name:      obj.Name,
			Namespace: obj.Namespace,
		}, resource); err != nil {
			return fmt.Errorf("failed to get required object %s: %w", obj.String(), err)
		}

		// If this is a Deployment with the bundle-desired-replicas annotation, restore the desired replica count
		if dep, ok := resource.(*appsv1.Deployment); ok {
			if dep.Annotations != nil {
				if desiredReplicas, exists := dep.Annotations[resources.BundleDesiredReplicasAnnotation]; exists {
					// Parse the desired replicas from the annotation
					if replicas, err := strconv.ParseInt(desiredReplicas, 10, 32); err == nil {
						dep.Spec.Replicas = ptr.To(int32(replicas))
					}
					// Remove the annotation from the exported deployment
					delete(dep.Annotations, resources.BundleDesiredReplicasAnnotation)
				}
			}
		}

		objects = append(objects, resource)
	}

	if len(objects) == 0 {
		return fmt.Errorf("no objects collected for bundle")
	}

	// Generate bundle data similar to the old client implementation
	bundleData, err := r.generateBundleData(objects)
	if err != nil {
		return fmt.Errorf("failed to generate bundle data: %w", err)
	}

	// Use reconciling framework to create/update the bundle Secret
	ownerRefWrapper := k8creconciling.OwnerRefWrapper(*metav1.NewControllerRef(bundle, operatorv1alpha1.SchemeGroupVersion.WithKind("Bundle")))

	secretReconcilers := []k8creconciling.NamedSecretReconcilerFactory{
		bundleSecretReconciler(bundle.Name, bundleData),
	}

	return k8creconciling.ReconcileSecrets(ctx, secretReconcilers, bundle.Namespace, r.Client, ownerRefWrapper)
}

// bundleSecretReconciler returns a reconciler for the bundle Secret
func bundleSecretReconciler(name string, data map[string][]byte) k8creconciling.NamedSecretReconcilerFactory {
	return func() (string, k8creconciling.SecretReconciler) {
		return name, func(secret *corev1.Secret) (*corev1.Secret, error) {
			secret.Data = data
			return secret, nil
		}
	}
}

// generateBundleData converts objects to bundle data format
func (r *BundleReconciler) generateBundleData(objects []ctrlruntimeclient.Object) (map[string][]byte, error) {
	data := make(map[string][]byte)

	for _, obj := range objects {
		key, err := r.generateObjectKey(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to generate key for object %s/%s: %w", obj.GetNamespace(), obj.GetName(), err)
		}

		// Clean object before marshaling (remove server-generated fields)
		objCopy := obj.DeepCopyObject().(ctrlruntimeclient.Object)
		objCopy.SetResourceVersion("")
		objCopy.SetUID("")
		objCopy.SetSelfLink("")
		objCopy.SetGeneration(0)
		objCopy.SetCreationTimestamp(metav1.Time{})
		objCopy.SetManagedFields(nil)
		objCopy.SetOwnerReferences(nil)

		// Marshal to JSON
		objData, err := json.Marshal(objCopy)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal object %s/%s: %w", obj.GetNamespace(), obj.GetName(), err)
		}

		data[key] = objData
	}

	return data, nil
}

// generateObjectKey generates a Kubernetes API-style path key for an object
// Format: api_v1_namespaces_<namespace>_<resource>_<name>
func (r *BundleReconciler) generateObjectKey(obj ctrlruntimeclient.Object) (string, error) {
	gvk, err := r.GroupVersionKindFor(obj)
	if err != nil {
		return "", err
	}

	restMapper := r.RESTMapper()
	mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return "", err
	}

	var pathBuilder strings.Builder

	// Build the base path
	if gvk.Group == "" {
		// Core API group
		pathBuilder.WriteString("api_")
		pathBuilder.WriteString(gvk.Version)
	} else {
		// Named API group
		pathBuilder.WriteString("apis_")
		pathBuilder.WriteString(strings.ReplaceAll(gvk.Group, ".", "_"))
		pathBuilder.WriteString("_")
		pathBuilder.WriteString(gvk.Version)
	}

	// Add namespace if object is namespaced
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		namespace := obj.GetNamespace()
		if namespace != "" {
			pathBuilder.WriteString("_namespaces_")
			pathBuilder.WriteString(strings.ReplaceAll(namespace, "-", "_"))
		}
	}

	// Add resource name
	pathBuilder.WriteString("_")
	pathBuilder.WriteString(mapping.Resource.Resource)

	// Add object name
	name := obj.GetName()
	if name != "" {
		pathBuilder.WriteString("_")
		pathBuilder.WriteString(strings.ReplaceAll(name, "-", "_"))
	}

	return pathBuilder.String(), nil
}

// updateStatus sets both phase and conditions on the reconciled Bundle object.
func (r *BundleReconciler) updateStatus(ctx context.Context, bundle *operatorv1alpha1.Bundle, oldStatus *operatorv1alpha1.BundleStatus, conditions []metav1.Condition, errs []error) error {
	for _, condition := range conditions {
		condition.ObservedGeneration = bundle.Generation
		bundle.Status.Conditions = updateCondition(bundle.Status.Conditions, condition)
	}

	// Set target name for display purposes
	bundle.Status.TargetName = bundle.Spec.Target.String()

	readyCond := meta.FindStatusCondition(bundle.Status.Conditions, string(operatorv1alpha1.ConditionTypeReady))
	switch {
	case readyCond != nil && readyCond.Status == metav1.ConditionTrue:
		bundle.Status.State = operatorv1alpha1.BundleStateReady

	case bundle.DeletionTimestamp != nil:
		bundle.Status.State = operatorv1alpha1.BundleStateDeleting

	case bundle.Status.State == "":
		bundle.Status.State = operatorv1alpha1.BundleStateProvisioning
	}

	// only patch the status if there are actual changes.
	if !equality.Semantic.DeepEqual(oldStatus, &bundle.Status) {
		if err := r.Client.Status().Update(ctx, bundle); err != nil {
			errs = append(errs, err)
		}
	}

	return kerrors.NewAggregate(errs)
}

// updateCondition updates or adds a condition in the conditions slice.
func updateCondition(conditions []metav1.Condition, newCondition metav1.Condition) []metav1.Condition {
	existingCondition := meta.FindStatusCondition(conditions, newCondition.Type)
	if existingCondition == nil {
		newCondition.LastTransitionTime = metav1.Now()
		return append(conditions, newCondition)
	}

	if existingCondition.Status != newCondition.Status {
		existingCondition.Status = newCondition.Status
		existingCondition.LastTransitionTime = metav1.Now()
	}

	existingCondition.Reason = newCondition.Reason
	existingCondition.Message = newCondition.Message
	existingCondition.ObservedGeneration = newCondition.ObservedGeneration

	return conditions
}
