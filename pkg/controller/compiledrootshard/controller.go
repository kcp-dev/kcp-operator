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

package compiledrootshard

import (
	"context"
	"errors"
	"fmt"
	"time"

	k8creconciling "k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/compiledfrontproxy"
	"github.com/kcp-dev/kcp-operator/internal/resources/compiledrootshard"
	"github.com/kcp-dev/kcp-operator/pkg/controller/util"
	"github.com/kcp-dev/kcp-operator/pkg/metrics"
	"github.com/kcp-dev/kcp-operator/pkg/reconciling/modifier"
	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// CompiledRootShardReconciler reconciles a CompiledRootShard object
type CompiledRootShardReconciler struct {
	Client ctrlruntimeclient.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *CompiledRootShardReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// The rendered Deployment mounts Secrets owned by the source object, not by this one,
	// so an ownership watch would never retry a Deployment blocked on a missing mount.
	mountHandler := util.EnqueueAllInNamespace(mgr.GetClient(), func() ctrlruntimeclient.ObjectList {
		return &deployv1alpha1.CompiledRootShardList{}
	})

	return ctrl.NewControllerManagedBy(mgr).
		Named("compiled-rootshard").
		For(&deployv1alpha1.CompiledRootShard{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Watches(&corev1.Secret{}, mountHandler).
		Complete(r)
}

// +kubebuilder:rbac:groups=deploy.operator.kcp.io,resources=compiledrootshards,verbs=get;list;watch
// +kubebuilder:rbac:groups=deploy.operator.kcp.io,resources=compiledrootshards/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=deploy.operator.kcp.io,resources=compiledrootshards/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=core,resources=configmaps;secrets,verbs=get;list;watch;create;update;patch

func (r *CompiledRootShardReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, recErr error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		metrics.RecordReconciliationMetrics(metrics.CompiledRootShardResourceType, duration.Seconds(), recErr)
	}()

	logger := log.FromContext(ctx)
	logger.V(4).Info("Reconciling")

	var rootShard deployv1alpha1.CompiledRootShard
	if err := r.Client.Get(ctx, req.NamespacedName, &rootShard); err != nil {
		if ctrlruntimeclient.IgnoreNotFound(err) != nil {
			metrics.RecordReconciliationError(metrics.CompiledRootShardResourceType, err.Error())
			return ctrl.Result{}, fmt.Errorf("failed to find %s/%s: %w", req.Namespace, req.Name, err)
		}

		// Object has apparently been deleted already.
		return ctrl.Result{}, nil
	}

	conditions, recErr := r.reconcile(ctx, &rootShard)

	if err := r.reconcileStatus(ctx, &rootShard, conditions); err != nil {
		recErr = kerrors.NewAggregate([]error{recErr, err})
	}

	return ctrl.Result{}, recErr
}

//nolint:unparam // Keep the controller working the same as all the others, even though currently it does always return nil conditions.
func (r *CompiledRootShardReconciler) reconcile(ctx context.Context, rootShard *deployv1alpha1.CompiledRootShard) ([]metav1.Condition, error) {
	var (
		errs       []error
		conditions []metav1.Condition
	)

	if rootShard.DeletionTimestamp != nil {
		return conditions, nil
	}

	ownerRefWrapper := k8creconciling.OwnerRefWrapper(*metav1.NewControllerRef(rootShard, deployv1alpha1.SchemeGroupVersion.WithKind("CompiledRootShard")))
	revisionLabels := modifier.RelatedRevisionsLabels(ctx, r.Client)

	if err := k8creconciling.ReconcileDeployments(ctx, []k8creconciling.NamedDeploymentReconcilerFactory{
		compiledrootshard.DeploymentReconciler(rootShard),
	}, rootShard.Namespace, r.Client, ownerRefWrapper, revisionLabels); err != nil {
		// Swallow these errors and instead rely on us watching Secrets and re-reconciling whenever they change.
		if !errors.Is(err, modifier.ErrMountNotFound) {
			errs = append(errs, err)
		}
	}

	if err := k8creconciling.ReconcileServices(ctx, []k8creconciling.NamedServiceReconcilerFactory{
		compiledrootshard.ServiceReconciler(rootShard),
	}, rootShard.Namespace, r.Client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	if err := compiledfrontproxy.NewRootShardProxy(rootShard).Reconcile(ctx, r.Client, rootShard.Namespace); err != nil {
		errs = append(errs, fmt.Errorf("failed to reconcile proxy: %w", err))
	}

	return conditions, kerrors.NewAggregate(errs)
}

// reconcileStatus sets both phase and conditions on the reconciled CompiledRootShard object.
func (r *CompiledRootShardReconciler) reconcileStatus(ctx context.Context, oldRootShard *deployv1alpha1.CompiledRootShard, conditions []metav1.Condition) error {
	rootShard := oldRootShard.DeepCopy()
	var errs []error

	depKey := types.NamespacedName{Namespace: rootShard.Namespace, Name: resources.GetCompiledRootShardDeploymentName(rootShard)}
	cond, err := util.GetDeploymentAvailableCondition(ctx, r.Client, depKey)
	if err != nil {
		errs = append(errs, err)
	} else {
		conditions = append(conditions, cond)
	}

	for _, condition := range conditions {
		condition.ObservedGeneration = rootShard.Generation
		rootShard.Status.Conditions = util.UpdateCondition(rootShard.Status.Conditions, condition)
	}

	if rootShard.DeletionTimestamp != nil {
		rootShard.Status.Phase = operatorv1alpha1.RootShardPhaseDeleting
	} else {
		availableCond := apimeta.FindStatusCondition(rootShard.Status.Conditions, string(operatorv1alpha1.ConditionTypeAvailable))

		if availableCond != nil && availableCond.Status == metav1.ConditionTrue {
			rootShard.Status.Phase = operatorv1alpha1.RootShardPhaseRunning
		} else {
			rootShard.Status.Phase = operatorv1alpha1.RootShardPhaseProvisioning
		}
	}

	if !equality.Semantic.DeepEqual(oldRootShard.Status, rootShard.Status) {
		if err := r.Client.Status().Patch(ctx, rootShard, ctrlruntimeclient.MergeFrom(oldRootShard)); err != nil {
			errs = append(errs, err)
		}
	}

	return kerrors.NewAggregate(errs)
}
