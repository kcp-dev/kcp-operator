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

package compiledshard

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
	"github.com/kcp-dev/kcp-operator/internal/resources/compiledshard"
	"github.com/kcp-dev/kcp-operator/pkg/controller/util"
	"github.com/kcp-dev/kcp-operator/pkg/metrics"
	"github.com/kcp-dev/kcp-operator/pkg/reconciling/modifier"
	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// CompiledShardReconciler reconciles a CompiledShard object
type CompiledShardReconciler struct {
	ctrlruntimeclient.Client
	Scheme *runtime.Scheme
}

func (r *CompiledShardReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// The rendered Deployment mounts Secrets owned by the source object, not by this one,
	// so an ownership watch would never retry a Deployment blocked on a missing mount.
	mountHandler := util.EnqueueAllInNamespace(mgr.GetClient(), func() ctrlruntimeclient.ObjectList {
		return &deployv1alpha1.CompiledShardList{}
	})

	return ctrl.NewControllerManagedBy(mgr).
		Named("compiled-shard").
		For(&deployv1alpha1.CompiledShard{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Watches(&corev1.Secret{}, mountHandler).
		Complete(r)
}

// +kubebuilder:rbac:groups=deploy.operator.kcp.io,resources=compiledshards,verbs=get;list;watch
// +kubebuilder:rbac:groups=deploy.operator.kcp.io,resources=compiledshards/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=deploy.operator.kcp.io,resources=compiledshards/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=core,resources=configmaps;secrets,verbs=get;list;watch

func (r *CompiledShardReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, recErr error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		metrics.RecordReconciliationMetrics(metrics.CompiledShardResourceType, duration.Seconds(), recErr)
	}()

	logger := log.FromContext(ctx)
	logger.V(4).Info("Reconciling CompiledShard object")

	var s deployv1alpha1.CompiledShard
	if err := r.Get(ctx, req.NamespacedName, &s); err != nil {
		if ctrlruntimeclient.IgnoreNotFound(err) != nil {
			metrics.RecordReconciliationError(metrics.CompiledShardResourceType, err.Error())
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

//nolint:unparam // Keep the controller working the same as all the others, even though currently it does always return nil conditions.
func (r *CompiledShardReconciler) reconcile(ctx context.Context, s *deployv1alpha1.CompiledShard) ([]metav1.Condition, error) {
	var (
		errs       []error
		conditions []metav1.Condition
	)

	if s.DeletionTimestamp != nil {
		return conditions, nil
	}

	ownerRefWrapper := k8creconciling.OwnerRefWrapper(*metav1.NewControllerRef(s, deployv1alpha1.SchemeGroupVersion.WithKind("CompiledShard")))
	revisionLabels := modifier.RelatedRevisionsLabels(ctx, r.Client)

	if err := k8creconciling.ReconcileDeployments(ctx, []k8creconciling.NamedDeploymentReconcilerFactory{
		compiledshard.DeploymentReconciler(s),
	}, s.Namespace, r.Client, ownerRefWrapper, revisionLabels); err != nil {
		// Swallow these errors and instead rely on us watching Secrets and re-reconciling whenever they change.
		if !errors.Is(err, modifier.ErrMountNotFound) {
			errs = append(errs, err)
		}
	}

	if err := k8creconciling.ReconcileServices(ctx, []k8creconciling.NamedServiceReconcilerFactory{
		compiledshard.ServiceReconciler(s),
	}, s.Namespace, r.Client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	return conditions, kerrors.NewAggregate(errs)
}

func (r *CompiledShardReconciler) reconcileStatus(ctx context.Context, oldShard *deployv1alpha1.CompiledShard, conditions []metav1.Condition) error {
	newShard := oldShard.DeepCopy()
	var errs []error

	depKey := types.NamespacedName{Namespace: newShard.Namespace, Name: resources.GetCompiledShardDeploymentName(newShard)}
	cond, err := util.GetDeploymentAvailableCondition(ctx, r.Client, depKey)
	if err != nil {
		errs = append(errs, err)
	} else {
		conditions = append(conditions, cond)
	}

	for _, condition := range conditions {
		condition.ObservedGeneration = newShard.Generation
		newShard.Status.Conditions = util.UpdateCondition(newShard.Status.Conditions, condition)
	}

	availableCond := apimeta.FindStatusCondition(newShard.Status.Conditions, string(operatorv1alpha1.ConditionTypeAvailable))

	switch {
	case availableCond != nil && availableCond.Status == metav1.ConditionTrue:
		newShard.Status.Phase = operatorv1alpha1.ShardPhaseRunning

	case newShard.DeletionTimestamp != nil:
		newShard.Status.Phase = operatorv1alpha1.ShardPhaseDeleting

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
