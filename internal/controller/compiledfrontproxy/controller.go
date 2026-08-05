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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kcp-dev/kcp-operator/internal/controller/util"
	"github.com/kcp-dev/kcp-operator/internal/metrics"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/compiledfrontproxy"
	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// CompiledFrontProxyReconciler reconciles a CompiledFrontProxy object
type CompiledFrontProxyReconciler struct {
	ctrlruntimeclient.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *CompiledFrontProxyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("compiled-frontproxy").
		For(&deployv1alpha1.CompiledFrontProxy{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=deploy.operator.kcp.io,resources=compiledfrontproxies,verbs=get;list;watch
// +kubebuilder:rbac:groups=deploy.operator.kcp.io,resources=compiledfrontproxies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=deploy.operator.kcp.io,resources=compiledfrontproxies/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=core,resources=services;configmaps;secrets,verbs=get;list;watch;create;update;patch

func (r *CompiledFrontProxyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, recErr error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		metrics.RecordReconciliationMetrics(metrics.CompiledFrontProxyResourceType, duration.Seconds(), recErr)
	}()

	logger := log.FromContext(ctx)
	logger.V(4).Info("Reconciling")

	var frontProxy deployv1alpha1.CompiledFrontProxy
	if err := r.Get(ctx, req.NamespacedName, &frontProxy); err != nil {
		if ctrlruntimeclient.IgnoreNotFound(err) != nil {
			metrics.RecordReconciliationError(metrics.CompiledFrontProxyResourceType, err.Error())
			return ctrl.Result{}, fmt.Errorf("failed to get CompiledFrontProxy object: %w", err)
		}

		// Object has apparently been deleted already.
		return ctrl.Result{}, nil
	}

	conditions, recErr := r.reconcile(ctx, &frontProxy)

	if err := r.reconcileStatus(ctx, &frontProxy, conditions); err != nil {
		recErr = kerrors.NewAggregate([]error{recErr, err})
	}

	return ctrl.Result{}, recErr
}

//nolint:unparam // Keep the controller working the same as all the others, even though currently it does always return nil conditions.
func (r *CompiledFrontProxyReconciler) reconcile(ctx context.Context, frontProxy *deployv1alpha1.CompiledFrontProxy) ([]metav1.Condition, error) {
	var (
		conditions []metav1.Condition
		errs       []error
	)

	if frontProxy.DeletionTimestamp != nil {
		return conditions, nil
	}

	if err := compiledfrontproxy.NewFrontProxy(frontProxy).Reconcile(ctx, r.Client, frontProxy.Namespace); err != nil {
		errs = append(errs, fmt.Errorf("failed to reconcile: %w", err))
	}

	return conditions, kerrors.NewAggregate(errs)
}

func (r *CompiledFrontProxyReconciler) reconcileStatus(ctx context.Context, oldFrontProxy *deployv1alpha1.CompiledFrontProxy, conditions []metav1.Condition) error {
	frontProxy := oldFrontProxy.DeepCopy()
	var errs []error

	depKey := types.NamespacedName{Namespace: frontProxy.Namespace, Name: resources.GetCompiledFrontProxyDeploymentName(frontProxy)}
	cond, err := util.GetDeploymentAvailableCondition(ctx, r.Client, depKey)
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
		if err := r.Status().Patch(ctx, frontProxy, ctrlruntimeclient.MergeFrom(oldFrontProxy)); err != nil {
			errs = append(errs, err)
		}
	}

	return kerrors.NewAggregate(errs)
}
