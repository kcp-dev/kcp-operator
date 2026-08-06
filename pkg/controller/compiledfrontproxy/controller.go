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

package compiledfrontproxy

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/log"
	mcbuilder "sigs.k8s.io/multicluster-runtime/pkg/builder"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
	"sigs.k8s.io/multicluster-runtime/pkg/multicluster"
	mcreconcile "sigs.k8s.io/multicluster-runtime/pkg/reconcile"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/compiledfrontproxy"
	"github.com/kcp-dev/kcp-operator/pkg/controller/util"
	"github.com/kcp-dev/kcp-operator/pkg/metrics"
	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// CompiledFrontProxyReconciler reconciles a CompiledFrontProxy object
type CompiledFrontProxyReconciler struct {
	GetCluster func(ctx context.Context, clusterName multicluster.ClusterName) (cluster.Cluster, error)
}

// SetupWithManager sets up the controller with the Manager.
func (r *CompiledFrontProxyReconciler) SetupWithManager(mgr mcmanager.Manager, opts ...mcbuilder.EngageOptions) error {
	// The rendered Deployment mounts Secrets owned by the source object, not by this one,
	// so an ownership watch would never retry a Deployment blocked on a missing mount.
	mountHandler := util.EnqueueAllInNamespace(func() ctrlruntimeclient.ObjectList {
		return &deployv1alpha1.CompiledFrontProxyList{}
	})

	return mcbuilder.ControllerManagedBy(mgr).
		Named("compiled-frontproxy").
		For(&deployv1alpha1.CompiledFrontProxy{}, util.EngageFor(opts)...).
		Owns(&appsv1.Deployment{}, util.EngageOwns(opts)...).
		Owns(&corev1.ConfigMap{}, util.EngageOwns(opts)...).
		Owns(&corev1.Service{}, util.EngageOwns(opts)...).
		Watches(&corev1.Secret{}, mountHandler, util.EngageWatches(opts)...).
		Complete(r)
}

// +kubebuilder:rbac:groups=deploy.operator.kcp.io,resources=compiledfrontproxies,verbs=get;list;watch
// +kubebuilder:rbac:groups=deploy.operator.kcp.io,resources=compiledfrontproxies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=deploy.operator.kcp.io,resources=compiledfrontproxies/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=core,resources=services;configmaps;secrets,verbs=get;list;watch;create;update;patch

func (r *CompiledFrontProxyReconciler) Reconcile(ctx context.Context, req mcreconcile.Request) (res ctrl.Result, recErr error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		metrics.RecordReconciliationMetrics(metrics.CompiledFrontProxyResourceType, duration.Seconds(), recErr)
	}()

	logger := log.FromContext(ctx).WithValues("cluster", req.ClusterName)
	logger.V(4).Info("Reconciling")

	cl, err := r.GetCluster(ctx, req.ClusterName)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get cluster %q: %w", req.ClusterName, err)
	}

	var frontProxy deployv1alpha1.CompiledFrontProxy
	if err := cl.GetClient().Get(ctx, req.NamespacedName, &frontProxy); err != nil {
		if ctrlruntimeclient.IgnoreNotFound(err) != nil {
			metrics.RecordReconciliationError(metrics.CompiledFrontProxyResourceType, err.Error())
			return ctrl.Result{}, fmt.Errorf("failed to get CompiledFrontProxy object: %w", err)
		}

		// Object has apparently been deleted already.
		return ctrl.Result{}, nil
	}

	conditions, recErr := r.reconcile(ctx, cl.GetClient(), &frontProxy)

	if err := r.reconcileStatus(ctx, cl.GetClient(), &frontProxy, conditions); err != nil {
		recErr = kerrors.NewAggregate([]error{recErr, err})
	}

	return ctrl.Result{}, recErr
}

//nolint:unparam // Keep the controller working the same as all the others, even though currently it does always return nil conditions.
func (r *CompiledFrontProxyReconciler) reconcile(ctx context.Context, client ctrlruntimeclient.Client, frontProxy *deployv1alpha1.CompiledFrontProxy) ([]metav1.Condition, error) {
	var (
		conditions []metav1.Condition
		errs       []error
	)

	if frontProxy.DeletionTimestamp != nil {
		return conditions, nil
	}

	if err := compiledfrontproxy.NewFrontProxy(frontProxy).Reconcile(ctx, client, frontProxy.Namespace); err != nil {
		errs = append(errs, fmt.Errorf("failed to reconcile: %w", err))
	}

	return conditions, kerrors.NewAggregate(errs)
}

func (r *CompiledFrontProxyReconciler) reconcileStatus(ctx context.Context, client ctrlruntimeclient.Client, oldFrontProxy *deployv1alpha1.CompiledFrontProxy, conditions []metav1.Condition) error {
	frontProxy := oldFrontProxy.DeepCopy()
	var errs []error

	depKey := types.NamespacedName{Namespace: frontProxy.Namespace, Name: resources.GetCompiledFrontProxyDeploymentName(frontProxy)}
	cond, err := util.GetDeploymentAvailableCondition(ctx, client, depKey)
	if err != nil {
		errs = append(errs, err)
	} else {
		conditions = append(conditions, cond)
	}

	for _, condition := range conditions {
		condition.ObservedGeneration = frontProxy.Generation
		frontProxy.Status.Conditions = util.UpdateCondition(frontProxy.Status.Conditions, condition)
	}

	if frontProxy.DeletionTimestamp != nil {
		frontProxy.Status.Phase = operatorv1alpha1.FrontProxyPhaseDeleting
	} else {
		availableCond := apimeta.FindStatusCondition(frontProxy.Status.Conditions, string(operatorv1alpha1.ConditionTypeAvailable))

		if availableCond != nil && availableCond.Status == metav1.ConditionTrue {
			frontProxy.Status.Phase = operatorv1alpha1.FrontProxyPhaseRunning
		} else {
			frontProxy.Status.Phase = operatorv1alpha1.FrontProxyPhaseProvisioning
		}
	}

	// only patch the status if there are actual changes.
	if !equality.Semantic.DeepEqual(oldFrontProxy.Status, frontProxy.Status) {
		if err := client.Status().Patch(ctx, frontProxy, ctrlruntimeclient.MergeFrom(oldFrontProxy)); err != nil {
			errs = append(errs, err)
		}
	}

	return kerrors.NewAggregate(errs)
}
