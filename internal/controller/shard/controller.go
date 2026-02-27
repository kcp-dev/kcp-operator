/*
Copyright 2024 The kcp Authors.

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

package shard

import (
	"context"
	"fmt"
	"slices"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	kcpcorev1alpha1 "github.com/kcp-dev/kcp/sdk/apis/core/v1alpha1"
	"github.com/kcp-dev/logicalcluster/v3"
	k8creconciling "k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kcp-dev/kcp-operator/internal/client"
	bundlehelper "github.com/kcp-dev/kcp-operator/internal/controller/bundle"
	"github.com/kcp-dev/kcp-operator/internal/controller/util"
	"github.com/kcp-dev/kcp-operator/internal/metrics"
	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/shard"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

const cleanupFinalizer = "operator.kcp.io/cleanup-shard"

// ShardReconciler reconciles a Shard object
type ShardReconciler struct {
	ctrlruntimeclient.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *ShardReconciler) SetupWithManager(mgr ctrl.Manager) error {
	rootShardHandler := handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, obj ctrlruntimeclient.Object) []reconcile.Request {
		rootShard := obj.(*operatorv1alpha1.RootShard)

		var shards operatorv1alpha1.ShardList
		if err := mgr.GetClient().List(ctx, &shards, &ctrlruntimeclient.ListOptions{Namespace: rootShard.Namespace}); err != nil {
			utilruntime.HandleError(err)
			return nil
		}

		var requests []reconcile.Request
		for _, shard := range shards.Items {
			if ref := shard.Spec.RootShard.Reference; ref != nil && ref.Name == rootShard.Name {
				requests = append(requests, reconcile.Request{NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(&shard)})
			}
		}

		return requests
	})

	return ctrl.NewControllerManagedBy(mgr).
		Named("shard").
		For(&operatorv1alpha1.Shard{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.Service{}).
		Owns(&certmanagerv1.Certificate{}).
		Watches(&operatorv1alpha1.RootShard{}, rootShardHandler).
		Complete(r)
}

// +kubebuilder:rbac:groups=operator.kcp.io,resources=shards,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=shards/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=shards/finalizers,verbs=update
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets;services,verbs=get;list;watch;create;update;patch;delete

func (r *ShardReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, recErr error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		metrics.RecordReconciliationMetrics(metrics.ShardResourceType, duration.Seconds(), recErr)
	}()

	logger := log.FromContext(ctx)
	logger.V(4).Info("Reconciling Shard object")

	var s operatorv1alpha1.Shard
	if err := r.Get(ctx, req.NamespacedName, &s); err != nil {
		if ctrlruntimeclient.IgnoreNotFound(err) != nil {
			metrics.RecordReconciliationError(metrics.ShardResourceType, err.Error())
			return ctrl.Result{}, fmt.Errorf("failed to get shard: %w", err)
		}

		return ctrl.Result{}, nil
	}

	conditions, recErr := r.reconcile(ctx, &s)

	if err := r.reconcileStatus(ctx, &s, conditions); err != nil {
		recErr = kerrors.NewAggregate([]error{recErr, err})
	}

	return ctrl.Result{}, recErr
}

func (r *ShardReconciler) reconcile(ctx context.Context, s *operatorv1alpha1.Shard) ([]metav1.Condition, error) {
	var (
		errs       []error
		conditions []metav1.Condition
	)

	if s.DeletionTimestamp != nil {
		return r.handleDeletion(ctx, s)
	}

	// Ensure finalizer before any other work
	if updated, err := r.ensureFinalizer(ctx, s); err != nil {
		return conditions, fmt.Errorf("failed to ensure cleanup finalizer: %w", err)
	} else if updated {
		return conditions, nil // Will be requeued
	}

	// Ensure Bundle object exists if annotation is present
	if _, err := bundlehelper.EnsureBundleForOwner(ctx, r.Client, r.Scheme, s); err != nil {
		errs = append(errs, fmt.Errorf("failed to ensure bundle: %w", err))
	}

	cond, rootShard := util.FetchRootShard(ctx, r.Client, s.Namespace, s.Spec.RootShard.Reference)
	conditions = append(conditions, cond)

	if rootShard == nil {
		return conditions, nil
	}

	ownerRefWrapper := k8creconciling.OwnerRefWrapper(*metav1.NewControllerRef(s, operatorv1alpha1.SchemeGroupVersion.WithKind("Shard")))

	certReconcilers := []reconciling.NamedCertificateReconcilerFactory{
		shard.ServerCertificateReconciler(s, rootShard),
		shard.ServiceAccountCertificateReconciler(s, rootShard),
		shard.VirtualWorkspacesCertificateReconciler(s, rootShard),
		shard.RootShardClientCertificateReconciler(s, rootShard),
		shard.LogicalClusterAdminCertificateReconciler(s, rootShard),
		shard.ExternalLogicalClusterAdminCertificateReconciler(s, rootShard),
	}

	if err := reconciling.ReconcileCertificates(ctx, certReconcilers, s.Namespace, r.Client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	if err := k8creconciling.ReconcileSecrets(ctx, []k8creconciling.NamedSecretReconcilerFactory{
		shard.RootShardClientKubeconfigReconciler(s, rootShard),
		shard.LogicalClusterAdminKubeconfigReconciler(s, rootShard),
		shard.ExternalLogicalClusterAdminKubeconfigReconciler(s, rootShard),
	}, s.Namespace, r.Client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	if s.Spec.CABundleSecretRef != nil {
		if err := k8creconciling.ReconcileSecrets(ctx, []k8creconciling.NamedSecretReconcilerFactory{
			shard.MergedCABundleSecretReconciler(ctx, s, r.Client),
		}, s.Namespace, r.Client, ownerRefWrapper); err != nil {
			errs = append(errs, err)
		}
	}

	// Deployment will be scaled to 0 if bundle annotation is present
	if err := k8creconciling.ReconcileDeployments(ctx, []k8creconciling.NamedDeploymentReconcilerFactory{
		shard.DeploymentReconciler(s, rootShard),
	}, s.Namespace, r.Client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	if err := k8creconciling.ReconcileServices(ctx, []k8creconciling.NamedServiceReconcilerFactory{
		shard.ServiceReconciler(s),
	}, s.Namespace, r.Client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	return conditions, kerrors.NewAggregate(errs)
}

// reconcileStatus sets both phase and conditions on the reconciled Shard object.
func (r *ShardReconciler) reconcileStatus(ctx context.Context, oldShard *operatorv1alpha1.Shard, conditions []metav1.Condition) error {
	newShard := oldShard.DeepCopy()
	var errs []error

	// Add Bundle condition
	bundleCond := bundlehelper.GetBundleReadyCondition(ctx, r.Client, newShard, newShard.Generation)
	conditions = append(conditions, bundleCond)

	// Check if shard is bundled (has bundle annotation with Ready bundle)
	isBundled := bundleCond.Status == metav1.ConditionTrue && bundleCond.Reason == "BundleReady"

	// Only check deployment status if not bundled
	if !isBundled {
		depKey := types.NamespacedName{Namespace: newShard.Namespace, Name: resources.GetShardDeploymentName(newShard)}
		cond, err := util.GetDeploymentAvailableCondition(ctx, r.Client, depKey)
		if err != nil {
			errs = append(errs, err)
		} else {
			conditions = append(conditions, cond)
		}
	}

	for _, condition := range conditions {
		condition.ObservedGeneration = newShard.Generation
		newShard.Status.Conditions = util.UpdateCondition(newShard.Status.Conditions, condition)
	}

	availableCond := apimeta.FindStatusCondition(newShard.Status.Conditions, string(operatorv1alpha1.ConditionTypeAvailable))
	bundleStatusCond := apimeta.FindStatusCondition(newShard.Status.Conditions, string(operatorv1alpha1.ConditionTypeBundle))

	switch {
	case availableCond != nil && availableCond.Status == metav1.ConditionTrue:
		newShard.Status.Phase = operatorv1alpha1.ShardPhaseRunning

	case newShard.DeletionTimestamp != nil:
		newShard.Status.Phase = operatorv1alpha1.ShardPhaseDeleting

	case isBundled:
		// Shard is bundled, deployment scaled to 0, resources exported via bundle
		newShard.Status.Phase = operatorv1alpha1.ShardPhaseBundled

	case bundleStatusCond != nil && bundleStatusCond.Status != metav1.ConditionTrue:
		newShard.Status.Phase = operatorv1alpha1.ShardPhaseProvisioning

	case availableCond != nil && availableCond.Status == metav1.ConditionTrue:
		newShard.Status.Phase = operatorv1alpha1.ShardPhaseRunning

	case newShard.Status.Phase == "":
		newShard.Status.Phase = operatorv1alpha1.ShardPhaseProvisioning
	}

	// only patch the status if there are actual changes.
	if !equality.Semantic.DeepEqual(oldShard.Status, newShard.Status) {
		if err := r.Client.Status().Patch(ctx, newShard, ctrlruntimeclient.MergeFrom(oldShard)); err != nil {
			errs = append(errs, err)
		}
	}

	return kerrors.NewAggregate(errs)
}

func (r *ShardReconciler) handleDeletion(ctx context.Context, s *operatorv1alpha1.Shard) ([]metav1.Condition, error) {
	logger := log.FromContext(ctx)

	if !slices.Contains(s.Finalizers, cleanupFinalizer) {
		return nil, nil
	}

	// Fetch RootShard
	cond, rootShard := util.FetchRootShard(ctx, r.Client, s.Namespace, s.Spec.RootShard.Reference)
	if rootShard == nil {
		logger.Info("RootShard not found, cannot clean up kcp Shard object", "condition", cond.Message)
		// Remove finalizer anyway - we can't clean up without the root shard
		if err := r.removeFinalizer(ctx, s); err != nil {
			return []metav1.Condition{cond}, fmt.Errorf("failed to remove finalizer: %w", err)
		}
		return []metav1.Condition{cond}, nil
	}

	// Create client to root shard
	kcpClient, err := client.NewRootShardClient(ctx, r.Client, rootShard, logicalcluster.Name("root"), r.Scheme)
	if err != nil {
		return nil, fmt.Errorf("failed to create root shard client: %w", err)
	}

	// Delete the kcp Shard object
	kcpShard := &kcpcorev1alpha1.Shard{}
	kcpShard.Name = s.Name

	logger.Info("Deleting kcp Shard object from root workspace", "name", s.Name)
	if err := kcpClient.Delete(ctx, kcpShard); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to delete kcp Shard: %w", err)
		}
		logger.V(2).Info("kcp Shard object already deleted")
	}

	// Remove finalizer
	if err := r.removeFinalizer(ctx, s); err != nil {
		return nil, fmt.Errorf("failed to remove finalizer: %w", err)
	}

	return nil, nil
}

func (r *ShardReconciler) ensureFinalizer(ctx context.Context, s *operatorv1alpha1.Shard) (bool, error) {
	finalizers := sets.New(s.GetFinalizers()...)
	if finalizers.Has(cleanupFinalizer) {
		return false, nil
	}

	original := s.DeepCopy()
	finalizers.Insert(cleanupFinalizer)
	s.SetFinalizers(sets.List(finalizers))

	if err := r.Patch(ctx, s, ctrlruntimeclient.MergeFrom(original)); err != nil {
		return false, err
	}

	return true, nil
}

func (r *ShardReconciler) removeFinalizer(ctx context.Context, s *operatorv1alpha1.Shard) error {
	finalizers := sets.New(s.GetFinalizers()...)
	if !finalizers.Has(cleanupFinalizer) {
		return nil
	}

	original := s.DeepCopy()
	finalizers.Delete(cleanupFinalizer)
	s.SetFinalizers(sets.List(finalizers))

	return r.Patch(ctx, s, ctrlruntimeclient.MergeFrom(original))
}
