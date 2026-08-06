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

package compiledvirtualworkspace

import (
	"context"
	"errors"
	"time"

	k8creconciling "k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/compiledvirtualworkspace"
	"github.com/kcp-dev/kcp-operator/pkg/controller/util"
	"github.com/kcp-dev/kcp-operator/pkg/metrics"
	"github.com/kcp-dev/kcp-operator/pkg/reconciling/modifier"
	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
)

// CompiledVirtualWorkspaceReconciler reconciles a CompiledVirtualWorkspace object
type CompiledVirtualWorkspaceReconciler struct {
	Client ctrlruntimeclient.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *CompiledVirtualWorkspaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// The rendered Deployment mounts Secrets owned by the source object, not by this one,
	// so an ownership watch would never retry a Deployment blocked on a missing mount.
	mountHandler := util.EnqueueAllInNamespace(mgr.GetClient(), func() ctrlruntimeclient.ObjectList {
		return &deployv1alpha1.CompiledVirtualWorkspaceList{}
	})

	return ctrl.NewControllerManagedBy(mgr).
		Named("compiled-virtualworkspace").
		For(&deployv1alpha1.CompiledVirtualWorkspace{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Watches(&corev1.Secret{}, mountHandler).
		Complete(r)
}

// +kubebuilder:rbac:groups=deploy.operator.kcp.io,resources=compiledvirtualworkspaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=deploy.operator.kcp.io,resources=compiledvirtualworkspaces/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=deploy.operator.kcp.io,resources=compiledvirtualworkspaces/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=core,resources=configmaps;secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *CompiledVirtualWorkspaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, recErr error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		metrics.RecordReconciliationMetrics(metrics.CompiledVirtualWorkspaceResourceType, duration.Seconds(), recErr)
	}()

	logger := log.FromContext(ctx)
	logger.V(4).Info("Reconciling")

	vw := &deployv1alpha1.CompiledVirtualWorkspace{}
	if err := r.Client.Get(ctx, req.NamespacedName, vw); err != nil {
		// object has been deleted.
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		metrics.RecordReconciliationError(metrics.CompiledVirtualWorkspaceResourceType, err.Error())
		return ctrl.Result{}, err
	}

	if vw.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	vwCopy := vw.DeepCopy()

	conditions, recErr := r.reconcile(ctx, r.Client, vwCopy)

	if err := r.reconcileStatus(ctx, r.Client, vw, vwCopy, conditions); err != nil {
		recErr = kerrors.NewAggregate([]error{recErr, err})
	}

	return ctrl.Result{}, recErr
}

//nolint:unparam // Keep the controller working the same as all the others, even though currently it does always return nil conditions.
func (r *CompiledVirtualWorkspaceReconciler) reconcile(ctx context.Context, client ctrlruntimeclient.Client, vw *deployv1alpha1.CompiledVirtualWorkspace) ([]metav1.Condition, error) {
	var (
		conditions []metav1.Condition
		errs       []error
	)

	ownerRefWrapper := k8creconciling.OwnerRefWrapper(*metav1.NewControllerRef(vw, deployv1alpha1.SchemeGroupVersion.WithKind("CompiledVirtualWorkspace")))
	revisionLabels := modifier.RelatedRevisionsLabels(ctx, client)

	if err := k8creconciling.ReconcileDeployments(ctx, []k8creconciling.NamedDeploymentReconcilerFactory{
		compiledvirtualworkspace.DeploymentReconciler(vw),
	}, vw.Namespace, client, ownerRefWrapper, revisionLabels); err != nil {
		// Swallow these errors and instead rely on us watching Secrets and re-reconciling whenever they change.
		if !errors.Is(err, modifier.ErrMountNotFound) {
			errs = append(errs, err)
		}
	}

	if err := k8creconciling.ReconcileServices(ctx, []k8creconciling.NamedServiceReconcilerFactory{
		compiledvirtualworkspace.ServiceReconciler(vw),
	}, vw.Namespace, client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	return conditions, kerrors.NewAggregate(errs)
}

func (r *CompiledVirtualWorkspaceReconciler) reconcileStatus(ctx context.Context, client ctrlruntimeclient.Client, oldVW *deployv1alpha1.CompiledVirtualWorkspace, vw *deployv1alpha1.CompiledVirtualWorkspace, conditions []metav1.Condition) error {
	// Check deployment status
	depKey := types.NamespacedName{Namespace: vw.Namespace, Name: resources.GetCompiledVirtualWorkspaceDeploymentName(vw)}
	cond, err := util.GetDeploymentAvailableCondition(ctx, client, depKey)
	if err != nil {
		return err
	}
	conditions = append(conditions, cond)

	for _, condition := range conditions {
		condition.ObservedGeneration = vw.Generation
		vw.Status.Conditions = util.UpdateCondition(vw.Status.Conditions, condition)
	}

	if !equality.Semantic.DeepEqual(oldVW.Status, vw.Status) {
		if err := client.Status().Patch(ctx, vw, ctrlruntimeclient.MergeFrom(oldVW)); err != nil {
			return err
		}
	}

	return nil
}
